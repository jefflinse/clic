# Handyman

![CI](https://github.com/jefflinse/handyman/workflows/CI/badge.svg?branch=master)

**Don't write CLI tools to manage your services; generate them!**

Handyman is a set of tools that allow you to compose, generate, and run custom CLI tools using simple text-based configuration files.

- [Overview](#overview)
- [Quickstart](#quickstart)
- [Specification Format](#specification-format)
  - [App](#app)
  - [Command](#command)
  - [Parameter](#parameter)
- [Command Types](#command-types)
  - [exec - run any local command](#exec)
  - [lambda - execute an AWS lambda function](#lambda)
  - [noop - do nothing](#noop)
- [Roadmap](#roadmap)

## Overview

If you frequently find yourself writing shell scripts make calling various web services and cloud resources easier, Handyman is for you. Handyman lets you define a hierarchy of command line tools using a simple YAML or JSON configuation file.

Handyman is built around the concept of commands, each of which is handled by a command provider. Providers define what happens when a command runs, such as executing a local command, calling a REST endpoint, interacting with a cloud resource, and so forth.

## Quickstart

Create a Handyman spec:

**myapp.yml**:

```yaml
name: myapp
description: an example of a Handyman app
commands:
  - name: say-hello
    description: prints a greeting to the world
    exec:
      path: "echo"
      args: ["Hello, World!"]
```

The `runner` tool runs a Handyman spec as an app on-the-fly. Its only required argument is the path to a spec file; all remaining arguments are passed to the app.

Run the app spec without any additional arguments to view its usage:

```bash
$ runner myapp.yml
myapp - an example of a Handyman app

usage:
  myapp  command [command options] [arguments...]

commands:
  say-hello  prints a greeting to the world
```

Now run our app spec with the `say-hello` command:

```bash
$ runner myapp.yml say-hello
Hello, World!
```

The `compiler` tool compiles a Handyman app spec into a native Go binary. Let's compile our app:

```bash
$ compiler myapp.yml

$ ls
myapp     myapp.yml
```

Now we can run it directly:

```bash
$ ./myapp
myapp - an example of a Handyman app

usage:
  myapp  command [command options] [arguments...]

commands:
  say-hello  prints a greeting to the world
```

```bash
$ ./myapp say-hello
Hello, World!
```

Handyman can do more than just execute local commands. See the complete list of [Command Types](#command-types) to learn more.

## Specification Format

A Handyman spec can be written in either YAML or JSON. The root object describes the application, which contains one or more commands, each of which can contain any number of nested subcommands.

### App

The app spec has the following properties:

| Property | Description | Type | Required |
| -------- | ----------- | ---- | -------- |
| `name` | The name of the app as invoked on the command line. | string | true |
| `description` | A description of the app. | string | true |
| `commands` | A set of commmand specs. | array | false |

### Command

A command spec has the following properties:

| Propery | Description | Type | Required |
| ------- | ----------- | ---- | -------- |
| `name` | The name of the command as invoked on the command line. | string | true |
| `description` | A description of the command. | string | true |
| `<type>` | Configuration for the provider that executes the logic for the command. | object | true |

`<type>` must be the name of a supported command type, and its value must be an object defining the configuration for that command type. See [Command Types](#command-types) for information how how to configure each command type.

### Parameter

Some commands take additional parameters. Each parameter spec has the following properties:

| Propery | Description | Type | Required |
| ------- | ----------- | ---- | -------- |
| `name` | The name of the parameter. Must use snake_casing. | string | true |
| `description` | A description of the parameter. | string | true |
| `type` | The type of value the parameter accepts. Must be one of [**int**, **number**, **string**]. | string | true |
| `required` | Whether or not the parameter is required. Default is false. | bool | false |

## Command Types

- [exec](#exec)
- [lambda](#lambda)
- [noop](#noop)

### exec

An `exec` command runs a local command. It executes the provided command, directly passing any supplied arguments.

```yaml
name: current-year
description: print the current year
exec:
  name: date
  args: ["+Y"]
```

### lambda

A `lambda` command executes an AWS Lambda function. It prints the response to stdout and any errors to stderr, respectively. When using this command type, the command spec must include the ARN of the Lambda to execute, and optionally any request parameters to be included. The request parameters are available as command line flags in the app.

Here's an example of a command that invokes a lambda function that accepts a single request parameter called `site_name`:

```yaml
name: update-site-name
description: update the website name
lambda:
  arn: "aws:arn:some:valid:arn"
  request_params:
    - name: site_name
      type: string
      description: the name of the website
      required: true
```

> **Note:** The command name does not need to match the Lambda function's name.

### noop

A `noop` command does nothing; it's truely a no-op. This is useful in scenarios where you might want to mock our your entire app's command struture in an app spec before specifying the actual behavior of each action.

```yaml
name: do-nothing
description: does absolutely nothing
noop:
```

## Roadmap

A very rough list of features and improvements I have in mind:

- App-level and command-level versioning
- Support for app- and command-level variables
- Support directory-based spec composition (a la Terraform)
- Exec provider should support parameters
- Add REST provider for calling endpoints
- Support reading parameter values from files
- Support for producing binaries/scripts for other languages
- Improved unit test coverage
