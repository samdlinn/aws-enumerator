package helper

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/joho/godotenv"
)

// LoadAWSConfig keeps legacy .env credentials optional. Selecting a profile
// bypasses .env entirely, including region settings left by a previous account.
func LoadAWSConfig(ctx context.Context, profile string) (aws.Config, error) {
	if profile == "" && os.Getenv("AWS_PROFILE") == "" && os.Getenv("AWS_DEFAULT_PROFILE") == "" {
		if err := godotenv.Load(".env"); err != nil && !os.IsNotExist(err) {
			return aws.Config{}, fmt.Errorf("unable to load .env: %w", err)
		}
	}
	var options []func(*config.LoadOptions) error
	if profile != "" {
		options = append(options, config.WithSharedConfigProfile(profile))
	}
	cfg, err := config.LoadDefaultConfig(ctx, options...)
	if err != nil {
		return aws.Config{}, fmt.Errorf("unable to load AWS configuration: %w", err)
	}
	selectedProfile := profile
	if selectedProfile == "" {
		selectedProfile = os.Getenv("AWS_PROFILE")
	}
	if selectedProfile == "" {
		selectedProfile = os.Getenv("AWS_DEFAULT_PROFILE")
	}
	// The pinned SDK can ignore a missing profile. Verify the loaded source
	// before allowing a fallback to environment or workload credentials.
	if selectedProfile != "" {
		found := false
		for _, source := range cfg.ConfigSources {
			if shared, ok := source.(config.SharedConfig); ok && strings.EqualFold(shared.Profile, selectedProfile) {
				found = true
			}
		}
		if !found {
			return aws.Config{}, fmt.Errorf("AWS profile %q was not found in shared configuration or credentials files", selectedProfile)
		}
	}
	if cfg.Region == "" {
		return aws.Config{}, fmt.Errorf("AWS region is required; set it in your profile or AWS_REGION")
	}
	return cfg, nil
}
