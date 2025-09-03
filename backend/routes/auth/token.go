package auth

import (
	"encoding/json"
	"net/http"
	"time"

	"nodofinance/utils/csrf"
	"nodofinance/utils/jwt"
	"nodofinance/utils/logger"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	cognitoTypes "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamoTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"go.uber.org/zap"
)

func Token(w http.ResponseWriter, r *http.Request, c *cognitoidentityprovider.Client, d *dynamodb.Client) {
	ctx := r.Context()

	var idToken string
	var accessToken string
	var refreshToken string
	var csrfToken string

	cookies := r.Cookies()
	for _, cookie := range cookies {
		switch cookie.Name {
		case "nodo_id_token":
			idToken = cookie.Value
		case "nodo_access_token":
			accessToken = cookie.Value
		case "nodo_refresh_token":
			refreshToken = cookie.Value
		case "nodo_csrf_token":
			csrfToken = cookie.Value
		}
	}

	if refreshToken == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if idToken != "" && accessToken != "" && csrfToken != "" {
		// *
		// ***
		// *****
		// COOKIES RECEIVED -> CHECK IF THEY ARE VALID
		idTokenValid, idTokenErr := jwt.ValidateToken(idToken, false)
		accessTokenValid, accessTokenErr := jwt.ValidateToken(accessToken, false)

		if !idTokenValid || !accessTokenValid || idTokenErr != nil || accessTokenErr != nil || !csrf.Validate(r) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		claims, err := jwt.GetTokenClaims(idToken)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		subscriptionStatus, ok := claims["custom:subscription_status"].(string)
		if !ok {
			logger.Log.Error("Failed to parse subscription status")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		response := map[string]bool{
			"plan": subscriptionStatus == "active",
		}

		jsonResponse, err := json.Marshal(response)
		if err != nil {
			logger.Log.Error("Failed to create JSON response")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "private, max-age=1800") // Cache for 30 minutes
		w.WriteHeader(http.StatusOK)
		w.Write(jsonResponse)
		return
	} else {
		// *
		// ***
		// *****
		// COOKIES MISSING -> TRY TO REFRESH
		refreshInput := &cognitoidentityprovider.InitiateAuthInput{
			AuthFlow: cognitoTypes.AuthFlowTypeRefreshToken,
			AuthParameters: map[string]string{
				"REFRESH_TOKEN": refreshToken,
			},
			ClientId: aws.String(COGNITO_CLIENT_ID),
		}

		refreshResult, err := c.InitiateAuth(ctx, refreshInput)
		if err != nil {
			logger.Log.Error("Error refreshing token -> removing refresh token", zap.Error(err))
			http.SetCookie(w, &http.Cookie{
				Name:     "nodo_refresh_token",
				Value:    "",
				Expires:  time.Unix(0, 0),
				Path:     "/",
				Secure:   true,
				HttpOnly: true,
				SameSite: http.SameSiteStrictMode,
			})
			w.WriteHeader(http.StatusOK)
			return
		}

		claims, err := jwt.GetTokenClaims(*refreshResult.AuthenticationResult.IdToken)
		if err != nil {
			logger.Log.Error("Error getting token claims", zap.Error(err))
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		subscriptionStatus, ok := claims["custom:subscription_status"].(string)
		if !ok {
			logger.Log.Error("Failed to parse subscription status")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		username, ok := claims["cognito:username"].(string)
		if !ok {
			logger.Log.Error("Failed to parse username")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		isCognitoPremium := false
		isDBPremium := false
		isPremiumUser := false

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

		// Check if item exists
		if result.Item == nil {
			// *
			// ***
			// *****
			// SCENARIO 1: never been premium
			if isCognitoPremium {
				logger.Log.Warn("User is premium in Cognito but not in DB", zap.String("username", username))

				err = UpdateCognitoSubscriptionStatus(ctx, c, username, "inactive")
				if err != nil {
					logger.Log.Error("Failed to update Cognito subscription status for potential tampering", zap.Error(err))
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}

				// Get new tokens
				refreshResult, err = c.InitiateAuth(ctx, refreshInput)
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

			// Unmarshal the user data
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
				isPremiumUser = true
			}

			// If they match, no action required, else -> update Cognito
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
				refreshResult, err = c.InitiateAuth(ctx, refreshInput)
				if err != nil {
					logger.Log.Error("Failed to re-initiate auth after potential tampering", zap.Error(err))
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
			}
		}

		ar := refreshResult.AuthenticationResult
		csrfToken, err := csrf.Generate(*ar.IdToken)
		if err != nil {
			logger.Log.Error("Failed to generate CSRF token", zap.Error(err))
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

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
			Name:     "nodo_csrf_token",
			Value:    csrfToken,
			Expires:  dayExpiration,
			Path:     "/",
			Secure:   true,
			HttpOnly: false,
			SameSite: http.SameSiteStrictMode,
		})

		response := map[string]bool{
			"plan": isPremiumUser,
		}

		jsonResponse, err := json.Marshal(response)
		if err != nil {
			logger.Log.Error("Failed to create JSON response")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "private, max-age=1800") // Cache for 30 minutes
		w.WriteHeader(http.StatusOK)
		w.Write(jsonResponse)
		return
	}
}
