package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"nodofinance/utils/jwt"
	"nodofinance/utils/logger"
	"nodofinance/utils/sanitize"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/patrickmn/go-cache"
	"go.uber.org/zap"
)

func Delete(w http.ResponseWriter, r *http.Request, d *dynamodb.Client, dataCache *cache.Cache) {
	ctx := r.Context()

	ticker := sanitize.Trim(r.URL.Query().Get("ticker"), "u")
	fullPeriod := sanitize.Trim(r.URL.Query().Get("period"), "u")

	if ticker == "" || fullPeriod == "" {
		logger.Log.Error("Ticker or period is empty")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if !sanitize.Ticker(ticker) || !sanitize.Period(fullPeriod) {
		logger.Log.Error("Invalid ticker or period", zap.String("ticker", ticker), zap.String("period", fullPeriod))
		w.WriteHeader(http.StatusBadRequest)
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

	username, exists := idClaims["cognito:username"].(string)
	if !exists {
		logger.Log.Error("Failed to get username from token claims")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	year, err := strconv.Atoi(fullPeriod[:4])
	if err != nil {
		logger.Log.Error("Failed to parse year from period", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	period := fullPeriod[5:]

	financeSK, err := buildFinanceSortKey(ticker, year, period)
	if err != nil {
		logger.Log.Error("Failed to build finance sort key", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Execute atomic delete operation
	err = deleteFinanceRecordAtomic(ctx, d, username, ticker, financeSK)
	if err != nil {
		if errors.Is(err, ErrRecordNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		logger.Log.Error("Failed to delete finance record", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	cacheKey := "tickers_" + username
	dataCache.Delete(cacheKey)

	w.WriteHeader(http.StatusOK)
}

var ErrRecordNotFound = errors.New("record not found")

func deleteFinanceRecordAtomic(ctx context.Context, d *dynamodb.Client, username, ticker, financeSK string) error {
	// Step 1: Check how many finance records exist for this ticker
	queryInput := &dynamodb.QueryInput{
		TableName:              aws.String("nodofinance_table"),
		KeyConditionExpression: aws.String("username = :username AND begins_with(composite_sk, :sk_prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":username":  &types.AttributeValueMemberS{Value: username},
			":sk_prefix": &types.AttributeValueMemberS{Value: fmt.Sprintf("FINANCE#%s#", ticker)},
		},
		Select: types.SelectCount,
		Limit:  aws.Int32(2), // We only need to know if count is 1 or >1
	}

	result, err := d.Query(ctx, queryInput)
	if err != nil {
		return fmt.Errorf("checking finance records count: %w", err)
	}

	// Step 2: Build transaction based on count
	transactItems := []types.TransactWriteItem{
		// Always delete the finance record
		{
			Delete: &types.Delete{
				TableName: aws.String("nodofinance_table"),
				Key: map[string]types.AttributeValue{
					"username":     &types.AttributeValueMemberS{Value: username},
					"composite_sk": &types.AttributeValueMemberS{Value: financeSK},
				},
				ConditionExpression: aws.String("attribute_exists(username)"),
			},
		},
	}

	// If this is the last finance record (count == 1), also delete ticker
	if result.Count == 1 {
		transactItems = append(transactItems, types.TransactWriteItem{
			Delete: &types.Delete{
				TableName: aws.String("nodofinance_table"),
				Key: map[string]types.AttributeValue{
					"username":     &types.AttributeValueMemberS{Value: username},
					"composite_sk": &types.AttributeValueMemberS{Value: fmt.Sprintf("TICKER#%s", ticker)},
				},
				// No condition check for ticker - if it doesn't exist, no problem
			},
		})
	}

	// Step 3: Execute transaction
	_, err = d.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: transactItems,
	})

	if err != nil {
		var conditionalCheckErr *types.TransactionCanceledException
		if errors.As(err, &conditionalCheckErr) {
			// Check if the failure was due to finance record not existing
			for _, reason := range conditionalCheckErr.CancellationReasons {
				if reason.Code != nil && *reason.Code == "ConditionalCheckFailed" {
					return ErrRecordNotFound
				}
			}
		}
		return fmt.Errorf("transaction failed: %w", err)
	}

	return nil
}
