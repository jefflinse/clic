package handyman

import (
	"encoding/json"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
)

// Executes the AWS Lambda function specified by an ARN, passing the specified payload, if any.
func executeLambda(arn string, request interface{}) ([]byte, error) {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	client := lambda.New(sess)

	payload, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	result, err := client.Invoke(&lambda.InvokeInput{FunctionName: aws.String(arn), Payload: payload})
	if err != nil {
		return nil, err
	}

	return result.Payload, nil
}
