package lambda

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/jefflinse/handyman/spec"
	"github.com/urfave/cli/v2"
)

// New creates a new command that executes an AWS lambda function and prints its results
func New(cmdSpec *spec.Command) *cli.Command {
	return &cli.Command{
		Name:   cmdSpec.Name,
		Usage:  cmdSpec.Description,
		Action: newActionFn(cmdSpec.LambdaARN, cmdSpec.LambdaRequestParameters),
		Flags:  generateFlags(cmdSpec.LambdaRequestParameters),
	}
}

// Creates the action function.
func newActionFn(lambdaARN string, params []*spec.Parameter) cli.ActionFunc {
	paramTypes := map[string]string{}
	for _, param := range params {
		paramTypes[param.Name] = param.Type
	}

	return func(ctx *cli.Context) error {
		request := map[string]interface{}{}
		for _, flagName := range ctx.LocalFlagNames() {
			reqParamName := toUnderscores(flagName)
			switch paramTypes[reqParamName] {
			case spec.BoolParamType:
				request[reqParamName] = ctx.Bool(flagName)
			case spec.IntParamType:
				request[reqParamName] = ctx.Int(flagName)
			case spec.NumberParamType:
				request[reqParamName] = ctx.Float64(flagName)
			case spec.StringParamType:
				request[reqParamName] = ctx.String(flagName)
			}
		}

		response, functionError, err := executeLambda(lambdaARN, request)
		if err != nil {
			return err
		} else if functionError != nil {
			fmt.Fprint(os.Stderr, *functionError)
		}

		fmt.Print(string(response))
		return nil
	}
}

// Generates the set of command line flags for this command.
func generateFlags(params []*spec.Parameter) []cli.Flag {
	flags := []cli.Flag{}
	for _, param := range params {
		var flag cli.Flag
		switch param.Type {
		case spec.StringParamType:
			flag = &cli.StringFlag{
				Name:     toDashes(param.Name),
				Usage:    param.Description,
				Required: param.Required,
			}
		}

		flags = append(flags, flag)
	}

	return flags
}

// Executes the AWS Lambda function specified by an ARN, passing the specified payload, if any.
func executeLambda(arn string, request interface{}) ([]byte, *string, error) {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	client := lambda.New(sess)

	payload, err := json.Marshal(request)
	if err != nil {
		return nil, nil, err
	}

	result, err := client.Invoke(&lambda.InvokeInput{FunctionName: aws.String(arn), Payload: payload})
	if err != nil {
		return nil, nil, err
	}

	return result.Payload, result.FunctionError, nil
}

// Underscores to dashes.
func toDashes(str string) string {
	return strings.ReplaceAll(str, "_", "-")
}

// Dashes to underscores.
func toUnderscores(str string) string {
	return strings.ReplaceAll(str, "-", "_")
}
