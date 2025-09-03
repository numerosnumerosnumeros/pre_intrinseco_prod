package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"nodofinance/utils/csrf"
	"nodofinance/utils/jwt"
	"nodofinance/utils/logger"
	"nodofinance/utils/sanitize"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	cognitoTypes "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamoTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"go.uber.org/zap"
)

func SignIn(w http.ResponseWriter, r *http.Request, c *cognitoidentityprovider.Client, d *dynamodb.Client) {
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

	username, err := GetUsernameByEmail(ctx, c, req.Email)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	input := &cognitoidentityprovider.InitiateAuthInput{
		ClientId: aws.String(COGNITO_CLIENT_ID),
		AuthFlow: cognitoTypes.AuthFlowTypeUserPasswordAuth,
		AuthParameters: map[string]string{
			"USERNAME": username,
			"PASSWORD": req.Password,
		},
	}

	mainInitiateAuthResult, err := c.InitiateAuth(ctx, input)
	if err != nil {
		var userNotConfirmedErr *cognitoTypes.UserNotConfirmedException
		if errors.As(err, &userNotConfirmedErr) {
			http.Error(w, "User not confirmed", http.StatusConflict)
			return
		}

		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// INITIATE_AUTH SUCCESSFUL -> verify db-cognito integrity
	ar := mainInitiateAuthResult.AuthenticationResult

	claims, err := jwt.GetTokenClaims(*ar.IdToken)
	if err != nil {
		logger.Log.Error("Error getting token claims", zap.Error(err))
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	isCognitoPremium := false
	isDBPremium := false

	subscriptionStatus, ok := claims["custom:subscription_status"].(string)
	if !ok {
		logger.Log.Error("Failed to parse subscription status")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if subscriptionStatus == "active" {
		isCognitoPremium = true
	}

	getUserInput := &dynamodb.GetItemInput{
		TableName: aws.String("nodofinance_table"),
		Key: map[string]dynamoTypes.AttributeValue{
			"username":     &dynamoTypes.AttributeValueMemberS{Value: username},
			"composite_sk": &dynamoTypes.AttributeValueMemberS{Value: "METADATA"},
		},
		ProjectionExpression: aws.String("expires_date"),
	}

	result, err := d.GetItem(ctx, getUserInput)
	if err != nil {
		logger.Log.Error("DynamoDB query failed", zap.Error(err), zap.String("username", username))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if result.Item == nil {
		// *
		// ***
		// *****
		// SCENARIO 1: never been premium
		if isCognitoPremium {
			logger.Log.Warn("User is premium in Cognito but not in DB. Updating Cognito and refreshing cookies", zap.String("username", username))

			err = UpdateCognitoSubscriptionStatus(ctx, c, username, "inactive")
			if err != nil {
				logger.Log.Error("Failed to update Cognito subscription status for potential tampering", zap.Error(err))
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			// Get new tokens
			mainInitiateAuthResult, err = c.InitiateAuth(ctx, input)
			if err != nil {
				logger.Log.Error("Failed to re-initiate auth after potential tampering", zap.Error(err))
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}
	} else {
		// *
		// ***
		// *****
		// SCENARIO 2: has been or is premium in db
		var user struct {
			ExpiresDate *int64 `dynamodbav:"expires_date,omitempty"`
		}

		err = attributevalue.UnmarshalMap(result.Item, &user)
		if err != nil {
			logger.Log.Error("Failed to unmarshal user data", zap.Error(err))
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if user.ExpiresDate == nil || *user.ExpiresDate == 0 {
			logger.Log.Warn("User has no expiration date in DB", zap.String("username", username))
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		expiresDate := time.Unix(*user.ExpiresDate, 0)

		if expiresDate.After(time.Now()) {
			isDBPremium = true
		}

		// If they match, not action required, else -> update Cognito
		if isCognitoPremium != isDBPremium {
			logger.Log.Warn("Cognito and DB subscription status mismatch", zap.String("username", username), zap.Bool("isCognitoPremium", isCognitoPremium), zap.Bool("isDBPremium", isDBPremium))

			if isDBPremium {
				err = UpdateCognitoSubscriptionStatus(ctx, c, username, "active")
				if err != nil {
					logger.Log.Error("Failed to update Cognito subscription status", zap.Error(err))
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}
			} else {
				err = UpdateCognitoSubscriptionStatus(ctx, c, username, "inactive")
				if err != nil {
					logger.Log.Error("Failed to update Cognito subscription status", zap.Error(err))
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}
			}

			// Get new tokens
			mainInitiateAuthResult, err = c.InitiateAuth(ctx, input)
			if err != nil {
				logger.Log.Error("Failed to re-initiate auth after potential tampering", zap.Error(err))
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}
	}

	ar = mainInitiateAuthResult.AuthenticationResult
	csrfToken, err := csrf.Generate(*ar.IdToken)
	if err != nil {
		logger.Log.Error("Failed to generate CSRF token", zap.Error(err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	yearExpiration := time.Now().AddDate(1, 0, 0)
	dayExpiration := time.Now().AddDate(0, 0, 1)

	http.SetCookie(w, &http.Cookie{
		Name:     "nodo_id_token",
		Value:    *ar.IdToken,
		Expires:  dayExpiration,
		Path:     "/",
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "nodo_access_token",
		Value:    *ar.AccessToken,
		Expires:  dayExpiration,
		Path:     "/",
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "nodo_refresh_token",
		Value:    *ar.RefreshToken,
		Expires:  yearExpiration,
		Path:     "/",
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "nodo_csrf_token",
		Value:    csrfToken,
		Expires:  dayExpiration,
		Path:     "/",
		Secure:   true,
		HttpOnly: false,
		SameSite: http.SameSiteStrictMode,
	})

	w.WriteHeader(http.StatusOK)
}
