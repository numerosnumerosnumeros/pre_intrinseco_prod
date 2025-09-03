package app

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"nodofinance/utils/jwt"
	"nodofinance/utils/logger"
	"nodofinance/utils/sanitize"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamoTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"go.uber.org/zap"
)

type FinancialData struct {
	CurrentAssets              *int64   `json:"current_assets,omitempty"`
	NonCurrentAssets           *int64   `json:"non_current_assets,omitempty"`
	CashAndEquivalents         *int64   `json:"cash_and_equivalents,omitempty"`
	CurrentLiabilities         *int64   `json:"current_liabilities,omitempty"`
	NonCurrentLiabilities      *int64   `json:"non_current_liabilities,omitempty"`
	Revenue                    *int64   `json:"revenue,omitempty"`
	NetIncome                  *int64   `json:"net_income,omitempty"`
	Eps                        *float64 `json:"eps,omitempty"`
	CashFlowFromOperations     *int64   `json:"cash_flow_from_operations,omitempty"`
	CashFlowFromInvesting      *int64   `json:"cash_flow_from_investing,omitempty"`
	CashFlowFromFinancing      *int64   `json:"cash_flow_from_financing,omitempty"`
	CurrentAssetsPrev          *int64   `json:"current_assets_prev,omitempty"`
	NonCurrentAssetsPrev       *int64   `json:"non_current_assets_prev,omitempty"`
	CashAndEquivalentsPrev     *int64   `json:"cash_and_equivalents_prev,omitempty"`
	CurrentLiabilitiesPrev     *int64   `json:"current_liabilities_prev,omitempty"`
	NonCurrentLiabilitiesPrev  *int64   `json:"non_current_liabilities_prev,omitempty"`
	RevenuePrev                *int64   `json:"revenue_prev,omitempty"`
	NetIncomePrev              *int64   `json:"net_income_prev,omitempty"`
	EpsPrev                    *float64 `json:"eps_prev,omitempty"`
	CashFlowFromOperationsPrev *int64   `json:"cash_flow_from_operations_prev,omitempty"`
	CashFlowFromInvestingPrev  *int64   `json:"cash_flow_from_investing_prev,omitempty"`
	CashFlowFromFinancingPrev  *int64   `json:"cash_flow_from_financing_prev,omitempty"`
}

type Response struct {
	Currency      string        `json:"currency,omitempty"`
	Analysis      string        `json:"analysis,omitempty"`
	Period        string        `json:"period,omitempty"`
	FinancialData FinancialData `json:"financial_data,omitempty"`
	Cursor        string        `json:"cursor,omitempty"`
}

func MountTicker(w http.ResponseWriter, r *http.Request, d *dynamodb.Client) {
	ctx := r.Context()

	ticker := sanitize.Trim(r.URL.Query().Get("ticker"), "u")
	cursor := sanitize.Trim(r.URL.Query().Get("cursor"), "")

	rWithCursor := false
	if cursor != "" {
		if !sanitize.Cursor(cursor) {
			logger.Log.Error("Invalid cursor")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		rWithCursor = true
	}

	if ticker == "" {
		logger.Log.Error("Ticker is empty")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if !sanitize.Ticker(ticker) {
		logger.Log.Error("Invalid ticker")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	idTokenCookie, err := r.Cookie("nodo_id_token")
	if err != nil {
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

	// Build query for financial data
	var queryInput *dynamodb.QueryInput

	if !rWithCursor {
		queryInput = &dynamodb.QueryInput{
			TableName:              aws.String("nodofinance_table"),
			KeyConditionExpression: aws.String("username = :username AND begins_with(composite_sk, :sk_prefix)"),
			ExpressionAttributeValues: map[string]dynamoTypes.AttributeValue{
				":username":  &dynamoTypes.AttributeValueMemberS{Value: username},
				":sk_prefix": &dynamoTypes.AttributeValueMemberS{Value: fmt.Sprintf("FINANCE#%s#", ticker)},
			},
			Limit: aws.Int32(1), // Get current + next for pagination
		}
	} else {
		exclusiveStartKey, err := parseCursor(cursor)
		if err != nil {
			logger.Log.Error("Invalid cursor", zap.Error(err))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		queryInput = &dynamodb.QueryInput{
			TableName:              aws.String("nodofinance_table"),
			KeyConditionExpression: aws.String("username = :username AND begins_with(composite_sk, :sk_prefix)"),
			ExpressionAttributeValues: map[string]dynamoTypes.AttributeValue{
				":username":  &dynamoTypes.AttributeValueMemberS{Value: username},
				":sk_prefix": &dynamoTypes.AttributeValueMemberS{Value: fmt.Sprintf("FINANCE#%s#", ticker)},
			},
			ExclusiveStartKey: exclusiveStartKey,
			Limit:             aws.Int32(1),
		}
	}

	result, err := d.Query(ctx, queryInput)
	if err != nil {
		logger.Log.Error("Error querying financial data", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(result.Items) == 0 {
		logger.Log.Error("Empty result set mounting ticker", zap.String("ticker", ticker), zap.String("username", username))
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Process current record
	currentRecord := result.Items[0]

	compositeSKAttr, exists := currentRecord["composite_sk"]
	if !exists {
		logger.Log.Error("Missing composite_sk in financial record")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	compositeSKMember, ok := compositeSKAttr.(*dynamoTypes.AttributeValueMemberS)
	if !ok {
		logger.Log.Error("Invalid composite_sk type in financial record")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	year, periodType, err := extractFromFinanceSK(compositeSKMember.Value)
	if err != nil {
		logger.Log.Error("Failed to parse financial sort key", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var response Response

	response.Period = fmt.Sprintf("%d-%s", year, periodType)

	if result.LastEvaluatedKey != nil {
		nextCursor := encodeCursor(result.LastEvaluatedKey)
		if nextCursor == "" {
			logger.Log.Error("Failed to encode next cursor")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		response.Cursor = nextCursor
	}

	// Get previous year data
	var prevYearRecord map[string]dynamoTypes.AttributeValue
	if year > 1 {
		prevYearSK, err := buildFinanceSortKey(ticker, year-1, periodType)
		if err == nil {
			prevYearResult, err := d.GetItem(ctx, &dynamodb.GetItemInput{
				TableName: aws.String("nodofinance_table"),
				Key: map[string]dynamoTypes.AttributeValue{
					"username":     &dynamoTypes.AttributeValueMemberS{Value: username},
					"composite_sk": &dynamoTypes.AttributeValueMemberS{Value: prevYearSK},
				},
			})
			if err == nil && len(prevYearResult.Item) > 0 {
				prevYearRecord = prevYearResult.Item
			}
		}
	}

	// Get ticker info (currency, analysis) when not using cursor
	if !rWithCursor {

		tickerResult, err := d.GetItem(ctx, &dynamodb.GetItemInput{
			TableName: aws.String("nodofinance_table"),
			Key: map[string]dynamoTypes.AttributeValue{
				"username":     &dynamoTypes.AttributeValueMemberS{Value: username},
				"composite_sk": &dynamoTypes.AttributeValueMemberS{Value: fmt.Sprintf("TICKER#%s", ticker)},
			},
			ProjectionExpression: aws.String("currency, analysis"),
		})

		if err != nil {
			logger.Log.Error("Error getting ticker metadata", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if len(tickerResult.Item) == 0 {
			logger.Log.Error("No ticker metadata found", zap.String("ticker", ticker), zap.String("username", username))
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if currencyAttr, exists := tickerResult.Item["currency"]; exists {
			if currencyStr, ok := currencyAttr.(*dynamoTypes.AttributeValueMemberS); ok {
				response.Currency = currencyStr.Value
			}
		}

		if analysisAttr, exists := tickerResult.Item["analysis"]; exists {
			if analysisStr, ok := analysisAttr.(*dynamoTypes.AttributeValueMemberS); ok {
				response.Analysis = analysisStr.Value
			}
		}
	}

	financialData := buildFinancialData(currentRecord, prevYearRecord)
	response.FinancialData = financialData

	// Send JSON response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Log.Error("Error encoding response", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

// Helper function to build financial data struct
func buildFinancialData(currentRecord, prevYearRecord map[string]dynamoTypes.AttributeValue) FinancialData {
	// Helper functions for pointer conversion
	setInt64Ptr := func(source *float64) *int64 {
		if source != nil {
			value := int64(*source)
			return &value
		}
		return nil
	}

	setFloat64Ptr := func(source *float64) *float64 {
		if source != nil {
			value := math.Round(*source*1000.0) / 1000.0
			return &value
		}
		return nil
	}

	financialData := FinancialData{}

	// Current period data
	financialData.CurrentAssets = setInt64Ptr(getDynamoDBFloatValue(currentRecord, "current_assets"))
	financialData.NonCurrentAssets = setInt64Ptr(getDynamoDBFloatValue(currentRecord, "non_current_assets"))
	financialData.Eps = setFloat64Ptr(getDynamoDBFloatValue(currentRecord, "eps"))
	financialData.CashAndEquivalents = setInt64Ptr(getDynamoDBFloatValue(currentRecord, "cash_and_equivalents"))
	financialData.CashFlowFromFinancing = setInt64Ptr(getDynamoDBFloatValue(currentRecord, "cash_flow_from_financing"))
	financialData.CashFlowFromInvesting = setInt64Ptr(getDynamoDBFloatValue(currentRecord, "cash_flow_from_investing"))
	financialData.CashFlowFromOperations = setInt64Ptr(getDynamoDBFloatValue(currentRecord, "cash_flow_from_operations"))
	financialData.Revenue = setInt64Ptr(getDynamoDBFloatValue(currentRecord, "revenue"))
	financialData.CurrentLiabilities = setInt64Ptr(getDynamoDBFloatValue(currentRecord, "current_liabilities"))
	financialData.NonCurrentLiabilities = setInt64Ptr(getDynamoDBFloatValue(currentRecord, "non_current_liabilities"))
	financialData.NetIncome = setInt64Ptr(getDynamoDBFloatValue(currentRecord, "net_income"))

	// Previous year data
	if prevYearRecord != nil {
		financialData.CurrentAssetsPrev = setInt64Ptr(getDynamoDBFloatValue(prevYearRecord, "current_assets"))
		financialData.NonCurrentAssetsPrev = setInt64Ptr(getDynamoDBFloatValue(prevYearRecord, "non_current_assets"))
		financialData.EpsPrev = setFloat64Ptr(getDynamoDBFloatValue(prevYearRecord, "eps"))
		financialData.CashAndEquivalentsPrev = setInt64Ptr(getDynamoDBFloatValue(prevYearRecord, "cash_and_equivalents"))
		financialData.CashFlowFromFinancingPrev = setInt64Ptr(getDynamoDBFloatValue(prevYearRecord, "cash_flow_from_financing"))
		financialData.CashFlowFromInvestingPrev = setInt64Ptr(getDynamoDBFloatValue(prevYearRecord, "cash_flow_from_investing"))
		financialData.CashFlowFromOperationsPrev = setInt64Ptr(getDynamoDBFloatValue(prevYearRecord, "cash_flow_from_operations"))
		financialData.RevenuePrev = setInt64Ptr(getDynamoDBFloatValue(prevYearRecord, "revenue"))
		financialData.CurrentLiabilitiesPrev = setInt64Ptr(getDynamoDBFloatValue(prevYearRecord, "current_liabilities"))
		financialData.NonCurrentLiabilitiesPrev = setInt64Ptr(getDynamoDBFloatValue(prevYearRecord, "non_current_liabilities"))
		financialData.NetIncomePrev = setInt64Ptr(getDynamoDBFloatValue(prevYearRecord, "net_income"))
	}

	return financialData
}
