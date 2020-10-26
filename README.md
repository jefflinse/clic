# Handyman

![build status](https://img.shields.io/github/workflow/status/jefflinse/handyman/CI) ![GitHub release (latest by date)](https://img.shields.io/github/v/release/jefflinse/handyman) ![go version](https://img.shields.io/github/go-mod/go-version/jefflinse/handyman)

Handyman is a tools that allow you to quickly define, generate, and run custom CLI tools using simple text-based configuration files.

- [Overview](#overview)
- [Installation](#installation)
- [Quickstart](#quickstart)
- [Specification Format](#specification-format)
  - [App](#app)
  - [Command](#command)
  - [Parameter](#parameter)
- [Command Providers](#command-providers)
  - [exec - run any local command](#exec)
  - [lambda - execute an AWS lambda function](#lambda)
  - [noop - do nothing](#noop)
  - [rest - make a request to a REST endpoint](#rest)
- [Roadmap](#roadmap)

## Overview

Handyman eliminates the need to maintain aliases, shell scripts, and custom CLI tools while developing and testing web services. Instead, quickly define a hierarchy of command line actions using a simple YAML or JSON spec file.

```yaml
# myapp.handyman.yml
name: myapp
description: tools for managing my service
commands:
  - name: list-items
    description: list items in the catalog
    rest:
      method: GET
      endpoint: https://postman-echo.com/get
      query_params:
        - name: category
          type: string
          description: limit to items in category
```

```bash
$ hm run myapp.handyman.yml
myapp - tools for managing my service

usage:
  myapp  command [command options] [arguments...]

commands:
  list-items  list all items in the catalog
```

```bash
$ hm run myapp.handyman.yml list-items --category apparel
{"args":{"category":"apparel"},"url":"https://postman-echo.com/get?category=apparel"}
```

## Installation

The easiest way to install Handyman on macOS is via homebrew:

```bash
$ brew install jefflinse/handyman/handyman
```

Verify your installation by running the handyman CLI tool, `hm`:

```bash
$ hm version
```

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
      name: "echo"
      args: ["Hello, World!"]
```

The `hm run` command runs a Handyman spec as an app on-the-fly. Its only required argument is the path to a spec file; all remaining arguments are passed to the app.

Run the app spec without any additional arguments to view its usage:

```bash
$ hm run myapp.yml
myapp - an example of a Handyman app

usage:
  myapp  command [command options] [arguments...]

commands:
  say-hello  prints a greeting to the world
```

Now run our app spec with the `say-hello` command:

```bash
$ hm run myapp.yml say-hello
Hello, World!
```

The `hm build` command compiles a Handyman app spec into a native Go binary. Let's compile our app:

```bash
$ hm build myapp.yml

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

Handyman can do more than just execute local commands. See the complete list of [Command Providers](#command-providers) to learn more.

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

A command spec either defines command behvior via a provider, or defines a list of subcommands. It has the following properties:

| Propery | Description | Type | Required |
| ------- | ----------- | ---- | -------- |
| `name` | The name of the command as invoked on the command line. | string | true |
| `description` | A description of the command. | string | true |
| `subcommands` | Subcommands for this command. | array | true (if no provider specified) |
| `<provider>` | Configuration for the provider that executes the logic for the command. | object | true (if no subcommands specified) |

Exactly one of `<provider>` or `subcommands` must be specified.

`<provider>` must be the name of a supported command provider, and its value must be an object defining the configuration for that provider. See [Command Providers](#command-providers) for information how how to configure each provider.

### Parameter

Some commands take additional parameters. Each parameter spec has the following properties:

| Propery | Description | Type | Required |
| ------- | ----------- | ---- | -------- |
| `name` | The name of the parameter. Must use snake_casing. | string | true |
| `description` | A description of the parameter. | string | false |
| `type` | The type of value the parameter accepts. Must be one of [**int**, **number**, **string**]. | string | true |
| `required` | Whether or not the parameter is required. Default is false. | bool | false |
| `default` | The default value to use for the parameter, if the parameter is not required. | _type_ | false |
| `as_flag` | For boolean type parameters, defining this will cause the parameter to render the specified value when true. | string | false |

## Command Providers

- [exec](#exec)
- [lambda](#lambda)
- [noop](#noop)
- [rest](#rest)
- [subcommands](#subcommands)

### exec

An `exec` command runs a local command. It executes the provided command, directly passing any supplied arguments.

```yaml
name: sample-exec
description: example command spec using an exec command provider
exec:
  name: date
  args: ["{{params.format}}"]
  params:
    - name: format
      type: string
      description: the format string for the date
      default: "+Y"
```

### lambda

A `lambda` command executes an AWS Lambda function. It prints the response to stdout and any errors to stderr, respectively. When using this provider, the command spec must include the ARN of the Lambda to execute, and optionally any request parameters to be included. The request parameters are sent as the JSON payload to the Lambda function and are available as command line flags in the app.

Here's an example of a command that invokes a lambda function that accepts a single request parameter called `site_name`:

```yaml
name: sample-lambda
description: example command spec using a Lambda command provider
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

A `noop` command does nothing; it's truly a no-op. This is useful in scenarios where you might want to mock our your entire app's command struture in an app spec before specifying the actual behavior of each action.

```yaml
name: sample-noop
description: example command spec using a noop command provider
noop:
```

### rest

A `rest` command makes a request to a REST endpoint. It can pass parameters as query string parameters or JSON-formatted request body parameters.

```yaml
name: sample-rest
description: example command spec using a REST command provider
rest:
  endpoint: https://postman-echo.com/get
  method: GET
  query_params:
    - name: my_query_param
      type: string
      description: a query parameter passed to the request
```

### subcommands

A command specifying `subcommands` instead of a provider allows for a spec to define a "Git-like" hierarchy of commands:

```yaml
name: sample-with-subcommands
description: example command spec using subcommands
subcommands:  
  - name: call-my-api
    description: call my API
    rest:
      endpoint: https://postman-echo.com/get
      method: GET
      query_params:
        - name: my_query_param
          type: string
          description: a query parameter passed to the request
  - name: clean temp directory
    description: cleans the temp directory
    exec:
      name: rm
      args: ["-f", "/tmp"]
```

## Roadmap

A very rough list of features and improvements I have in mind:

- App-level and command-level versioning
- Support for app- and command-level variables
- Support directory-based spec composition (a la Terraform)
- Support reading parameter values from files
- Support for producing binaries/scripts for other languages
- Improved unit test coverage
- registry: cache latest spec content so app can be run even if spec is moved or deleted
- Add run protection for spec files obtained from the internet
- Providers for Azure Functions and Google Cloud Functions
- Support for specifying environment variables
