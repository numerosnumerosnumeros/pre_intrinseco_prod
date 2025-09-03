package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"

	"strings"

	"nodofinance/utils/jwt"
	"nodofinance/utils/logger"
	"nodofinance/utils/sanitize"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamoTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/openai/openai-go"

	"go.uber.org/zap"
)

type AnalystReq struct {
	Ticker   string `json:"ticker"`
	Currency string `json:"currency"`
}

type AnalystRes struct {
	AnalystMessage string `json:"analyst_message"`
}

func Analyst(w http.ResponseWriter, r *http.Request, d *dynamodb.Client, ai openai.Client) {
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

	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Log.Error("Failed to read request body", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	var req AnalystReq
	if err := json.Unmarshal(body, &req); err != nil {
		logger.Log.Error("Failed to unmarshal request body", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ticker := req.Ticker
	currency := req.Currency

	ticker = sanitize.Trim(ticker, "u")
	currency = sanitize.Trim(currency, "u")

	if !sanitize.Ticker(ticker) || !sanitize.Currency(currency) {
		logger.Log.Error("Invalid ticker or currency", zap.String("ticker", ticker), zap.String("currency", currency))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Get user tokens
	getUserResult, err := d.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String("nodofinance_table"),
		Key: map[string]dynamoTypes.AttributeValue{
			"username":     &dynamoTypes.AttributeValueMemberS{Value: username},
			"composite_sk": &dynamoTypes.AttributeValueMemberS{Value: "METADATA"},
		},
		ProjectionExpression: aws.String("ctokens"),
	})

	if err != nil {
		logger.Log.Error("Failed to get user metadata", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if getUserResult.Item == nil {
		logger.Log.Error("No rows found for user", zap.String("username", username))
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var userMetadata struct {
		CTokens *int64 `dynamodbav:"ctokens,omitempty"`
	}

	err = attributevalue.UnmarshalMap(getUserResult.Item, &userMetadata)
	if err != nil {
		logger.Log.Error("Failed to unmarshal user metadata", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if userMetadata.CTokens == nil {
		logger.Log.Warn("CTokens is nil for user", zap.String("username", username))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Check token limit (identical logic)
	if *userMetadata.CTokens >= MAX_TOKENS {
		logger.Log.Warn("cTokens limit", zap.String("username", username), zap.Int64("cTokens", *userMetadata.CTokens))

		limitMessage := fmt.Sprintf("You have consumed all your tokens. Current limit: %d tokens per month.", MAX_TOKENS)

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(limitMessage))
		return
	}

	mergedFinances, rowsCount, err := GetFinances(ctx, d, username, ticker)
	if err != nil {
		logger.Log.Error("Failed to get finances", zap.Error(err), zap.String("username", username), zap.String("ticker", ticker))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if mergedFinances == "" {
		logger.Log.Error("No finances found", zap.String("username", username), zap.String("ticker", ticker))
		w.WriteHeader(http.StatusNotFound)
		return
	}

	openAIResponse, err := callOpenAI(ctx, ai, ticker, mergedFinances, currency, rowsCount)
	if openAIResponse == (OpenAIResponse{}) {
		logger.Log.Error("Failed to call OpenAI", zap.String("username", username), zap.String("ticker", ticker))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err != nil {
		logger.Log.Error("Failed to call OpenAI", zap.Error(err), zap.String("username", username), zap.String("ticker", ticker))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(openAIResponse.FinalContent) < 100 {
		logger.Log.Error("OpenAI response too short", zap.String("username", username), zap.String("ticker", ticker))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(openAIResponse.FinalContent) > 8000 {
		openAIResponse.FinalContent = openAIResponse.FinalContent[:8000] + "..."
	}

	var totalTokens int64
	if rowsCount > 2 {
		totalTokens = int64(
			float64(openAIResponse.PromptTokens)*(INPUT_RATE_BIG/INPUT_RATE_SMALL) +
				float64(openAIResponse.CompletionTokens)*(OUTPUT_RATE_BIG/INPUT_RATE_SMALL))
	} else {
		totalTokens = int64(
			float64(openAIResponse.PromptTokens) +
				float64(openAIResponse.CompletionTokens)*(OUTPUT_RATE_SMALL/INPUT_RATE_SMALL))
	}

	// Update user tokens (atomic increment)
	_, err = d.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String("nodofinance_table"),
		Key: map[string]dynamoTypes.AttributeValue{
			"username":     &dynamoTypes.AttributeValueMemberS{Value: username},
			"composite_sk": &dynamoTypes.AttributeValueMemberS{Value: "METADATA"},
		},
		UpdateExpression: aws.String("ADD ctokens :tokens"),
		ExpressionAttributeValues: map[string]dynamoTypes.AttributeValue{
			":tokens": &dynamoTypes.AttributeValueMemberN{Value: strconv.FormatInt(totalTokens, 10)},
		},
		ConditionExpression: aws.String("attribute_exists(username) AND attribute_exists(composite_sk)"),
	})

	if err != nil {
		// Check if it's a condition check failure (user doesn't exist)
		var conditionCheckFailed *dynamoTypes.ConditionalCheckFailedException
		if errors.As(err, &conditionCheckFailed) {
			logger.Log.Error("User metadata not found for token update", zap.String("username", username))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		logger.Log.Error("Failed to update user tokens", zap.Error(err), zap.String("username", username))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Direct update using known sort key pattern
	_, err = d.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String("nodofinance_table"),
		Key: map[string]dynamoTypes.AttributeValue{
			"username":     &dynamoTypes.AttributeValueMemberS{Value: username},
			"composite_sk": &dynamoTypes.AttributeValueMemberS{Value: fmt.Sprintf("TICKER#%s", ticker)},
		},
		UpdateExpression: aws.String("SET analysis = :analysis"),
		ExpressionAttributeValues: map[string]dynamoTypes.AttributeValue{
			":analysis": &dynamoTypes.AttributeValueMemberS{Value: openAIResponse.FinalContent},
		},
		ConditionExpression: aws.String("attribute_exists(username) AND attribute_exists(composite_sk)"),
	})

	if err != nil {
		// Check if it's a condition check failure (record doesn't exist)
		var conditionCheckFailed *dynamoTypes.ConditionalCheckFailedException
		if errors.As(err, &conditionCheckFailed) {
			logger.Log.Error("Ticker record not found for analysis update",
				zap.String("username", username),
				zap.String("ticker", ticker))
			w.WriteHeader(http.StatusNotFound) // More appropriate status
			return
		}

		logger.Log.Error("Failed to update analysis",
			zap.Error(err),
			zap.String("username", username),
			zap.String("ticker", ticker))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response := AnalystRes{
		AnalystMessage: openAIResponse.FinalContent,
	}

	jsonResponse, err := json.Marshal(response)
	if err != nil {
		logger.Log.Error("Failed to marshal response", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonResponse)
}

// *
// **
// ***
// ****
// ***** CONSTRUCT FINANCIAL DATA
func GetFinances(ctx context.Context, d *dynamodb.Client, username, ticker string) (string, int, error) {
	// Query DynamoDB for all financial records for this user+ticker
	result, err := d.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String("nodofinance_table"),
		KeyConditionExpression: aws.String("username = :pk AND begins_with(composite_sk, :sk_prefix)"),
		ExpressionAttributeValues: map[string]dynamoTypes.AttributeValue{
			":pk":        &dynamoTypes.AttributeValueMemberS{Value: username},
			":sk_prefix": &dynamoTypes.AttributeValueMemberS{Value: fmt.Sprintf("FINANCE#%s#", ticker)},
		},
		ScanIndexForward: aws.Bool(false), // DESC order (newest first, matches reverse_year design)
	})

	if err != nil {
		return "", 0, fmt.Errorf("database query failed: %w", err)
	}

	// Convert DynamoDB items to FinanceMap format (same as SQL version)
	finances, err := DynamoItemsToFinanceMaps(result.Items)
	if err != nil {
		return "", 0, fmt.Errorf("failed to convert items to finance maps: %w", err)
	}

	if len(finances) == 0 {
		return "", 0, fmt.Errorf("no finance records found")
	}

	// Use existing preprocessing logic (no changes needed)
	mergedFinances, rowsCount, err := PreprocessFinancesFromMaps(finances)
	if err != nil {
		return "", 0, err
	}

	return mergedFinances, rowsCount, nil
}

// DynamoItemsToFinanceMaps converts DynamoDB items to FinanceMap (equivalent to RowsToFinanceMaps)
func DynamoItemsToFinanceMaps(items []map[string]dynamoTypes.AttributeValue) ([]FinanceMap, error) {
	result := make([]FinanceMap, 0, len(items))

	for _, item := range items {
		entry := make(FinanceMap)

		// Convert each DynamoDB attribute to the expected format
		for key, attr := range item {
			switch a := attr.(type) {
			case *dynamoTypes.AttributeValueMemberS:
				entry[key] = a.Value
			case *dynamoTypes.AttributeValueMemberN:
				// Parse number based on field type
				if key == "eps" {
					// EPS should be float64
					if val, err := strconv.ParseFloat(a.Value, 64); err == nil {
						entry[key] = val
					} else {
						entry[key] = nil
					}
				} else {
					// Other financial fields should be int64
					if val, err := strconv.ParseInt(a.Value, 10, 64); err == nil {
						entry[key] = val
					} else {
						entry[key] = nil
					}
				}
			case *dynamoTypes.AttributeValueMemberNULL:
				entry[key] = nil
			default:
				// Skip other types (like PK, SK which aren't used in calculations)
				continue
			}
		}

		result = append(result, entry)
	}

	return result, nil
}

// PreprocessFinancesFromMaps - identical logic to PreprocessFinances but works with FinanceMap slice
func PreprocessFinancesFromMaps(finances []FinanceMap) (string, int, error) {
	if len(finances) == 0 {
		return "", 0, fmt.Errorf("no finance records found")
	}

	// Get the row count
	rowCount := len(finances)

	// Create entries that will be converted to JSON
	type Entry struct {
		Key   string
		Value map[string]any
	}

	var entries []Entry

	for _, row := range finances {
		year, _ := row["year"].(int64)
		periodType, _ := row["period_type"].(string)

		key := fmt.Sprintf("%d-%s", year, periodType)
		data := make(map[string]any)

		// Add original fields (IDENTICAL logic from original)
		addField := func(fieldName string) {
			if val, exists := row[fieldName]; exists && val != nil {
				if fieldName == "eps" {
					data[fieldName] = val // Keep as is (should be float64)
				} else {
					// For other financial fields, prefer int64 representation
					switch v := val.(type) {
					case float64:
						data[fieldName] = int64(v)
					default:
						data[fieldName] = val
					}
				}
			} else {
				data[fieldName] = nil
			}
		}

		// ALL CALCULATION LOGIC REMAINS IDENTICAL
		totalAssets := SafeIntOperation(row,
			[]string{"current_assets", "non_current_assets"},
			func(values []int64) int64 { return values[0] + values[1] })

		totalLiabilities := SafeIntOperation(row,
			[]string{"current_liabilities", "non_current_liabilities"},
			func(values []int64) int64 { return values[0] + values[1] })

		equity := SafeIntOperation(row,
			[]string{"current_assets", "non_current_assets", "current_liabilities", "non_current_liabilities"},
			func(values []int64) int64 { return (values[0] + values[1]) - (values[2] + values[3]) })

		workingCapital := SafeIntOperation(row,
			[]string{"current_assets", "current_liabilities"},
			func(values []int64) int64 { return values[0] - values[1] })

		addField("current_assets")
		addField("cash_and_equivalents")
		addField("non_current_assets")
		data["total_assets"] = totalAssets

		addField("current_liabilities")
		addField("non_current_liabilities")
		data["total_liabilities"] = totalLiabilities
		data["equity"] = equity
		data["working_capital"] = workingCapital

		// Calculate complex ratios (IDENTICAL logic)
		if workingCapital != nil {
			workingFloat := float64(*workingCapital)
			data["working_capital_over_non_current_liabilities"] = SafeDoubleOperation(row,
				[]string{"non_current_liabilities"},
				func(values []float64) float64 { return workingFloat / values[0] })
		} else {
			data["working_capital_over_non_current_liabilities"] = nil
		}

		addField("revenue")
		addField("net_income")
		addField("eps")

		// Calculate financial ratios (IDENTICAL logic)
		if totalAssets != nil && totalLiabilities != nil {
			assetsFloat := float64(*totalAssets)
			liabilitiesFloat := float64(*totalLiabilities)
			ratio := RoundTo2Decimals(assetsFloat / liabilitiesFloat)
			data["solvency_ratio"] = ratio
		} else {
			data["solvency_ratio"] = nil
		}

		if totalLiabilities != nil && equity != nil {
			liabilitiesFloat := float64(*totalLiabilities)
			equityFloat := float64(*equity)
			ratio := RoundTo2Decimals(liabilitiesFloat / equityFloat)
			data["debt_ratio"] = ratio
		} else {
			data["debt_ratio"] = nil
		}

		data["liquidity_ratio"] = SafeDoubleOperation(row,
			[]string{"current_assets", "current_liabilities"},
			func(values []float64) float64 { return values[0] / values[1] })

		if totalAssets != nil {
			assetsFloat := float64(*totalAssets)
			data["roa"] = SafeDoubleOperation(row,
				[]string{"net_income"},
				func(values []float64) float64 { return values[0] / assetsFloat })
		} else {
			data["roa"] = nil
		}

		if equity != nil {
			equityFloat := float64(*equity)
			data["roe"] = SafeDoubleOperation(row,
				[]string{"net_income"},
				func(values []float64) float64 { return values[0] / equityFloat })
		} else {
			data["roe"] = nil
		}

		data["net_margin"] = SafeDoubleOperation(row,
			[]string{"net_income", "revenue"},
			func(values []float64) float64 { return values[0] / values[1] })

		addField("cash_flow_from_financing")
		addField("cash_flow_from_investing")
		addField("cash_flow_from_operations")

		entries = append(entries, Entry{Key: key, Value: data})
	}

	// Reverse entries (IDENTICAL logic)
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}

	// Create final JSON (IDENTICAL logic)
	result := make(map[string]any)
	for _, entry := range entries {
		result[entry.Key] = entry.Value
	}

	jsonData, err := json.Marshal(result)
	if err != nil {
		return "", 0, fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return string(jsonData), rowCount, nil
}

// FinanceMap represents a row from the finances table as a map
type FinanceMap map[string]any

func RoundTo2Decimals(value float64) float64 {
	return math.Round(value*100) / 100
}

// SafeIntOperation performs operations on integer fields
func SafeIntOperation(row FinanceMap, fields []string, operation func([]int64) int64) *int64 {
	values := make([]int64, 0, len(fields))

	for _, field := range fields {
		val, exists := row[field]
		if !exists || val == nil {
			return nil
		}

		var intVal int64
		switch v := val.(type) {
		case int64:
			intVal = v
		case float64:
			intVal = int64(v)
		default:
			return nil
		}

		if intVal == 0 {
			return nil
		}

		values = append(values, intVal)
	}

	result := operation(values)
	return &result
}

// SafeDoubleOperation performs operations on float fields
func SafeDoubleOperation(row FinanceMap, fields []string, operation func([]float64) float64) *float64 {
	values := make([]float64, 0, len(fields))

	for _, field := range fields {
		val, exists := row[field]
		if !exists || val == nil {
			return nil
		}

		var floatVal float64
		switch v := val.(type) {
		case float64:
			floatVal = v
		case int64:
			floatVal = float64(v)
		default:
			return nil
		}

		if floatVal == 0.0 {
			return nil
		}

		values = append(values, floatVal)
	}

	result := RoundTo2Decimals(operation(values))
	return &result
}

// *
// **
// ***
// ****
// ***** PROMPT
func promptEngineer(mergedFinances, ticker, currency string, rows int) string {
	var builder strings.Builder
	var currencyStr string

	if currency == "EUR" || currency == "USD" || currency == "GBP" {
		currencyStr = fmt.Sprintf("  expressed in %s", currency)
	} else {
		currencyStr = ". Currency of the data is not specified."
	}

	// common for both cases
	fmt.Fprintf(&builder, "Financial analysis of %s%s:\n", ticker, currencyStr)
	fmt.Fprintf(&builder, "%s\n\n", mergedFinances)
	builder.WriteString("Periods: annual (YYYY-Y), quarterly (YYYY-Q1/Q2/Q3/Q4) or semi-annual (YYYY-S1/S2).\n")
	builder.WriteString("Mention material limitations in the data if detected.\n")
	builder.WriteString("Format:\n")
	builder.WriteString("* Professional markdown\n")

	if rows > 2 {
		builder.WriteString("* Highlight material variations\n")
		builder.WriteString("Methodology:\n")
		builder.WriteString("* Trend analysis\n")
		builder.WriteString("* KPI evolution\n")
		builder.WriteString("* Seasonal factors\n")
		builder.WriteString("* Fundamental drivers\n")
		builder.WriteString("Analytical focus on:\n")
		builder.WriteString("* Critical trends\n")
		builder.WriteString("* Strategic implications\n")
		builder.WriteString("* Well-founded perspectives\n")
		builder.WriteString("* Future EPS potential\n")
		builder.WriteString("Financial analysis structure:\n")
		builder.WriteString("1. Executive Summary\n")
		builder.WriteString("   - Main Conclusions\n")
		builder.WriteString("   - Critical Trends\n")
		builder.WriteString("   - Perspectives\n")
		builder.WriteString("2. Fundamental Analysis\n")
		builder.WriteString("   A. Profitability\n")
		builder.WriteString("      - Revenue\n")
		builder.WriteString("      - Margins\n")
		builder.WriteString("      - Quality of Results\n")
		builder.WriteString("   B. Financial Position\n")
		builder.WriteString("      - Capital Structure\n")
		builder.WriteString("      - Solvency and Liquidity\n")
		builder.WriteString("      - Operational Efficiency\n")
		builder.WriteString("   C. Cash Generation\n")
		builder.WriteString("      - Operating Cash Flow\n")
		builder.WriteString("      - Investment/Financing Policy\n")
		builder.WriteString("      - Cash Model Sustainability\n")
		builder.WriteString("3. Projections\n")
		builder.WriteString("   A. Operating KPIs\n")
		builder.WriteString("      - Revenue Growth\n")
		builder.WriteString("      - Margins and Profitability\n")
		builder.WriteString("      - Cash Generation\n")
		builder.WriteString("   B. Scenarios (including projected EPS)\n")
		builder.WriteString("      - Base Case\n")
		builder.WriteString("      - Bull Case\n")
		builder.WriteString("      - Bear Case\n")
		builder.WriteString("4. Risks\n")
		builder.WriteString("   - Operational\n")
		builder.WriteString("   - Financial\n")
		builder.WriteString("   - Structural")
	} else {
		builder.WriteString("Financial analysis structure:\n")
		builder.WriteString("1. Executive Summary\n")
		builder.WriteString("   - Main Conclusions\n")
		builder.WriteString("   - Key Indicators\n")
		builder.WriteString("   - Financial position\n")
		builder.WriteString("2. Fundamental Analysis\n")
		builder.WriteString("   A. Profitability\n")
		builder.WriteString("      - Revenue\n")
		builder.WriteString("      - Margins\n")
		builder.WriteString("      - Quality of Results\n")
		builder.WriteString("   B. Financial Position\n")
		builder.WriteString("      - Capital Structure\n")
		builder.WriteString("      - Solvency and Liquidity\n")
		builder.WriteString("      - Operational Efficiency\n")
		builder.WriteString("   C. Cash Generation\n")
		builder.WriteString("      - Operating Cash Flow\n")
		builder.WriteString("      - Investment/Financing Policy\n")
		builder.WriteString("      - Cash Model Sustainability\n")
		builder.WriteString("3. Considerations\n")
		builder.WriteString("   - Main Risks\n")
		builder.WriteString("   - Key Factors\n")
		builder.WriteString("   - Analysis Limitations")
	}

	return builder.String()
}

// *
// **
// ***
// ****
// ***** OPENAI
type OpenAIResponse struct {
	PromptTokens     int64  `json:"prompt_tokens"`
	CompletionTokens int64  `json:"completion_tokens"`
	FinalContent     string `json:"final_content"`
}

func callOpenAI(ctx context.Context, ai openai.Client, ticker, mergedFinances, currency string, rows int) (OpenAIResponse, error) {
	model := openai.ChatModelGPT4oMini
	if rows > 2 {
		model = openai.ChatModelGPT4o
	}

	openAIResponse := OpenAIResponse{}

	systemContent := "You are a senior equity research analyst at " +
		"a large investment bank. Your specialty is fundamental analysis " +
		"and company valuation. You must maintain a professional but " +
		"direct tone, emphasizing the critical points that affect the investment thesis. " +
		"Your recommendations must be backed by quantitative data."

	userPrompt := promptEngineer(mergedFinances, ticker, currency, rows)

	chatCompletion, err := ai.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemContent),
			openai.UserMessage(userPrompt),
		},
		Model:       model,
		MaxTokens:   openai.Int(10000),
		Temperature: openai.Float(0.4),
	})

	if err != nil {
		logger.Log.Error("Failed to call OpenAI API", zap.Error(err))
		return openAIResponse, err
	}

	if len(chatCompletion.Choices) > 0 {
		openAIResponse.FinalContent = chatCompletion.Choices[0].Message.Content
		openAIResponse.PromptTokens = chatCompletion.Usage.PromptTokens
		openAIResponse.CompletionTokens = chatCompletion.Usage.CompletionTokens
	} else {
		logger.Log.Error("OpenAI API returned no choices", zap.Any("response", chatCompletion))
		return openAIResponse, fmt.Errorf("invalid response format from OpenAI API")
	}

	return openAIResponse, nil

}
