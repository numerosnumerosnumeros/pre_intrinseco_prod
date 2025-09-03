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

func Resend(w http.ResponseWriter, r *http.Request, c *cognitoidentityprovider.Client) {
	ctx := r.Context()

	var req EmailReq
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

	username, err := GetUsernameByEmail(ctx, c, req.Email)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	input := &cognitoidentityprovider.ResendConfirmationCodeInput{
		ClientId: aws.String(COGNITO_CLIENT_ID),
		Username: aws.String(username),
	}

	_, err = c.ResendConfirmationCode(ctx, input)
	if err != nil {
		logger.Log.Error("Failed to resend confirmation code", zap.Error(err))
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}
