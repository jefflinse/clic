package lambda

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/jefflinse/clic/ioutil"
	"github.com/jefflinse/clic/provider"
	"github.com/spf13/cobra"
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

// Configure wires up the command's positional arguments, flags, and run behavior.
func (s *Spec) Configure(cmd *cobra.Command) {
	if usage := s.RequestParams.ArgsUsage(); usage != "" {
		cmd.Use += " " + usage
	}

	s.RequestParams.RegisterFlags(cmd.Flags())

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		request, err := s.parameterizedRequest(cmd, args)
		if err != nil {
			return err
		}

		response, functionError, err := executeLambda(cmd.Context(), s.ARN, request)
		if err != nil {
			return err
		} else if functionError != nil {
			fmt.Fprint(os.Stderr, *functionError)
			return nil
		}

		fmt.Print(string(response))
		return nil
	}
}

// Type returns the type.
func (s *Spec) Type() string {
	return "lambda"
}

// Validate validates the provider.
func (s *Spec) Validate() error {
	if s.ARN == "" {
		return fmt.Errorf("invalid %s command spec: missing ARN", s.Type())
	} else if err := s.RequestParams.Validate(); err != nil {
		return err
	}

	return nil
}

func (s *Spec) parameterizedRequest(cmd *cobra.Command, args []string) (map[string]any, error) {
	if err := s.RequestParams.ResolveValues(cmd, args); err != nil {
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
