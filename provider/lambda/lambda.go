package lambda

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/jefflinse/handyman/ioutil"
	"github.com/jefflinse/handyman/provider"
	"github.com/urfave/cli/v2"
)

// Spec describes the provider.
type Spec struct {
	ARN           string                `json:"arn"                      yaml:"arn"`
	RequestParams provider.ParameterSet `json:"request_params,omitempty" yaml:"request_params,omitempty"`
}

// New creates a new provider.
func New(v interface{}) (provider.Provider, error) {
	s := Spec{}
	return &s, ioutil.Intermarshal(v, &s)
}

// CLIActionFn creates a CLI action fuction.
func (s Spec) CLIActionFn() cli.ActionFunc {
	return func(ctx *cli.Context) error {
		request := s.parameterizedRequest(ctx)

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
				Name:     param.CLIFlagName(),
				Usage:    param.Description,
				Required: param.Required,
			}
		case "int":
			flag = &cli.IntFlag{
				Name:     param.CLIFlagName(),
				Usage:    param.Description,
				Required: param.Required,
			}
		case "number":
			flag = &cli.Float64Flag{
				Name:     param.CLIFlagName(),
				Usage:    param.Description,
				Required: param.Required,
			}
		case "string":
			flag = &cli.StringFlag{
				Name:     param.CLIFlagName(),
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

func (s *Spec) parameterizedRequest(ctx *cli.Context) map[string]interface{} {
	request := map[string]interface{}{}
	s.RequestParams.ResolveValues(ctx)
	for _, param := range s.RequestParams {
		request[param.Name] = param.Value()
	}

	return request
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
