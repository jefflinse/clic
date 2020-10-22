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
	ARN           string                `json:"arn"                      yaml:"arn"`
	RequestParams provider.ParameterSet `json:"request_params,omitempty" yaml:"request_params,omitempty"`
}

// New creates a new provider.
func New(v interface{}) (provider.Provider, error) {
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

// CLIActionFn creates a CLI action fuction.
func (s Spec) CLIActionFn() cli.ActionFunc {
	return func(ctx *cli.Context) error {
		request, err := s.parameterizedRequest(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n\n", err)
			cli.ShowCommandHelpAndExit(ctx, ctx.Command.Name, 1)
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

func (s *Spec) parameterizedRequest(ctx *cli.Context) (map[string]interface{}, error) {
	if err := s.RequestParams.ResolveValues(ctx); err != nil {
		return nil, err
	}

	request := map[string]interface{}{}
	for _, param := range s.RequestParams {
		request[param.Name] = param.Value()
	}

	return request, nil
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
