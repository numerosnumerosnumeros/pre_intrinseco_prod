package payments

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"nodofinance/utils/jwt"
	"nodofinance/utils/logger"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamoTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/patrickmn/go-cache"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/checkout/session"
	"go.uber.org/zap"
)

func Checkout(w http.ResponseWriter, r *http.Request, d *dynamodb.Client, dataCache *cache.Cache) {
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

	email, exists := idClaims["email"].(string)
	if !exists {
		logger.Log.Error("Failed to get email from token claims")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	randomBytes := make([]byte, 16) // 16 bytes
	_, err = rand.Read(randomBytes)
	if err != nil {
		http.Error(w, "Failed to generate random bytes", http.StatusInternalServerError)
		return
	}

	token := hex.EncodeToString(randomBytes)

	cacheKey := "stripe_token_" + email
	dataCache.Set(cacheKey, token, 10*time.Minute)

	successURL := BASE_URL + "/welcome?token=" + token

	checkoutParams := &stripe.CheckoutSessionParams{
		SuccessURL: stripe.String(successURL),
		CancelURL:  stripe.String(BASE_URL),
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		AutomaticTax: &stripe.CheckoutSessionAutomaticTaxParams{
			Enabled: stripe.Bool(true),
		},
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(PRICE_ID),
				Quantity: stripe.Int64(1),
			},
		},
		CustomerEmail: stripe.String(email),
	}

	getUserInput := &dynamodb.GetItemInput{
		TableName: aws.String("nodofinance_table"),
		Key: map[string]dynamoTypes.AttributeValue{
			"username":     &dynamoTypes.AttributeValueMemberS{Value: username},
			"composite_sk": &dynamoTypes.AttributeValueMemberS{Value: "METADATA"},
		},
		ProjectionExpression: aws.String("expires_date, stripe_id"),
	}

	result, err := d.GetItem(ctx, getUserInput)
	if err != nil {
		logger.Log.Error("Error querying DynamoDB", zap.Error(err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Check if user exists
	if result.Item != nil {
		// Unmarshal the user data
		var user struct {
			StripeID    string `dynamodbav:"stripe_id,omitempty"`
			ExpiresDate *int64 `dynamodbav:"expires_date,omitempty"`
		}

		err = attributevalue.UnmarshalMap(result.Item, &user)
		if err != nil {
			logger.Log.Error("Failed to unmarshal user data", zap.Error(err))
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if user.StripeID == "" || user.ExpiresDate == nil || *user.ExpiresDate == 0 {
			logger.Log.Warn("User has no expiration date or stripe ID in DB", zap.String("username", username))
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		expiresDate := time.Unix(*user.ExpiresDate, 0)

		if expiresDate.After(time.Now()) {
			logger.Log.Warn("User already has an active subscription but tried to checkout again", zap.String("username", username))
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		stripeIDfromDB := user.StripeID

		checkoutParams.CustomerEmail = nil
		checkoutParams.Customer = stripe.String(stripeIDfromDB)
	}

	s, err := session.New(checkoutParams)
	if err != nil {
		logger.Log.Error("error creating checkout session", zap.Error(err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return

	}

	response := struct {
		SessionID string `json:"sessionId"`
	}{
		SessionID: s.ID,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Log.Error("error encoding response", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
