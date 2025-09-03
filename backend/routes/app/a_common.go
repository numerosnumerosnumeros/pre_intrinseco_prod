package app

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"nodofinance/utils/env"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamoTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

var (
	MAX_TOKENS        int64
	MAX_PERIODS       int64
	MAX_TICKERS       int64
	INPUT_RATE_SMALL  float64
	OUTPUT_RATE_SMALL float64
	INPUT_RATE_BIG    float64
	OUTPUT_RATE_BIG   float64
)

func init() {
	env.RegisterValidator(validateVar)
}

func validateVar() error {
	var exists bool

	MAX_TOKENS_LIMIT, exists := env.Get("MAX_TOKENS_LIMIT")
	if !exists {
		return fmt.Errorf("missing required environment variable: MAX_TOKENS_LIMIT")
	}
	parsedValue, err := strconv.ParseInt(MAX_TOKENS_LIMIT, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid value for MAX_TOKENS_LIMIT: %w", err)
	}
	MAX_TOKENS = parsedValue

	VITE_MAX_TICKERS, exists := env.Get("VITE_MAX_TICKERS")
	if !exists {
		return fmt.Errorf("missing required environment variable: VITE_MAX_TICKERS")
	}
	parsedValue, err = strconv.ParseInt(VITE_MAX_TICKERS, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid value for VITE_MAX_TICKERS: %w", err)
	}
	MAX_PERIODS = parsedValue

	VITE_MAX_PERIODS_PER_TICKER, exists := env.Get("VITE_MAX_PERIODS_PER_TICKER")
	if !exists {
		return fmt.Errorf("missing required environment variable: VITE_MAX_PERIODS_PER_TICKER")
	}
	parsedValue, err = strconv.ParseInt(VITE_MAX_PERIODS_PER_TICKER, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid value for VITE_MAX_PERIODS_PER_TICKER: %w", err)
	}
	MAX_TICKERS = parsedValue

	inputRateSmallStr, exists := env.Get("AI_INPUT_RATE_PER_MILLION_SMALL")
	if !exists {
		return fmt.Errorf("missing required environment variable: AI_INPUT_RATE_PER_MILLION_SMALL")
	}
	parsedInputRateSmall, err := strconv.ParseFloat(inputRateSmallStr, 64)
	if err != nil {
		return fmt.Errorf("invalid value for AI_INPUT_RATE_PER_MILLION_SMALL: %w", err)
	}
	INPUT_RATE_SMALL = parsedInputRateSmall

	outputRateSmallStr, exists := env.Get("AI_OUTPUT_RATE_PER_MILLION_SMALL")
	if !exists {
		return fmt.Errorf("missing required environment variable: AI_OUTPUT_RATE_PER_MILLION_SMALL")
	}
	parsedOutputRateSmall, err := strconv.ParseFloat(outputRateSmallStr, 64)
	if err != nil {
		return fmt.Errorf("invalid value for AI_OUTPUT_RATE_PER_MILLION_SMALL: %w", err)
	}
	OUTPUT_RATE_SMALL = parsedOutputRateSmall

	inputRateBigStr, exists := env.Get("AI_INPUT_RATE_PER_MILLION_BIG")
	if !exists {
		return fmt.Errorf("missing required environment variable: AI_INPUT_RATE_PER_MILLION_BIG")
	}
	parsedInputRateBig, err := strconv.ParseFloat(inputRateBigStr, 64)
	if err != nil {
		return fmt.Errorf("invalid value for AI_INPUT_RATE_PER_MILLION_BIG: %w", err)
	}
	INPUT_RATE_BIG = parsedInputRateBig

	outputRateBigStr, exists := env.Get("AI_OUTPUT_RATE_PER_MILLION_BIG")
	if !exists {
		return fmt.Errorf("missing required environment variable: AI_OUTPUT_RATE_PER_MILLION_BIG")
	}
	parsedOutputRateBig, err := strconv.ParseFloat(outputRateBigStr, 64)
	if err != nil {
		return fmt.Errorf("invalid value for AI_OUTPUT_RATE_PER_MILLION_BIG: %w", err)
	}
	OUTPUT_RATE_BIG = parsedOutputRateBig

	return nil
}

type LimitType int

const (
	LimitTypeNone LimitType = iota
	LimitTypePeriods
	LimitTypeTickers
	LimitTypeTokens
)

type ConsumptionLimitResult struct {
	Allowed      bool
	LimitReached LimitType
}

func checkConsumptionLimits(ctx context.Context, d *dynamodb.Client, username, ticker string) (ConsumptionLimitResult, error) {
	// Check periods count for this ticker
	queryPeriodsInput := &dynamodb.QueryInput{
		TableName:              aws.String("nodofinance_table"),
		KeyConditionExpression: aws.String("#pk = :username AND begins_with(#sk, :finance_prefix)"),
		ExpressionAttributeNames: map[string]string{
			"#pk": "username",
			"#sk": "composite_sk",
		},
		ExpressionAttributeValues: map[string]dynamoTypes.AttributeValue{
			":username":       &dynamoTypes.AttributeValueMemberS{Value: username},
			":finance_prefix": &dynamoTypes.AttributeValueMemberS{Value: fmt.Sprintf("FINANCE#%s#", ticker)},
		},
		Select: dynamoTypes.SelectCount,
	}

	periodsResult, err := d.Query(ctx, queryPeriodsInput)
	if err != nil {
		return ConsumptionLimitResult{}, fmt.Errorf("checking periods count: %w", err)
	}

	periodsCount := int(periodsResult.Count)
	if periodsCount >= int(MAX_PERIODS) {
		return ConsumptionLimitResult{false, LimitTypePeriods}, nil
	}

	// If no records found for this ticker, check total unique tickers
	if periodsCount == 0 {
		queryTickersInput := &dynamodb.QueryInput{
			TableName:              aws.String("nodofinance_table"),
			KeyConditionExpression: aws.String("#pk = :username AND begins_with(#sk, :ticker_prefix)"),
			ExpressionAttributeNames: map[string]string{
				"#pk": "username",
				"#sk": "composite_sk",
			},
			ExpressionAttributeValues: map[string]dynamoTypes.AttributeValue{
				":username":      &dynamoTypes.AttributeValueMemberS{Value: username},
				":ticker_prefix": &dynamoTypes.AttributeValueMemberS{Value: "TICKER#"},
			},
			Select: dynamoTypes.SelectCount,
		}

		tickersResult, err := d.Query(ctx, queryTickersInput)
		if err != nil {
			return ConsumptionLimitResult{}, fmt.Errorf("checking tickers count: %w", err)
		}

		tickersCount := int(tickersResult.Count)
		if tickersCount >= int(MAX_TICKERS) {
			return ConsumptionLimitResult{false, LimitTypeTickers}, nil
		}
	}

	getUserInput := &dynamodb.GetItemInput{
		TableName: aws.String("nodofinance_table"),
		Key: map[string]dynamoTypes.AttributeValue{
			"username":     &dynamoTypes.AttributeValueMemberS{Value: username},
			"composite_sk": &dynamoTypes.AttributeValueMemberS{Value: "METADATA"},
		},
		ProjectionExpression: aws.String("ctokens"),
	}

	result, err := d.GetItem(ctx, getUserInput)
	if err != nil {
		return ConsumptionLimitResult{}, fmt.Errorf("checking token consumption: %w", err)
	}

	if result.Item == nil {
		return ConsumptionLimitResult{false, LimitTypeTokens}, nil
	}

	var user struct {
		CTokens *int64 `dynamodbav:"ctokens,omitempty"`
	}

	err = attributevalue.UnmarshalMap(result.Item, &user)
	if err != nil {
		return ConsumptionLimitResult{}, fmt.Errorf("unmarshaling user data: %w", err)
	}

	if user.CTokens == nil || *user.CTokens >= int64(MAX_TOKENS) {
		return ConsumptionLimitResult{false, LimitTypeTokens}, nil
	}

	return ConsumptionLimitResult{true, LimitTypeNone}, nil
}

func getPeriodOrder(period string) (int, error) {
	switch period {
	case "Y":
		return 1, nil
	case "S2":
		return 2, nil
	case "S1":
		return 3, nil
	case "Q4":
		return 4, nil
	case "Q3":
		return 5, nil
	case "Q2":
		return 6, nil
	case "Q1":
		return 7, nil
	default:
		return 0, fmt.Errorf("invalid period: %s", period)
	}
}

// Convert period order back to period string
func getPeriodFromOrder(order int) string {
	switch order {
	case 1:
		return "Y"
	case 2:
		return "S2"
	case 3:
		return "S1"
	case 4:
		return "Q4"
	case 5:
		return "Q3"
	case 6:
		return "Q2"
	case 7:
		return "Q1"
	default:
		return ""
	}
}

func buildFinanceSortKey(ticker string, year int, period string) (string, error) {
	reverseYear := 9999 - year

	periodOrder, err := getPeriodOrder(period)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("FINANCE#%s#%04d#%02d", ticker, reverseYear, periodOrder), nil
}

// Helper function to extract float values from DynamoDB attributes
func getDynamoDBFloatValue(item map[string]dynamoTypes.AttributeValue, key string) *float64 {
	if item == nil {
		return nil
	}
	if attr, exists := item[key]; exists {
		if n, ok := attr.(*dynamoTypes.AttributeValueMemberN); ok {
			if val, err := strconv.ParseFloat(n.Value, 64); err == nil {
				return &val
			}
		}
	}
	return nil
}

// Helper function to format *int64 for DynamoDB
func formatInt64Ptr(val *int64) string {
	if val == nil {
		return ""
	}
	return strconv.FormatInt(*val, 10)
}

// Helper function to format *float64 for DynamoDB
func formatFloat64Ptr(val *float64) string {
	if val == nil {
		return ""
	}
	return fmt.Sprintf("%.4f", *val)
}

// Extract year and period from FINANCE sort key
func extractFromFinanceSK(sortKey string) (int, string, error) {
	// FINANCE#AAPL#7977#04 -> year=2022, period=Q4
	parts := strings.Split(sortKey, "#")
	if len(parts) != 4 {
		return 0, "", fmt.Errorf("invalid finance sort key format")
	}

	reverseYear, err := strconv.Atoi(parts[2])
	if err != nil {
		return 0, "", fmt.Errorf("invalid reverse year: %w", err)
	}

	periodOrder, err := strconv.Atoi(parts[3])
	if err != nil {
		return 0, "", fmt.Errorf("invalid period order: %w", err)
	}

	year := 9999 - reverseYear
	period := getPeriodFromOrder(periodOrder)

	return year, period, nil
}

// Encode LastEvaluatedKey as base64 cursor
func encodeCursor(lastEvaluatedKey map[string]dynamoTypes.AttributeValue) string {
	// Convert to simple map for JSON encoding
	cursorData := make(map[string]interface{})

	for key, value := range lastEvaluatedKey {
		switch v := value.(type) {
		case *dynamoTypes.AttributeValueMemberS:
			cursorData[key] = map[string]string{"S": v.Value}
		case *dynamoTypes.AttributeValueMemberN:
			cursorData[key] = map[string]string{"N": v.Value}
		}
	}

	jsonData, err := json.Marshal(cursorData)
	if err != nil {
		return ""
	}

	return base64.URLEncoding.EncodeToString(jsonData)
}

// Parse base64 cursor back to ExclusiveStartKey
func parseCursor(cursor string) (map[string]dynamoTypes.AttributeValue, error) {
	jsonData, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor encoding: %w", err)
	}

	var cursorData map[string]any
	if err := json.Unmarshal(jsonData, &cursorData); err != nil {
		return nil, fmt.Errorf("invalid cursor format: %w", err)
	}

	exclusiveStartKey := make(map[string]dynamoTypes.AttributeValue)

	for key, value := range cursorData {
		valueMap, ok := value.(map[string]interface{})
		if !ok {
			continue
		}

		if s, exists := valueMap["S"]; exists {
			if str, ok := s.(string); ok {
				exclusiveStartKey[key] = &dynamoTypes.AttributeValueMemberS{Value: str}
			}
		} else if n, exists := valueMap["N"]; exists {
			if str, ok := n.(string); ok {
				exclusiveStartKey[key] = &dynamoTypes.AttributeValueMemberN{Value: str}
			}
		}
	}

	return exclusiveStartKey, nil
}
