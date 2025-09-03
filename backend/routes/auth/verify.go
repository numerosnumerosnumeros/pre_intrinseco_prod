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

func Verify(w http.ResponseWriter, r *http.Request, c *cognitoidentityprovider.Client) {
	ctx := r.Context()

	var req EmailCodeReq
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

	input := &cognitoidentityprovider.ConfirmSignUpInput{
		ClientId:         aws.String(COGNITO_CLIENT_ID),
		Username:         aws.String(username),
		ConfirmationCode: aws.String(req.ConfirmationCode),
	}

	_, err = c.ConfirmSignUp(ctx, input)
	if err != nil {
		logger.Log.Error("Failed to confirm sign up", zap.Error(err))
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}
