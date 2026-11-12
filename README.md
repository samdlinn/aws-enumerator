# AWS Service Enumeration

# Disclaimer

The tool is in beta stage (testing in progress), no destructive API Calls used ( read only actions ).
I hope, there will be no issues with the tool. If any issues encountered, please submit the ticket. 

# Description

The AWS Enumerator was created for service enumeration and info dumping for investigations of penetration testers during Black-Box  testing. The tool is intended to speed up the process of Cloud review in case the security researcher compromised AWS Account Credentials. 

AWS Enumerator supports more than 600 API Calls ( reading actions `Get`,  `List`, `Describe` etc... ), and will be extended. 

The tool provides interface for result analysis. All results are saved in json files (one time "Database").

# Install

Build this fork's profile-support branch with Go installed:

```sh
git clone --branch feature/aws-profile https://github.com/samdlinn/aws-enumerator.git
cd aws-enumerator
go build -o aws-enumerator .
```

The Go module path remains `github.com/shabarkin/aws-enumerator` so internal
imports continue to resolve within the fork. Installing the upstream module with
`go install github.com/shabarkin/aws-enumerator@latest` will install the upstream
version instead of this branch.

# Basic Usage

## Credentials setup

### AWS profiles

Use an existing AWS CLI profile without creating a `.env` file:

```sh
./aws-enumerator enum -profile dev -services all
./aws-enumerator enum --profile dev -services iam,s3
./aws-enumerator --profile dev enum -services all
AWS_PROFILE=dev ./aws-enumerator enum -services all
```

Both `-profile` and `--profile` accept a profile name before or after `enum`.
The flag selects a profile from the standard AWS shared `config` and
`credentials` files and overrides `AWS_PROFILE`. The SDK also honors
`AWS_CONFIG_FILE` and `AWS_SHARED_CREDENTIALS_FILE` for custom file locations.
Profile selection applies to both the STS credential check and every service client.

When a profile flag, `AWS_PROFILE`, or `AWS_DEFAULT_PROFILE` is set, the tool does
not read or require `.env`. An explicit profile flag uses that profile's credential
provider, even if static credentials are exported in the shell. With only
`AWS_PROFILE`, exported credentials retain the AWS SDK's normal precedence.
Region environment variables still take precedence over the region in a profile.
Configure a region in the profile or set `AWS_REGION`.

Credential providers are handled by the pinned AWS SDK for Go v2 version, including
shared static/session credentials and supported role profiles. This change does
not add modern IAM Identity Center `sso_session` support or an interactive login flow.
Missing profiles, missing regions, and failed authentication return a nonzero exit
status before service enumeration begins.

### Existing .env workflow

You can still save credentials using the `cred` subcommand:

```sh
./aws-enumerator cred -aws_access_key_id ACCESS_KEY -aws_region us-west-2 -aws_secret_access_key SECRET_KEY
```

This creates `.env` in the current directory. When no profile selector is set,
`enum` loads `.env` if present. Existing shell environment variables take precedence
over values in `.env`. If `.env` is absent, the tool uses the SDK's normal credential
chain, including the default shared profile and workload credentials.

## Enumeration

To enumerate all services, you should use enum subcommand and supply all value or iam,s3,sts,rds ( no spaces between commas ), etc. ...

```bash
./aws-enumerator enum -services all
```

 If you want to check specific services (up to 10 ):

```bash
./aws-enumerator enum -services iam,sts,rds
```

![_img/Screenshot_2021-04-10_at_13.36.56.png](_img/Screenshot_2021-04-10_at_13.36.56.png)

(`-speed` flag is optional, the default value is `normal` ) There are 3 options `slow`, `normal`, `fast` 

```bash
./aws-enumerator enum -services all -speed slow
```

## Analysis

To analyse the collected information, you should use `dump` subcommand: ( Use `all` for quick overview of available API calls )

```bash
./aws-enumerator dump -services all
```

![_img/Screenshot_2021-04-10_at_13.56.12.png](_img/Screenshot_2021-04-10_at_13.56.12.png)

Analyze specific services (up to 10) `iam,s3,sts`, etc ...

```bash
./aws-enumerator dump -services iam,s3,sts
```

![_img/Screenshot_2021-04-10_at_14.03.16.png](_img/Screenshot_2021-04-10_at_14.03.16.png)

To filter API calls, you should use `-filter` option, start typing the name of API call (`GetA` ...):

```bash
./aws-enumerator dump -services iam -filter GetA
```

![_img/Screenshot_2021-04-10_at_14.06.18.png](_img/Screenshot_2021-04-10_at_14.06.18.png)

To retrieve the result of API call, you should use `-print` option

```bash
./aws-enumerator dump -services iam -filter ListS -print
```

![_img/Screenshot_2021-04-10_at_14.08.01.png](_img/Screenshot_2021-04-10_at_14.08.01.png)

## Demo Video

[Pavel Shabarkin LinkedIn](https://www.linkedin.com/posts/pavelshabarkin_cybersecurity-hacking-awssecurity-activity-6785479892881416192-O29U/)
