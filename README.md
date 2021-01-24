# clic - the command line interface composer

![build status](https://img.shields.io/github/workflow/status/jefflinse/clic/CI) ![GitHub release (latest by date)](https://img.shields.io/github/v/release/jefflinse/clic) ![go version](https://img.shields.io/github/go-mod/go-version/jefflinse/clic)

**clic** is a tool for rapidly defining and composing command line apps from simple configuration files. It utilizes a common app spec format that can be built into a native Go binary.

- [Quickstart](#quickstart)
- [Specification Format](#specification-format)
  - [App](#app)
  - [Command](#command)
- [Command Providers](#command-providers)
  - [exec - execute a local command](#exec)
- [Roadmap](#roadmap)

## Quickstart

Create a clic spec:

```yaml
# mytool.yml
name: mytool
description: an example of a clic app
commands:
  - name: greet
    description: prints a greeting message
    exec:
      path: "echo"
      args: ["hello, world!"]
```

The `clic build` command builds a clic spec into a CLI application:

```shell
$ clic build mytool.yml
```

Run your app with no arguments to view its usage:

```shell
$ ./mytool
an example of a clic app

Usage:
  mytool [command]

Available Commands:
  greet       prints a greeting message

Flags:
  -h, --help   help for mytool

Use "mytool [command] --help" for more information about a command.
```

Run it again, this time specifying the `greet` command:

```shell
$ ./mytool greet
hello, world!
```

clic can do more than just execute local commands. See the complete list of [Command Providers](#command-providers) to learn more.

## Specification Format

A clic spec can be written in YAML or JSON. The root object describes the application, which contains one or more commands, each of which can contain any number of nested subcommands.

### App

An app spec has the following properties:

| Property | Description | Type |
| -------- | ----------- | ---- | -------- |
| `name` | **Required.** The name of the app as invoked on the command line. | string |
| `commands` | **Required.** A set of commmand specs. | array |
| `description` | A description of the app. | string |

### Command

A command spec defines a command's behavior via a provider. It has the following properties:

| Propery | Description | Type |
| ------- | ----------- | ---- |
| `name` | **Required.** The name of the command as invoked on the command line. | string |
| `<provider>` | **Required.** Configuration for the provider that executes the logic for the command. | object |
| `description` | A description of the command. | string |

`<provider>` must be the name of a supported [command provider](#command-providers), and its value is an object defining the provider's configuration.

## Command Providers

- [exec](#exec)

### exec

The `exec` provider runs a local command. It executes the provided command, directly passing any supplied arguments.

```yaml
exec:
  name: echo
  args: ["-e", "hello, world!"]
```

## V1 Roadmap

- Full unit test coverage
- Support for running a spec on the fly (`clic run`) using Go + Cobra defaults for the interface

## Post-V1 Roadmap

- Add functional tests
- Support for app-specific clic config in spec
- Suppport for directory-based spec composition (i.e. Terraform)
- Support for generating and/or building with more than one provider simultaneously
- Support for TOML spec files?
- Support for HCL spec files?
