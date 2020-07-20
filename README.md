# Handyman - Do All Sorts of Things

![CI](https://github.com/jefflinse/handyman/workflows/CI/badge.svg?branch=master)

Compose, generate, and run custom CLI tools using simple JSON specs.

- [Overview](#overview)
- [Quickstart](#quickstart)
- [Specification Format](#specification-format)
  - [App](#app)
  - [Command](#command)
  - [Parameter](#parameter)
- [Command Types](#command-types)
  - [EXEC](#Exec)
  - [LAMBDA](#Lambda)
  - [NOOP](#Noop)
  - [SUBCOMMANDS](#Subcommands)
- [Roadmap](#roadmap)

## Overview

Handyman is a set of tools for generating CLI applications from JSON specs and running them. Rather than maintaining source code for a command line tool whose command set frequently changes, Handyman lets you define a command hierarchy in a simple JSON spec. This spec can either be run directly or precompiled into a native Go binary.

## Quickstart

Create a Handyman spec:

`myapp.json`:

```json
{
    "name": "myapp",
    "description": "An example of a Handyman app",
    "commands": [
        {
            "name": "say-hello",
            "description": "prints a greeting to the world",
            "type": "EXEC",
            "exec": "echo Hello, World!"
        }
    ]
}
```

The `runner` tool runs a Handyman spec as an app on-the-fly. Its only required argument is the path to a spec file; all remaining arguments are passed to the app.

Run the spec without any additional arguments to view its usage:

```bash
$ runner ./myapp.json

NAME:
   myapp - An example of a Handyman app

USAGE:
   myapp  command [command options] [arguments...]

COMMANDS:s
   say-hello  prints a greeting to the world
```

Now run the `say-hello` command:

```bash
$ runner ./myapp.json say-hello
Hello, World!
```

## Specification Format

A Handyman spec is a JSON file. The root object describes the application, which contains one or more commands, each of which can contain any number of nested subcommands.

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
| `type` | The [type](#command-types) of command. | string | true |
| `description` | A description of the command. | string | true |
| `subcommands` | A set of commmand specs. | array | when `type=SUBCOMMANDS` |
| `exec` | The local shell command to execute. | string |when `type=EXEC` |
| `lambda_arn` | The ARN of an AWS Lambda function. | string |when `type=LAMBDA` |
| `lambda_request_parameters` | A set of request [Parameters](#parameter) accepted by the Lambda function. | array |when `type=LAMBDA` |

### Parameter

Some commands take additional parameters. Each parameter spec has the following properties:

| Propery | Description | Type | Required |
| ------- | ----------- | ---- | -------- |
| `name` | The name of the parameter. Must use snake_casing. | string | true |
| `description` | A description of the parameter. | string | true |
| `type` | The type of value the parameter accepts. | string | true |
| `required` | Whether or not the parameter is required. Default is false. | bool | false |

## Command Types

Handyman supports the following command types:

| Type | Description | Required Fields | Optional Fields |
| ---- | ----------- | --------------- | --------------- |
| `EXEC` | Execute a local shell command | `exec` | |
| `LAMBDA` | Execute an AWS lambda function | `lambda_arn` | `lambda_request_parameters` |
| `NOOP` | Do nothing (no-op) |  |  |

### Exec

An `EXEC` command runs a local shell command on the current system. It is akin to opening a shell and issuing the command normally.

Here's an example of a command that prints the current year:

```json
{
    "name": "current-year",
    "description": "Show the current year",
    "type": "EXEC",
    "exec": "date +Y"
}
```

### Lambda

A `LAMBDA` command executes an AWS Lambda function. It prints the response to stdout and any errors to stderr, respectively. When using this command type, the command spec must include the ARN of the Lambda to execute, and optionally any request parameters to be included. The request parameters are available as command line flags in the app.

Here's an example of a command that invokes a lambda function that accepts a single request parameter called `site_name`:

```json
{
    "name": "update-site-name",
    "description": "update the website name",
    "type": "LAMBDA",
    "lambda_arn": "aws:arn:us-west-2:1234567890:lambda:function:update-site-name:$LATEST",
    "lambda_request_parameters": [
        {
            "name": "site_name",
            "type": "string",
            "description": "the name of the website",
            "required": true
        }
    ]
}
```

> **Note:** The command name does not need to match the Lambda function's name.

### Noop

A `NOOP` command does nothing; it's truely a no-op. This is useful in scenarios where you might want to mock our your entire app's command struture in an app spec before specifying the actual behavior of each action.

```json
{
    "name": "do-nothing",
    "description": "does absolutely nothing",
    "type": "NOOP"
}
```

### Subcommands

A `SUBCOMMANDS` command contains one or more subcommands. Each subcommand is defined by a command spec and may itself be a `SUBCOMMANDS` type containing other subcommands.

At least one subcommand must be defined.

```json
{
    "name": "inventory",
    "description": "manage inventory",
    "type": "SUBCOMMANDS",
    "subcommands": [
        {
            "name": "add-item",
            "description": "add an item to inventory",
            "type": "LAMBDA",
            "lambda_arn": "some:lambda:arn"
        },
        {
            "name": "clear-local-cache",
            "description": "clear the local inventory cache",
            "type": "EXEC",
            "exec": "/usr/local/bin/clearcache"
        },
    ]
}
```

## Roadmap

A very rough list of features and improvements I have in mind:

- Codegen to produce a native Go binary from a spec
- Remove bash shell dependency from EXEC commands
- Support additional parameter types:
  - natives (int, number, bool)
  - binary data
- Support reading parameter values from files
- Support directory-based spec composition (a la Terraform)
- Improved unit test coverage
