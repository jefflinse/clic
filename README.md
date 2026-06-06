# clic

![build status](https://img.shields.io/github/actions/workflow/status/jefflinse/clic/main-ci.yml?branch=main) ![GitHub release (latest by date)](https://img.shields.io/github/v/release/jefflinse/clic) ![go version](https://img.shields.io/github/go-mod/go-version/jefflinse/clic)

clic lets you quickly define, generate, and run custom CLI tools using simple text-based configuration files — or instantly from an existing OpenAPI document.

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
- [OpenAPI](#openapi)
- [Roadmap](#roadmap)

## Overview

clic eliminates the need to maintain aliases, shell scripts, and custom CLI tools while developing and testing web services. Instead, quickly define a hierarchy of command line actions using a simple YAML or JSON spec file.

```yaml
# myapp.clic.yml
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
$ clic run myapp.clic.yml
tools for managing my service

Usage:
  myapp [command]

Available Commands:
  list-items  list items in the catalog

Flags:
  -h, --help   help for myapp
```

```bash
$ clic run myapp.clic.yml list-items --category apparel
{"args":{"category":"apparel"},"url":"https://postman-echo.com/get?category=apparel"}
```

## Installation

The easiest way to install clic on macOS is via homebrew:

```bash
$ brew install jefflinse/clic/clic
```

Verify your installation by running the clic CLI tool, `clic`:

```bash
$ clic version
```

## Quickstart

Create a clic spec:

**myapp.yml**:

```yaml
name: myapp
description: an example of a clic app
commands:
  - name: say-hello
    description: prints a greeting to the world
    exec:
      name: "echo"
      args: ["Hello, World!"]
```

The `clic run` command runs a clic spec as an app on-the-fly. Its only required argument is the path to a spec file; all remaining arguments are passed to the app.

Run the app spec without any additional arguments to view its usage:

```bash
$ clic run myapp.yml
an example of a clic app

Usage:
  myapp [command]

Available Commands:
  say-hello  prints a greeting to the world

Flags:
  -h, --help   help for myapp
```

Now run our app spec with the `say-hello` command:

```bash
$ clic run myapp.yml say-hello
Hello, World!
```

The `clic build` command compiles a clic app spec into a native Go binary. Let's compile our app:

```bash
$ clic build myapp.yml

$ ls
myapp     myapp.yml
```

Now we can run it directly:

```bash
$ ./myapp
an example of a clic app

Usage:
  myapp [command]

Available Commands:
  say-hello  prints a greeting to the world

Flags:
  -h, --help   help for myapp
```

```bash
$ ./myapp say-hello
Hello, World!
```

clic can do more than just execute local commands. See the complete list of [Command Providers](#command-providers) to learn more.

## Specification Format

A clic spec can be written in either YAML or JSON. The root object describes the application, which contains one or more commands, each of which can contain any number of nested subcommands.

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

## OpenAPI

clic can turn any OpenAPI 3.x document into a CLI. Internally it *compiles* the OpenAPI spec into a clic spec, then runs or builds that — so everything in this README applies to the result.

Anywhere clic takes a spec, it accepts a local path **or** an `http(s)` URL, and auto-detects whether it's a clic spec or an OpenAPI document:

```bash
# auto-mode: detect the format and run it, passing trailing args to the app
$ clic ./petstore.openapi.yaml pets list --limit 10
$ clic https://api.example.com/openapi.json users get 42

# force a format (errors if the file isn't that format)
$ clic run --openapi ./api.yaml ...
$ clic run --spec    ./app.clic.yml ...

# see (and keep) the generated clic spec — edit it, commit it, run it
$ clic convert ./petstore.openapi.yaml -o petstore.clic.yml

# compile straight to a native binary
$ clic build ./petstore.openapi.yaml
```

### How operations map to commands

Paths become nested command groups and the HTTP method picks the verb:

| OpenAPI | clic command |
| ------- | ------------ |
| `GET /pets` | `pets list` (alias `ls`) |
| `POST /pets` | `pets create --body @pet.json` (or `-i`) |
| `GET /pets/{id}` | `pets get <id>` |
| `PATCH /pets/{id}` | `pets update <id> --body @patch.json` (or `-i`) |
| `PUT /pets/{id}` | `pets update <id>` — or `replace` when a `PATCH` also exists |
| `DELETE /pets/{id}` | `pets delete <id>` (alias `rm`) |
| `GET /users/{id}/posts` | `users posts list <id>` |
| `POST /pets/{id}/vaccinate` | `pets vaccinate <id>` (single, childless action) |

Parameters map as follows:

- **path** parameters → required positional arguments, substituted into the URL
- **query** and **header** parameters → flags (required ones become required flags)
- **request body** → `--body` (inline JSON or `@file.json`), or an interactive form with `-i` (see [Interactive input](#interactive-input))

### Interactive input

For operations that take a request body, pass `-i` (`--interactive`) to fill it
in via a terminal form instead of hand-writing JSON. clic builds the form from
the request-body schema: text inputs for strings and numbers, a confirm for
booleans, a select for `enum` values, and grouped fields for nested objects.
Scalar lists are entered one per line; lists of objects are gathered with a
repeatable "add another?" prompt. Required fields are validated, and optional
fields left blank are omitted from the request.

```bash
$ clic -i ./petstore.openapi.yaml pets create
# prompts for name, status (a select), age, tags, ... then sends the request
```

`--body` still works and takes precedence — `-i` only prompts when no `--body`
is supplied.

### Server and authentication

The first `servers` entry becomes the default base URL, overridable with the global `--server` flag. Security schemes (everything except OAuth2) surface as global flags, each with a `CLIC_*` environment-variable fallback:

| Scheme | Flag(s) | Env |
| ------ | ------- | --- |
| HTTP bearer | `--token` | `CLIC_TOKEN` |
| HTTP basic | `--username` / `--password` | `CLIC_USERNAME` / `CLIC_PASSWORD` |
| API key | `--api-key` | `CLIC_API_KEY` |

clic's global flags (`--server`, `-i`, and the auth flags) are clic's own and must be placed **before** the spec; everything after the spec is passed through to the generated app as its own arguments. This keeps them from ever colliding with a parameter of the same name in the spec.

```bash
$ clic --token "$MY_TOKEN" ./api.yaml users get 42
$ CLIC_TOKEN="$MY_TOKEN" clic ./api.yaml users get 42
$ clic --server https://staging.example.com ./api.yaml users get 42
```

> **Note:** OpenAPI 3.0 and 3.1 are supported (Swagger/OpenAPI 2.0 is not).

## Roadmap

A very rough list of features and improvements I have in mind:

- OAuth2 support for OpenAPI-generated CLIs
- App-level and command-level versioning
- Support for app- and command-level variables
- Support directory-based spec composition (a la Terraform)
- Support reading parameter values from files
- Support for producing binaries/scripts for other languages
- registry: cache latest spec content so app can be run even if spec is moved or deleted
- Add run protection for spec files obtained from the internet
- Providers for Azure Functions and Google Cloud Functions
- Support for specifying environment variables
