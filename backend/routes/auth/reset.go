package auth

import (
	"encoding/json"
	"net/http"
	"nodofinance/utils/logger"
	"nodofinance/utils/sanitize"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"go.uber.org/zap"
)

func Reset(w http.ResponseWriter, r *http.Request, c *cognitoidentityprovider.Client) {
	ctx := r.Context()

	var req EmailPasswordCodeReq
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

	req.ConfirmationCode = sanitize.Trim(req.ConfirmationCode, "")

	if !sanitize.Code(req.ConfirmationCode) {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	username, err := GetUsernameByEmail(ctx, c, req.Email)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	input := &cognitoidentityprovider.ConfirmForgotPasswordInput{
		ClientId:         aws.String(COGNITO_CLIENT_ID),
		Username:         aws.String(username),
		ConfirmationCode: aws.String(req.ConfirmationCode),
		Password:         aws.String(req.Password),
	}

	_, err = c.ConfirmForgotPassword(ctx, input)
	if err != nil {
		logger.Log.Error("Failed to confirm forgot password", zap.Error(err))
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}
