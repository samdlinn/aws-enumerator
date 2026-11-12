package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/shabarkin/aws-enumerator/helper"
	"github.com/shabarkin/aws-enumerator/utils"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

// parseCommand parses flags before the subcommand. Enumeration flags are parsed
// separately so profile selection works on either side of enum.
func parseCommand(args []string) (string, []string, string, error) {
	root := flag.NewFlagSet("aws-enumerator", flag.ContinueOnError)
	profile := root.String("profile", "", "AWS shared profile for enum")
	root.Usage = func() { fmt.Fprint(os.Stderr, helper.Cloudrider_help) }
	if err := root.Parse(args); err != nil {
		return "", nil, "", err
	}
	if root.NArg() == 0 {
		root.Usage()
		return "", nil, "", fmt.Errorf("a command is required")
	}
	if *profile != "" && root.Arg(0) != "enum" {
		return "", nil, "", fmt.Errorf("-profile is only supported with enum")
	}
	profileSet := false
	root.Visit(func(f *flag.Flag) { profileSet = profileSet || f.Name == "profile" })
	if profileSet && strings.TrimSpace(*profile) == "" {
		return "", nil, "", fmt.Errorf("-profile requires a non-empty name")
	}
	return root.Arg(0), root.Args()[1:], *profile, nil
}

func run(args []string) error {

	helper.Cred.Usage = func() {
		fmt.Fprint(os.Stderr, helper.Cloudrider_cred_help)
	}
	helper.Dump.Usage = func() {
		fmt.Fprint(os.Stderr, helper.Cloudrider_dump_help)
	}

	command, commandArgs, profile, err := parseCommand(args)
	if err != nil {
		return err
	}
	switch command {

	case "cred":
		helper.Cred.Parse(commandArgs)
		utils.CreateAWScredentialsFile(helper.AWS_region, helper.AWS_access_key_id, helper.AWS_secret_access_key, helper.AWS_session_token)
		fmt.Println(utils.Green("Message: "), utils.Yellow("File"), utils.Red(".env"), utils.Yellow("with AWS credentials were created in current folder"))
	case "enum":
		opts, err := helper.ParseEnumArgs(commandArgs, profile)
		if err != nil {
			return err
		}
		if err := helper.SetEnumerationPipeline(opts.Services, opts.Speed, opts.Profile); err != nil {
			return err
		}
		fmt.Println(utils.Green("Message: "), utils.Yellow("Enumeration finished"))

	case "dump":
		helper.Dump.Parse(commandArgs)
		helper.DumpInfo(helper.Services_dump, helper.Print, helper.Filter, helper.Errors_dump)

	default:
		fmt.Println(helper.Cloudrider_help)
		return fmt.Errorf("unknown command %q", command)
	}
	return nil
}
