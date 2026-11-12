package helper

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

type EnumOptions struct {
	Services string
	Speed    string
	Profile  string
}

func ParseEnumArgs(args []string, profile string) (EnumOptions, error) {
	var options EnumOptions
	flags := flag.NewFlagSet("enum", flag.ContinueOnError)
	flags.StringVar(&options.Services, "services", "all", "services to enumerate")
	flags.StringVar(&options.Speed, "speed", "normal", "enumeration speed")
	flags.StringVar(&options.Profile, "profile", profile, "AWS shared profile")
	flags.Usage = func() { fmt.Fprint(os.Stderr, Cloudrider_enum_help) }
	if err := flags.Parse(args); err != nil {
		return options, err
	}
	if flags.NArg() != 0 {
		return options, fmt.Errorf("unexpected enum argument %q", flags.Arg(0))
	}
	profileSet := false
	flags.Visit(func(f *flag.Flag) { profileSet = profileSet || f.Name == "profile" })
	if profileSet && strings.TrimSpace(options.Profile) == "" {
		return options, fmt.Errorf("-profile requires a non-empty name")
	}
	return options, nil
}
