package payments

import (
	"encoding/json"
	"net/http"

	"nodofinance/utils/jwt"
	"nodofinance/utils/logger"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamoTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/billingportal/session"
	"go.uber.org/zap"
)

func Portal(w http.ResponseWriter, r *http.Request, d *dynamodb.Client) {
	ctx := r.Context()

	idTokenCookie, errIT := r.Cookie("nodo_id_token")
	if errIT != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idClaims, err := jwt.GetTokenClaims(idTokenCookie.Value)
	if err != nil {
		logger.Log.Error("Failed to get token claims", zap.Error(err))
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	username, exists := idClaims["cognito:username"].(string)
	if !exists {
		logger.Log.Error("Failed to get username from token claims")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
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
		logger.Log.Error("Error querying DynamoDB", zap.Error(err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if result.Item == nil {
		logger.Log.Warn("No rows found for user, review potential tampering", zap.String("username", username))
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Unmarshal the user data
	var user struct {
		StripeID string `dynamodbav:"stripe_id,omitempty"`
	}

	err = attributevalue.UnmarshalMap(result.Item, &user)
	if err != nil {
		logger.Log.Error("Failed to unmarshal user data", zap.Error(err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if user.StripeID == "" {
		logger.Log.Warn("Stripe ID is not valid", zap.String("username", username))
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(user.StripeID),
		ReturnURL: stripe.String(BASE_URL),
	}

	s, err := session.New(params)
	if err != nil {
		logger.Log.Error("error creating portal session", zap.Error(err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	response := map[string]string{"url": s.URL}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Log.Error("error encoding response", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
