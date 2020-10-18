package lambda

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/jefflinse/handyman/ioutil"
	"github.com/jefflinse/handyman/provider"
	"github.com/urfave/cli/v2"
)

// Spec describes the provider.
type Spec struct {
	ARN           string               `json:"arn"                      yaml:"arn"`
	RequestParams []provider.Parameter `json:"request_params,omitempty" yaml:"request_params,omitempty"`
}

// New creates a new provider.
func New(v interface{}) (provider.Provider, error) {
	s := Spec{}
	return &s, ioutil.Intermarshal(v, &s)
}

// CLIActionFn creates a CLI action fuction.
func (s Spec) CLIActionFn() cli.ActionFunc {
	paramTypes := map[string]string{}
	for _, param := range s.RequestParams {
		paramTypes[param.Name] = param.Type
	}

	return func(ctx *cli.Context) error {
		request := map[string]interface{}{}
		for _, flagName := range ctx.LocalFlagNames() {
			reqParamName := toUnderscores(flagName)
			switch paramTypes[reqParamName] {
			case "bool":
				request[reqParamName] = ctx.Bool(flagName)
			case "int":
				request[reqParamName] = ctx.Int(flagName)
			case "number":
				request[reqParamName] = ctx.Float64(flagName)
			case "string":
				request[reqParamName] = ctx.String(flagName)
			}
		}

		response, functionError, err := executeLambda(s.ARN, request)
		if err != nil {
			fmt.Fprint(os.Stderr, err)
			return err
		} else if functionError != nil {
			fmt.Fprint(os.Stderr, *functionError)
			return nil
		}

		fmt.Print(string(response))
		return nil
	}
}

// CLIFlags creates a set of CLI flags.
func (s Spec) CLIFlags() []cli.Flag {
	flags := []cli.Flag{}
	for _, param := range s.RequestParams {
		var flag cli.Flag
		switch param.Type {
		case "bool":
			flag = &cli.BoolFlag{
				Name:     toDashes(param.Name),
				Usage:    param.Description,
				Required: param.Required,
			}
		case "int":
			flag = &cli.IntFlag{
				Name:     toDashes(param.Name),
				Usage:    param.Description,
				Required: param.Required,
			}
		case "number":
			flag = &cli.Float64Flag{
				Name:     toDashes(param.Name),
				Usage:    param.Description,
				Required: param.Required,
			}
		case "string":
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

// Type returns the type.
func (s Spec) Type() string {
	return "lambda"
}

// Validate validates the provider.
func (s Spec) Validate() error {
	if s.ARN == "" {
		return fmt.Errorf("invalid %s command spec: missing ARN", s.Type())
	}

	return nil
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
