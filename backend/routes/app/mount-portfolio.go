package app

import (
	"encoding/json"
	"net/http"
	"nodofinance/utils/jwt"
	"nodofinance/utils/logger"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/patrickmn/go-cache"
	"go.uber.org/zap"
)

func MountPortfolio(w http.ResponseWriter, r *http.Request, d *dynamodb.Client, dataCache *cache.Cache) {
	ctx := r.Context()

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

	username, exists := idClaims["cognito:username"].(string)
	if !exists {
		logger.Log.Error("Failed to get username from token claims")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	type Response struct {
		Tickers []string `json:"tickers"`
	}

	cacheKey := "tickers_" + username
	if cached, found := dataCache.Get(cacheKey); found {
		if cachedTickers, ok := cached.([]string); ok {
			response := Response{Tickers: cachedTickers}

			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(response); err != nil {
				logger.Log.Error("Error encoding cached response", zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			return
		}
	}

	// Query all ticker records for the user
	queryInput := &dynamodb.QueryInput{
		TableName:              aws.String("nodofinance_table"),
		KeyConditionExpression: aws.String("username = :username AND begins_with(composite_sk, :sk_prefix)"),
		ProjectionExpression:   aws.String("composite_sk, last_update"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":username":  &types.AttributeValueMemberS{Value: username},
			":sk_prefix": &types.AttributeValueMemberS{Value: "TICKER#"},
		},
	}

	result, err := d.Query(ctx, queryInput)
	if err != nil {
		logger.Log.Error("Error querying DynamoDB", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	type Ticker struct {
		Ticker     string
		LastUpdate int64
	}

	var tickers []Ticker

	// Process each ticker record
	for _, item := range result.Items {
		// Extract ticker name from composite_sk (TICKER#AAPL -> AAPL)
		compositeSKAttr, exists := item["composite_sk"]
		if !exists {
			logger.Log.Warn("Missing composite_sk in ticker record")
			continue
		}

		compositeSKMember, ok := compositeSKAttr.(*types.AttributeValueMemberS)
		if !ok {
			logger.Log.Warn("Invalid composite_sk type")
			continue
		}
		compositeSK := compositeSKMember.Value

		ticker := strings.TrimPrefix(compositeSK, "TICKER#")

		if ticker == compositeSK {
			// TrimPrefix didn't work, skip this record
			logger.Log.Warn("Invalid ticker composite_sk format", zap.String("sk", compositeSK))
			continue
		}

		// Extract last_update timestamp
		lastUpdateAttr, exists := item["last_update"]
		if !exists {
			logger.Log.Warn("Missing last_update in ticker record", zap.String("ticker", ticker))
			continue
		}

		lastUpdateStrMember, ok := lastUpdateAttr.(*types.AttributeValueMemberN)
		if !ok {
			logger.Log.Warn("Invalid last_update type")
			continue
		}
		lastUpdateStr := lastUpdateStrMember.Value
		lastUpdate, err := strconv.ParseInt(lastUpdateStr, 10, 64)
		if err != nil {
			logger.Log.Warn("Failed to parse last_update", zap.Error(err), zap.String("ticker", ticker))
			continue
		}

		tickers = append(tickers, Ticker{
			Ticker:     ticker,
			LastUpdate: lastUpdate,
		})
	}

	// Sort by LastUpdate (newest first)
	sort.Slice(tickers, func(i, j int) bool {
		return tickers[i].LastUpdate > tickers[j].LastUpdate
	})

	response := Response{
		Tickers: make([]string, len(tickers)),
	}

	for i, ticker := range tickers {
		response.Tickers[i] = ticker.Ticker
	}

	dataCache.Set(cacheKey, response.Tickers, 10*time.Minute)

	// Send JSON response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Log.Error("Error encoding response", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
