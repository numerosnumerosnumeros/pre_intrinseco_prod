package payments

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"nodofinance/routes/auth"
	"nodofinance/utils/logger"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamoTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/webhook"
	"go.uber.org/zap"
)

const MaxBodyBytes = int64(65536)

func Webhooks(w http.ResponseWriter, r *http.Request, c *cognitoidentityprovider.Client, d *dynamodb.Client) {
	ctx := r.Context()

	logger.Log.Info("Stripe Webhook", zap.String("path", r.URL.Path))

	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Log.Error("Error reading request body", zap.Error(err))
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	event, err := webhook.ConstructEvent(payload, r.Header.Get("Stripe-Signature"), ENDPOINT_SECRET)
	if err != nil {
		logger.Log.Error("Error verifying webhook signature", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	switch event.Type {
	case "invoice.payment_succeeded":
		var invoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
			logger.Log.Error("Error parsing webhook JSON", zap.Error(err))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if invoice.Customer == nil {
			logger.Log.Error("Missing customer ID from webhook")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		customerId := invoice.Customer.ID
		if customerId == "" {
			logger.Log.Error("Missing or invalid customer ID from webhook")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		customerEmail := invoice.CustomerEmail
		if customerEmail == "" {
			logger.Log.Error("Missing or invalid customer email from webhook")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		periodEnd := invoice.PeriodEnd
		if periodEnd == 0 {
			logger.Log.Error("Missing or invalid period end from webhook")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		logger.Log.Info("Processing invoice payment",
			zap.String("Customer Email", customerEmail),
			zap.String("Customer ID", customerId),
			zap.Int64("Expires At", periodEnd))

		username, err := auth.GetUsernameByEmail(ctx, c, customerEmail)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		getUserInput := &dynamodb.GetItemInput{
			TableName: aws.String("nodofinance_table"),
			Key: map[string]dynamoTypes.AttributeValue{
				"username":     &dynamoTypes.AttributeValueMemberS{Value: username},
				"composite_sk": &dynamoTypes.AttributeValueMemberS{Value: "METADATA"},
			},
			ProjectionExpression: aws.String("stripe_id"),
		}

		result, err := d.GetItem(ctx, getUserInput)
		if err != nil {
			logger.Log.Error("Error querying DynamoDB in webhook", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if result.Item == nil {
			// *
			// **
			// ***
			// ****
			// ***** new subscription
			putUserInput := &dynamodb.PutItemInput{
				TableName: aws.String("nodofinance_table"),
				Item: map[string]dynamoTypes.AttributeValue{
					"username":     &dynamoTypes.AttributeValueMemberS{Value: username},
					"composite_sk": &dynamoTypes.AttributeValueMemberS{Value: "METADATA"},
					"stripe_id":    &dynamoTypes.AttributeValueMemberS{Value: customerId},
					"expires_date": &dynamoTypes.AttributeValueMemberN{Value: fmt.Sprintf("%d", periodEnd)},
					"ctokens":      &dynamoTypes.AttributeValueMemberN{Value: "0"},
				},
			}

			_, err := d.PutItem(ctx, putUserInput)
			if err != nil {
				logger.Log.Error("Error inserting new user into DynamoDB", zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			logger.Log.Info("New user subscription created",
				zap.String("email", customerEmail),
				zap.Int64("Expires At", periodEnd))
		} else {
			// *
			// **
			// ***
			// ****
			// ***** renewed subscription

			// Unmarshal existing user data
			var user struct {
				StripeID string `dynamodbav:"stripe_id,omitempty"`
			}

			err = attributevalue.UnmarshalMap(result.Item, &user)
			if err != nil {
				logger.Log.Error("Failed to unmarshal user data", zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			if user.StripeID == "" {
				logger.Log.Warn("Stripe ID is not valid", zap.String("username", username))
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			if user.StripeID != customerId {
				logger.Log.Warn("Webhook Stripe ID mismatch for user", zap.String("username", username), zap.String("customerId", customerId))
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			updateUserInput := &dynamodb.UpdateItemInput{
				TableName: aws.String("nodofinance_table"),
				Key: map[string]dynamoTypes.AttributeValue{
					"username":     &dynamoTypes.AttributeValueMemberS{Value: username},
					"composite_sk": &dynamoTypes.AttributeValueMemberS{Value: "METADATA"},
				},
				UpdateExpression: aws.String("SET expires_date = :expires, ctokens = :ctokens"),
				ExpressionAttributeValues: map[string]dynamoTypes.AttributeValue{
					":expires":           &dynamoTypes.AttributeValueMemberN{Value: fmt.Sprintf("%d", periodEnd)},
					":ctokens":           &dynamoTypes.AttributeValueMemberN{Value: "0"},
					":current_stripe_id": &dynamoTypes.AttributeValueMemberS{Value: customerId},
				},
				ConditionExpression: aws.String("stripe_id = :current_stripe_id"), // More specific than attribute_exists
			}

			_, err := d.UpdateItem(ctx, updateUserInput)
			if err != nil {
				// Check if it's a condition check failure (equivalent to rowsAffected == 0)
				var conditionalCheckFailedErr *dynamoTypes.ConditionalCheckFailedException
				if errors.As(err, &conditionalCheckFailedErr) {
					logger.Log.Error("No user found for subscription update", zap.String("username", username), zap.String("email", customerEmail))
					w.WriteHeader(http.StatusNotFound)
					return
				}

				logger.Log.Error("Error updating user subscription", zap.Error(err), zap.String("username", username), zap.String("email", customerEmail))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			logger.Log.Info("User subscription updated",
				zap.String("email", customerEmail),
				zap.Int64("Expires At", periodEnd))
		}

	default:
		logger.Log.Info("Unhandled webhook event type", zap.Any("type", event.Type))
	}

	logger.Log.Info("Webhook processed successfully",
		zap.String("customer_id", event.Data.Object["customer"].(string)),
		zap.String("customer_email", event.Data.Object["customer_email"].(string)),
		zap.Int64("period_end", event.Data.Object["period_end"].(int64)))

	w.WriteHeader(http.StatusOK)
}
