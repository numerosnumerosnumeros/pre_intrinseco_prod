package env

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

var (
	parameters    = make(map[string]string)
	varValidators []func() error
)

func LoadEnv(ssmClient *ssm.Client, pathPrefix string) error {
	ctx := context.Background()

	input := &ssm.GetParametersByPathInput{
		Path:           aws.String(pathPrefix),
		Recursive:      aws.Bool(true),
		WithDecryption: aws.Bool(true),
	}

	hasMore := true
	for hasMore {
		output, err := ssmClient.GetParametersByPath(ctx, input)
		if err != nil {
			return fmt.Errorf("failed to fetch AWS parameters: %w", err)
		}

		for _, parameter := range output.Parameters {
			key := *parameter.Name
			key = strings.TrimPrefix(key, pathPrefix)
			parameters[key] = *parameter.Value
		}

		if output.NextToken == nil || *output.NextToken == "" {
			hasMore = false
		} else {
			input.NextToken = output.NextToken
		}
	}

	if os.Getenv("DEV_MODE") == "true" {
		if testVal, exists := parameters["VITE_STRIPE_PK_TEST"]; exists {
			parameters["VITE_STRIPE_PK"] = testVal
		}
		if testVal, exists := parameters["STRIPE_SK_TEST"]; exists {
			parameters["STRIPE_SK"] = testVal
		}
		if testVal, exists := parameters["STRIPE_WEBHOOK_SECRET_TEST"]; exists {
			parameters["STRIPE_WEBHOOK_SECRET"] = testVal
		}
		if testVal, exists := parameters["STRIPE_PRICE_ID_TEST"]; exists {
			parameters["STRIPE_PRICE_ID"] = testVal
		}
		if devVal, exists := parameters["VITE_BASE_URL_DEV"]; exists {
			parameters["VITE_BASE_URL"] = devVal
		}
	}

	return nil
}

func RegisterValidator(validator func() error) {
	varValidators = append(varValidators, validator)
}

func ValidateAllVars() error {
	for _, validator := range varValidators {
		if err := validator(); err != nil {
			return err
		}
	}
	return nil
}

func Get(key string) (string, bool) {
	value, exists := parameters[key]
	return value, exists
}
