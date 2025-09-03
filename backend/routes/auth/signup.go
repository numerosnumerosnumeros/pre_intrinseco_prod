package auth

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"nodofinance/utils/logger"
	"nodofinance/utils/sanitize"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"go.uber.org/zap"
)

var usernameCounter atomic.Uint64

func generateUniqueUsername() string {
	current := uint16(usernameCounter.Add(1)) % 1296
	timestamp := time.Now().UnixMilli()

	suffix1 := "0123456789abcdefghijklmnopqrstuvwxyz"[current/36]
	suffix2 := "0123456789abcdefghijklmnopqrstuvwxyz"[current%36]

	return "user" + strconv.FormatInt(timestamp, 10) + string(suffix1) + string(suffix2)
}

func SignUp(w http.ResponseWriter, r *http.Request, c *cognitoidentityprovider.Client) {
	ctx := r.Context()

	var req EmailPasswordReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	req.Email = sanitize.Trim(req.Email, "l")

	if !sanitize.Email(req.Email) {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	req.Password = sanitize.Trim(req.Password, "")

	if !sanitize.Password(req.Password) {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	listUsersInput := &cognitoidentityprovider.ListUsersInput{
		UserPoolId: aws.String(COGNITO_USER_POOL_ID),
		Filter:     aws.String("email = \"" + req.Email + "\""),
		Limit:      aws.Int32(1),
	}

	listUsersOutput, err := c.ListUsers(ctx, listUsersInput)
	if err != nil || len(listUsersOutput.Users) > 0 {
		if err != nil {
			logger.Log.Error("Failed to list users", zap.Error(err))
		} else {
			logger.Log.Error("Email already registered", zap.String("email", req.Email))
		}
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	username := generateUniqueUsername()

	input := &cognitoidentityprovider.SignUpInput{
		ClientId: aws.String(COGNITO_CLIENT_ID),
		Username: aws.String(username),
		Password: aws.String(req.Password),
		UserAttributes: []types.AttributeType{
			{Name: aws.String("email"), Value: aws.String(req.Email)},
			{Name: aws.String("custom:subscription_status"), Value: aws.String("inactive")}},
	}

	_, err = c.SignUp(ctx, input)
	if err != nil {
		logger.Log.Error("Failed to sign up user", zap.Error(err))
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}
