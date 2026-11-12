package main

import (
	"errors"
	"flag"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shabarkin/aws-enumerator/helper"
)

func TestProfileArguments(t *testing.T) {
	for _, args := range [][]string{
		{"-profile", "dev", "enum", "-services", "s3"},
		{"--profile=dev", "enum", "-services", "s3"},
		{"enum", "-profile", "dev", "-services", "s3"},
		{"enum", "--profile=dev", "-services", "s3"},
		{"enum", "-services", "s3", "--profile", "dev"},
		{"-profile", "other", "enum", "--profile", "dev", "-services", "s3"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			command, rest, profile, err := parseCommand(args)
			if err != nil || command != "enum" {
				t.Fatalf("parseCommand: %q, %v", command, err)
			}
			opts, err := helper.ParseEnumArgs(rest, profile)
			if err != nil || opts.Profile != "dev" || opts.Services != "s3" || opts.Speed != "normal" {
				t.Fatalf("ParseEnumArgs: %+v, %v", opts, err)
			}
		})
	}
}

func TestInvalidProfileArguments(t *testing.T) {
	for _, args := range [][]string{
		{"-profile"}, {"-profile=", "enum"}, {"--profile", " ", "enum"},
		{"-profile", "dev", "dump"}, {"enum", "-profile"},
		{"enum", "--profile="}, {"enum", "-profile", " "},
		{"enum", "extra", "-profile", "dev"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			_, rest, profile, err := parseCommand(args)
			if err == nil {
				_, err = helper.ParseEnumArgs(rest, profile)
			}
			if err == nil {
				t.Fatal("expected argument error")
			}
		})
	}
}

func TestHelp(t *testing.T) {
	if _, _, _, err := parseCommand([]string{"--help"}); !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("root help: %v", err)
	}
	if _, err := helper.ParseEnumArgs([]string{"-h"}, ""); !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("enum help: %v", err)
	}
}

func TestCLIFailure(t *testing.T) {
	if os.Getenv("AWS_ENUMERATOR_TEST_PROCESS") == "1" {
		os.Args = []string{"aws-enumerator", "enum", "--profile", "missing"}
		main()
		return
	}
	dir := t.TempDir()
	for _, name := range []string{"config", "credentials"} {
		if err := ioutil.WriteFile(filepath.Join(dir, name), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestCLIFailure$")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "AWS_ENUMERATOR_TEST_PROCESS=1", "AWS_CONFIG_FILE="+filepath.Join(dir, "config"), "AWS_SHARED_CREDENTIALS_FILE="+filepath.Join(dir, "credentials"), "AWS_EC2_METADATA_DISABLED=true")
	output, err := cmd.CombinedOutput()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
		t.Fatalf("expected exit 1, got %v: %s", err, output)
	}
	if !strings.Contains(string(output), "missing") || strings.Contains(string(output), "Enumeration finished") {
		t.Fatalf("unexpected failure output: %s", output)
	}
}
