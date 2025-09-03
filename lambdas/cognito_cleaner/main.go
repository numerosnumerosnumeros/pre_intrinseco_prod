package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	cognito "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// *
// **
// ***
// ****
// ***** logger
var logger *zap.Logger

func setupLogger() *zap.Logger {
	var core zapcore.Core
	var options []zap.Option

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.LevelKey = "level"
	encoderConfig.MessageKey = "message"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder // "error", "warn", "info"

	core = zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig), // JSON format for CloudWatch filters
		zapcore.AddSync(os.Stdout),
		zap.InfoLevel,
	)

	options = append(options, zap.AddCaller())

	return zap.New(core, options...)
}

func init() {
	logger = setupLogger()
}

// *
// **
// ***
// ****
// ***** main
func logic() error {
	ctx := context.Background()
	sdkConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("unable to load SDK config: %v", err)
	}

	ssmClient := ssm.NewFromConfig(sdkConfig)
	paramOutput, err := ssmClient.GetParameter(ctx, &ssm.GetParameterInput{
		Name:           aws.String("COGNITO_USER_POOL_ID"),
		WithDecryption: aws.Bool(true),
	})
	if err != nil {
		return fmt.Errorf("failed to get user pool ID from SSM: %v", err)
	}

	userPoolID := *paramOutput.Parameter.Value

	cognitoClient := cognito.NewFromConfig(sdkConfig)

	filter := "cognito:user_status = \"UNCONFIRMED\""

	listUsersOutput, err := cognitoClient.ListUsers(ctx, &cognito.ListUsersInput{
		UserPoolId: aws.String(userPoolID),
		Filter:     aws.String(filter),
	})
	if err != nil {
		return fmt.Errorf("failed to list users: %v", err)
	}

	twoDaysAgo := time.Now().AddDate(0, 0, -2)

	deletedCount := 0

	for _, user := range listUsersOutput.Users {
		if user.UserCreateDate.Before(twoDaysAgo) {

			_, err := cognitoClient.AdminDeleteUser(ctx, &cognito.AdminDeleteUserInput{
				UserPoolId: aws.String(userPoolID),
				Username:   user.Username,
			})
			if err != nil {
				logger.Error("Failed to delete user",
					zap.String("username", *user.Username),
					zap.Error(err),
				)
				continue
			}
			deletedCount++
		}
	}

	logger.Info("Cognito Cleaner executed",
		zap.Int("deleted_users_count", deletedCount),
	)
	return nil
}

func main() {
	defer logger.Sync()
	lambda.Start(func(ctx context.Context) error {
		return logic()
	})
}
