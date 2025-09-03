package auth

import (
	"context"
	"fmt"
	"nodofinance/utils/env"
	"nodofinance/utils/logger"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"go.uber.org/zap"
)

type EmailReq struct {
	Email string `json:"email"`
}

type EmailPasswordReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type EmailCodeReq struct {
	Email            string `json:"email"`
	ConfirmationCode string `json:"confirmationCode"`
}

type EmailPasswordCodeReq struct {
	Email            string `json:"email"`
	Password         string `json:"password"`
	ConfirmationCode string `json:"confirmationCode"`
}

var (
	COGNITO_CLIENT_ID    string
	COGNITO_USER_POOL_ID string
)

func init() {
	env.RegisterValidator(validateVar)
}

func validateVar() error {
	var exists bool

	COGNITO_CLIENT_ID, exists = env.Get("COGNITO_CLIENT_ID")
	if !exists {
		return fmt.Errorf("missing required environment variable: COGNITO_CLIENT_ID")
	}

	COGNITO_USER_POOL_ID, exists = env.Get("COGNITO_USER_POOL_ID")
	if !exists {
		return fmt.Errorf("missing required environment variable: COGNITO_USER_POOL_ID")
	}

	return nil
}

func GetUsernameByEmail(ctx context.Context, c *cognitoidentityprovider.Client, email string) (string, error) {
	listUsersInput := &cognitoidentityprovider.ListUsersInput{
		UserPoolId: aws.String(COGNITO_USER_POOL_ID),
		Filter:     aws.String("email = \"" + email + "\""),
		Limit:      aws.Int32(1),
	}

	listUsersOutput, err := c.ListUsers(ctx, listUsersInput)
	if err != nil {
		logger.Log.Error("Failed to list users", zap.Error(err))
		return "", err
	}

	if len(listUsersOutput.Users) == 0 {
		return "", fmt.Errorf("user not found")
	}

	if listUsersOutput.Users[0].Username == nil || *listUsersOutput.Users[0].Username == "" {
		return "", fmt.Errorf("username is empty")
	}

	username := *listUsersOutput.Users[0].Username
	return username, nil
}

func UpdateCognitoSubscriptionStatus(ctx context.Context, c *cognitoidentityprovider.Client, username string, status string) error {
	_, err := c.AdminUpdateUserAttributes(ctx, &cognitoidentityprovider.AdminUpdateUserAttributesInput{
		UserPoolId: aws.String(COGNITO_USER_POOL_ID),
		Username:   aws.String(username),
		UserAttributes: []types.AttributeType{
			{
				Name:  aws.String("custom:subscription_status"),
				Value: aws.String(status),
			},
		},
	})

	if err != nil {
		logger.Log.Error("Failed to update user attributes", zap.Error(err))
		return err
	}

	return nil
}
