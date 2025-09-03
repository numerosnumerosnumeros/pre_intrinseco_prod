package payments

import (
	"errors"
	"fmt"
	"net/http"
	"nodofinance/routes/auth"
	"nodofinance/utils/csrf"
	"nodofinance/utils/jwt"
	"nodofinance/utils/logger"
	"nodofinance/utils/sanitize"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamoTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/patrickmn/go-cache"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/customer"
	"github.com/stripe/stripe-go/v81/subscription"
	"go.uber.org/zap"
)

func Refresh(w http.ResponseWriter, r *http.Request, c *cognitoidentityprovider.Client, d *dynamodb.Client, dataCache *cache.Cache) {
	ctx := r.Context()

	token := r.URL.Query().Get("token")

	if token == "" || !sanitize.Hex(token) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	refreshTokenCookie, errIT := r.Cookie("nodo_refresh_token")
	if errIT != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	idTokenCookie, errIT := r.Cookie("nodo_id_token")
	if errIT != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	idClaims, err := jwt.GetTokenClaims(idTokenCookie.Value)
	if err != nil {
		logger.Log.Error("Failed to get token claims", zap.Error(err))
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	email, exists := idClaims["email"].(string)
	if !exists {
		logger.Log.Error("Failed to get email from token claims")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	cacheKey := "stripe_token_" + email
	cacheItem, found := dataCache.Get(cacheKey)
	if !found {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if token != cacheItem {
		logger.Log.Error("Token mismatch", zap.String("expected", cacheItem.(string)), zap.String("received", token))
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	dataCache.Delete(cacheKey)

	username, exists := idClaims["cognito:username"].(string)
	if !exists {
		logger.Log.Error("Failed to get username from token claims")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// *
	// **
	// ***
	// ****
	// ***** Directly check Stripe API for subscription status (avoid data race)
	// -> if it fails, fallback to SQL
	// -> if it succeeds, update Cognito and database
	triggerFallback := false
	customerID := ""
	subscriptionStatus := false
	currentPeriodEnd := int64(0)

	customerParams := &stripe.CustomerListParams{
		Email: stripe.String(email),
	}
	customerParams.Limit = stripe.Int64(1)
	customerResult := customer.List(customerParams)

	for customerResult.Next() {
		customerID = customerResult.Customer().ID
	}
	if err := customerResult.Err(); err != nil {
		logger.Log.Error("failed to list customers", zap.Error(err))
		triggerFallback = true
	}

	if customerID == "" {
		logger.Log.Error("failed to get customer ID", zap.String("email", email))
		triggerFallback = true
	}

	if !triggerFallback {
		subscriptionParams := &stripe.SubscriptionListParams{
			Customer: stripe.String(customerID),
		}
		subscriptionParams.Limit = stripe.Int64(1)
		subscriptionResult := subscription.List(subscriptionParams)

		for subscriptionResult.Next() {
			s := subscriptionResult.Subscription()
			if s.Status == "active" {
				subscriptionStatus = true
				currentPeriodEnd = s.CurrentPeriodEnd
			}
		}
		if err := subscriptionResult.Err(); err != nil {
			logger.Log.Error("failed to list subscriptions", zap.Error(err), zap.String("email", email))
			triggerFallback = true
		}
	}

	if !triggerFallback && subscriptionStatus {
		if currentPeriodEnd <= time.Now().Unix() {
			logger.Log.Warn("Stripe subscription not active but passed validations",
				zap.String("email", email),
				zap.Int64("currentPeriodEnd", currentPeriodEnd))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		err = auth.UpdateCognitoSubscriptionStatus(ctx, c, username, "active")
		if err != nil {
			logger.Log.Error("Failed to update Cognito subscription status", zap.Error(err))
			triggerFallback = true
		}
	}

	if !triggerFallback && subscriptionStatus {
		refreshInput := &cognitoidentityprovider.InitiateAuthInput{
			AuthFlow: types.AuthFlowTypeRefreshToken,
			AuthParameters: map[string]string{
				"REFRESH_TOKEN": refreshTokenCookie.Value,
			},
			ClientId: aws.String(auth.COGNITO_CLIENT_ID),
		}

		refreshResult, err := c.InitiateAuth(ctx, refreshInput)
		if err != nil {
			logger.Log.Error("Failed to refresh tokens", zap.Error(err), zap.String("username", username), zap.String("email", email))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		ar := refreshResult.AuthenticationResult

		csrfToken, err := csrf.Generate(*ar.IdToken)
		if err != nil {
			logger.Log.Error("Failed to generate CSRF token in SQL fallback", zap.Error(err), zap.String("username", username), zap.String("email", email))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		var user struct {
			StripeID    string `dynamodbav:"stripe_id,omitempty"`
			ExpiresDate *int64 `dynamodbav:"expires_date,omitempty"`
			CTokens     *int64 `dynamodbav:"ctokens,omitempty"`
		}

		if !triggerFallback {
			getUserInput := &dynamodb.GetItemInput{
				TableName: aws.String("nodofinance_table"),
				Key: map[string]dynamoTypes.AttributeValue{
					"username":     &dynamoTypes.AttributeValueMemberS{Value: username},
					"composite_sk": &dynamoTypes.AttributeValueMemberS{Value: "METADATA"},
				},
			}

			result, err := d.GetItem(ctx, getUserInput)
			if err != nil {
				logger.Log.Error("Error querying DynamoDB, review webhook hit", zap.Error(err), zap.String("username", username))
				triggerFallback = true
			} else if result.Item == nil {
				logger.Log.Info("No rows found for user review webhook hit -> manually updating database", zap.String("username", username))

				putUserInput := &dynamodb.PutItemInput{
					TableName: aws.String("nodofinance_table"),
					Item: map[string]dynamoTypes.AttributeValue{
						"username":     &dynamoTypes.AttributeValueMemberS{Value: username},
						"composite_sk": &dynamoTypes.AttributeValueMemberS{Value: "METADATA"},
						"stripe_id":    &dynamoTypes.AttributeValueMemberS{Value: customerID},
						"expires_date": &dynamoTypes.AttributeValueMemberN{Value: fmt.Sprintf("%d", currentPeriodEnd)},
						"ctokens":      &dynamoTypes.AttributeValueMemberN{Value: "0"},
					},
				}

				_, err := d.PutItem(ctx, putUserInput)
				if err != nil {
					logger.Log.Error("Error inserting new user into database", zap.Error(err))
					triggerFallback = true
				} else {
					logger.Log.Info("New user subscription manually created.",
						zap.String("username", username),
						zap.String("email", email),
						zap.Int64("Expires At", currentPeriodEnd))
				}
			} else {
				// Unmarshal existing user data
				err = attributevalue.UnmarshalMap(result.Item, &user)
				if err != nil {
					logger.Log.Error("Failed to unmarshal user data", zap.Error(err))
					triggerFallback = true
				} else if user.StripeID == "" || user.ExpiresDate == nil || *user.ExpiresDate == 0 {
					logger.Log.Error("Stripe ID or expiresTimestamp is not valid", zap.String("username", username))
					triggerFallback = true
				} else if user.StripeID != customerID {
					logger.Log.Error("Webhook failed and Stripe ID mismatch", zap.String("expected", user.StripeID), zap.String("received", customerID),
						zap.String("username", username), zap.String("email", email))
					triggerFallback = true
				} else if *user.ExpiresDate < currentPeriodEnd {
					updateUserInput := &dynamodb.UpdateItemInput{
						TableName: aws.String("nodofinance_table"),
						Key: map[string]dynamoTypes.AttributeValue{
							"username":     &dynamoTypes.AttributeValueMemberS{Value: username},
							"composite_sk": &dynamoTypes.AttributeValueMemberS{Value: "METADATA"},
						},
						UpdateExpression: aws.String("SET expires_date = :expires, ctokens = :ctokens"),
						ExpressionAttributeValues: map[string]dynamoTypes.AttributeValue{
							":expires":           &dynamoTypes.AttributeValueMemberN{Value: fmt.Sprintf("%d", currentPeriodEnd)},
							":ctokens":           &dynamoTypes.AttributeValueMemberN{Value: "0"},
							":current_stripe_id": &dynamoTypes.AttributeValueMemberS{Value: customerID},
						},
						ConditionExpression: aws.String("stripe_id = :current_stripe_id"),
					}

					_, err := d.UpdateItem(ctx, updateUserInput)
					if err != nil {
						// Check if it's a condition check failure (equivalent to rowsAffected == 0)
						var conditionalCheckFailedErr *dynamoTypes.ConditionalCheckFailedException
						if errors.As(err, &conditionalCheckFailedErr) {
							logger.Log.Error("User not found for subscription update", zap.String("username", username), zap.String("email", email))
							triggerFallback = true
						} else {
							logger.Log.Error("Error updating timestamp", zap.Error(err), zap.String("username", username), zap.String("email", email))
							triggerFallback = true
						}
					} else {
						logger.Log.Info("Webhook failed. Recurring subscription updated",
							zap.String("username", username),
							zap.String("email", email),
							zap.Int64("expires", currentPeriodEnd))
					}
				}
			}

			if !triggerFallback {
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

				logger.Log.Info("SUBSCRIBED: Stripe subscription active, tokens refreshed",
					zap.String("username", username),
					zap.String("email", email),
					zap.Int64("expires", currentPeriodEnd))

				w.WriteHeader(http.StatusOK)
				return
			}
		}
	}

	// *
	// **
	// ***
	// ****
	// ***** Fall back to SQL if stripe api checks fail
	// -> check if webhook worked instead
	if triggerFallback {
		logger.Log.Error("Triggering SQL fallback authentication. Stripe API failed", zap.String("username", username), zap.String("email", email))

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
			logger.Log.Error("Error querying DynamoDB during fallback", zap.Error(err), zap.String("username", username), zap.String("email", email))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if result.Item == nil {
			logger.Log.Error("User not found during fallback", zap.String("username", username), zap.String("email", email))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Unmarshal user data
		var user struct {
			ExpiresDate *int64 `dynamodbav:"expires_date,omitempty"`
		}

		err = attributevalue.UnmarshalMap(result.Item, &user)
		if err != nil {
			logger.Log.Error("Failed to unmarshal user data during fallback", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if user.ExpiresDate == nil {
			logger.Log.Warn("expiresTimestamp is not valid", zap.String("username", username), zap.String("email", email))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if *user.ExpiresDate < time.Now().Unix() {
			logger.Log.Warn("Subscription expired but reached refresh", zap.String("username", username), zap.Int64("expiresTimestamp", *user.ExpiresDate), zap.String("email", email))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		err = auth.UpdateCognitoSubscriptionStatus(ctx, c, username, "active")
		if err != nil {
			logger.Log.Error("Failed to update Cognito subscription status in SQL fallback", zap.Error(err), zap.String("username", username), zap.String("email", email))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		refreshInput := &cognitoidentityprovider.InitiateAuthInput{
			AuthFlow: types.AuthFlowTypeRefreshToken,
			AuthParameters: map[string]string{
				"REFRESH_TOKEN": refreshTokenCookie.Value,
			},
			ClientId: aws.String(auth.COGNITO_CLIENT_ID),
		}

		refreshResult, err := c.InitiateAuth(ctx, refreshInput)
		if err != nil {
			logger.Log.Error("Failed to refresh tokens in SQL fallback", zap.Error(err), zap.String("username", username), zap.String("email", email))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		ar := refreshResult.AuthenticationResult

		csrfToken, err := csrf.Generate(*ar.IdToken)
		if err != nil {
			logger.Log.Error("Failed to generate CSRF token in SQL fallback", zap.Error(err), zap.String("username", username), zap.String("email", email))
			w.WriteHeader(http.StatusInternalServerError)
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

		logger.Log.Info("SUBSCRIBED: SQL fallback successful, tokens refreshed",
			zap.String("username", username),
			zap.String("email", email),
			zap.Int64("expires", *user.ExpiresDate))

		w.WriteHeader(http.StatusOK)
		return
	} else {
		// catch all, should not happen
		logger.Log.Error("ShouldNeverHappen: Failed to refresh tokens and fallback did not work. Review bug (potential error without setting triggerFallback properly)", zap.String("username", username), zap.String("email", email))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
