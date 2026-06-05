package lambda

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/jefflinse/clic/ioutil"
	"github.com/jefflinse/clic/provider"
	"github.com/urfave/cli/v2"
)

// Spec describes the provider.
type Spec struct {
	ARN           string                `json:"arn"                      yaml:"arn"`
	RequestParams provider.ParameterSet `json:"request_params,omitempty" yaml:"request_params,omitempty"`
}

// New creates a new provider.
func New(v any) (provider.Provider, error) {
	s := Spec{}
	return &s, ioutil.Intermarshal(v, &s)
}

// ArgsUsage returns usage text for the arguments.
func (s Spec) ArgsUsage() string {
	argNames := []string{}
	for _, param := range s.RequestParams {
		if param.Required {
			argNames = append(argNames, param.CLIFlagName())
		}
	}

	return strings.Join(argNames, " ")
}

// CLIActionFn creates a CLI action function.
func (s Spec) CLIActionFn() cli.ActionFunc {
	return func(ctx *cli.Context) error {
		request, err := s.parameterizedRequest(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n\n", err)
			cli.ShowCommandHelpAndExit(ctx, ctx.Command.Name, 1)
		}

		response, functionError, err := executeLambda(ctx.Context, s.ARN, request)
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
	return s.RequestParams.CreateCLIFlags()
}

// Type returns the type.
func (s Spec) Type() string {
	return "lambda"
}

// Validate validates the provider.
func (s Spec) Validate() error {
	if s.ARN == "" {
		return fmt.Errorf("invalid %s command spec: missing ARN", s.Type())
	} else if err := s.RequestParams.Validate(); err != nil {
		return err
	}

	return nil
}

func (s *Spec) parameterizedRequest(ctx *cli.Context) (map[string]any, error) {
	if err := s.RequestParams.ResolveValues(ctx); err != nil {
		return nil, err
	}

	request := map[string]any{}
	for _, param := range s.RequestParams {
		request[param.Name] = param.Value()
	}

	return request, nil
}

// Executes the AWS Lambda function specified by an ARN, passing the specified payload, if any.
func executeLambda(ctx context.Context, arn string, request any) ([]byte, *string, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, nil, err
	}

	client := lambda.NewFromConfig(cfg)

	payload, err := json.Marshal(request)
	if err != nil {
		return nil, nil, err
	}

	result, err := client.Invoke(ctx, &lambda.InvokeInput{FunctionName: aws.String(arn), Payload: payload})
	if err != nil {
		return nil, nil, err
	}

	return result.Payload, result.FunctionError, nil
}
