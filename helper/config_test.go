package helper

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/shabarkin/aws-enumerator/servicemaster"
	"github.com/shabarkin/aws-enumerator/servicestructs"
)

// Keep fixtures isolated from the caller's AWS credentials and metadata services.
// These tests change the process environment and working directory, so run serially.
func awsTestEnvironment(t *testing.T) string {
	t.Helper()
	for _, name := range []string{
		"AWS_PROFILE", "AWS_DEFAULT_PROFILE", "AWS_ACCESS_KEY_ID", "AWS_ACCESS_KEY",
		"AWS_SECRET_ACCESS_KEY", "AWS_SECRET_KEY", "AWS_SESSION_TOKEN", "AWS_REGION", "AWS_DEFAULT_REGION",
		"AWS_ROLE_ARN", "AWS_ROLE_SESSION_NAME", "AWS_WEB_IDENTITY_TOKEN_FILE",
		"AWS_CONTAINER_CREDENTIALS_RELATIVE_URI", "AWS_CONTAINER_CREDENTIALS_FULL_URI",
		"AWS_CONFIG_FILE", "AWS_SHARED_CREDENTIALS_FILE", "AWS_EC2_METADATA_DISABLED",
	} {
		value, present := os.LookupEnv(name)
		key := name
		t.Cleanup(func() {
			if present {
				os.Setenv(key, value)
			} else {
				os.Unsetenv(key)
			}
		})
		os.Unsetenv(name)
	}
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(cwd) })
	os.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	os.Setenv("AWS_CONFIG_FILE", filepath.Join(dir, "config"))
	os.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(dir, "credentials"))
	writeFixture(t, "config", "[default]\nregion = us-east-1\n[profile dev]\nregion = us-west-2\n[profile other]\nregion = eu-west-1\n")
	writeFixture(t, "credentials", "[default]\naws_access_key_id = DEFAULTKEY\naws_secret_access_key = default-secret\n[dev]\naws_access_key_id = DEVKEY\naws_secret_access_key = dev-secret\naws_session_token = dev-token\n[other]\naws_access_key_id = OTHERKEY\naws_secret_access_key = other-secret\n")
	return dir
}

func writeFixture(t *testing.T, name, contents string) {
	t.Helper()
	if err := ioutil.WriteFile(name, []byte(contents), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestAWSConfigSources(t *testing.T) {
	for _, tc := range []struct {
		name, profile, envProfile, envKey, envRegion, dotenv, key, region, token string
	}{
		{name: "explicit profile without dotenv", profile: "dev", key: "DEVKEY", region: "us-west-2", token: "dev-token"},
		{name: "explicit profile beats env profile and keys", profile: "dev", envProfile: "other", envKey: "ENVKEY", key: "DEVKEY", region: "us-west-2", token: "dev-token"},
		{name: "explicit profile ignores conflicting dotenv", profile: "dev", dotenv: "AWS_ACCESS_KEY_ID=OLDKEY\nAWS_SECRET_ACCESS_KEY=old-secret\nAWS_REGION=ap-south-1\n", key: "DEVKEY", region: "us-west-2", token: "dev-token"},
		{name: "env profile ignores malformed dotenv", envProfile: "dev", dotenv: "invalid .env !", key: "DEVKEY", region: "us-west-2", token: "dev-token"},
		{name: "env credentials retain SDK precedence", envProfile: "dev", envKey: "ENVKEY", key: "ENVKEY", region: "us-west-2"},
		{name: "region environment overrides profile", profile: "dev", envRegion: "eu-central-1", key: "DEVKEY", region: "eu-central-1", token: "dev-token"},
		{name: "default profile without dotenv", key: "DEFAULTKEY", region: "us-east-1"},
		{name: "legacy dotenv", dotenv: "AWS_ACCESS_KEY_ID=OLDKEY\nAWS_SECRET_ACCESS_KEY=old-secret\nAWS_SESSION_TOKEN=old-token\nAWS_REGION=ap-south-1\n", key: "OLDKEY", region: "ap-south-1", token: "old-token"},
		{name: "shell credentials beat dotenv", envKey: "ENVKEY", dotenv: "AWS_ACCESS_KEY_ID=OLDKEY\nAWS_SECRET_ACCESS_KEY=old-secret\n", key: "ENVKEY", region: "us-east-1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			awsTestEnvironment(t)
			os.Setenv("AWS_PROFILE", tc.envProfile)
			if tc.envRegion != "" {
				os.Setenv("AWS_REGION", tc.envRegion)
			}
			if tc.envKey != "" {
				os.Setenv("AWS_ACCESS_KEY_ID", tc.envKey)
				os.Setenv("AWS_SECRET_ACCESS_KEY", "env-secret")
			}
			if tc.dotenv != "" {
				writeFixture(t, ".env", tc.dotenv)
			}
			cfg, err := LoadAWSConfig(context.Background(), tc.profile)
			if err != nil {
				t.Fatal(err)
			}
			creds, err := cfg.Credentials.Retrieve(context.Background())
			if err != nil || creds.AccessKeyID != tc.key || creds.SessionToken != tc.token || cfg.Region != tc.region {
				t.Fatalf("unexpected credential source or region: key=%q token=%q region=%q err=%v", creds.AccessKeyID, creds.SessionToken, cfg.Region, err)
			}
		})
	}
}

func TestAWSConfigErrors(t *testing.T) {
	for _, kind := range []string{"missing profile", "malformed dotenv", "missing region"} {
		t.Run(kind, func(t *testing.T) {
			awsTestEnvironment(t)
			profile := ""
			switch kind {
			case "missing profile":
				profile = "missing"
			case "malformed dotenv":
				writeFixture(t, ".env", "invalid .env !")
			case "missing region":
				writeFixture(t, "config", "")
			}
			if _, err := LoadAWSConfig(context.Background(), profile); err == nil {
				t.Fatal("expected configuration error")
			}
		})
	}
}

func TestSelectedConfigUsedForSTSAndServiceClients(t *testing.T) {
	awsTestEnvironment(t)
	cfg, err := LoadAWSConfig(context.Background(), "dev")
	if err != nil {
		t.Fatal(err)
	}
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		if !strings.Contains(r.Header.Get("Authorization"), "Credential=DEVKEY/") || !strings.Contains(r.Header.Get("Authorization"), "/us-west-2/") || r.Header.Get("X-Amz-Security-Token") != "dev-token" {
			t.Error("request did not use the selected profile credentials, token, and region")
		}
		w.Header().Set("Content-Type", "text/xml")
		if r.Method == http.MethodPost {
			fmt.Fprint(w, `<GetCallerIdentityResponse xmlns="https://sts.amazonaws.com/doc/2011-06-15/"><GetCallerIdentityResult><Account>123456789012</Account><Arn>arn:aws:iam::123456789012:user/test</Arn><UserId>test</UserId></GetCallerIdentityResult></GetCallerIdentityResponse>`)
		} else {
			fmt.Fprint(w, `<ListAllMyBucketsResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><Buckets/></ListAllMyBucketsResult>`)
		}
	}))
	defer server.Close()
	cfg.EndpointResolver = aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
		return aws.Endpoint{URL: server.URL, SigningRegion: region, HostnameImmutable: true}, nil
	})
	if err := servicemaster.CheckAWSCredentials(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	for _, svc := range servicestructs.GetServices(cfg) {
		if client, ok := svc.Svc.(*s3.Client); ok {
			if _, err := client.ListBuckets(context.Background(), &s3.ListBucketsInput{}); err != nil {
				t.Fatal(err)
			}
		}
	}
	if atomic.LoadInt32(&calls) != 2 {
		t.Fatalf("expected STS and S3 requests, got %d", calls)
	}
}

func TestAuthenticationFailure(t *testing.T) {
	awsTestEnvironment(t)
	cfg, err := LoadAWSConfig(context.Background(), "dev")
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `<ErrorResponse><Error><Code>InvalidClientTokenId</Code><Message>invalid test token</Message></Error></ErrorResponse>`)
	}))
	defer server.Close()
	cfg.EndpointResolver = aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
		return aws.Endpoint{URL: server.URL, SigningRegion: region}, nil
	})
	if err := servicemaster.CheckAWSCredentials(context.Background(), cfg); err == nil || !strings.Contains(err.Error(), "unable to authenticate") {
		t.Fatalf("expected authentication error, got %v", err)
	}
}
