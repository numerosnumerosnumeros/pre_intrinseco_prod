package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"nodofinance/utils/jwt"
	"nodofinance/utils/logger"
	"nodofinance/utils/sanitize"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamoTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"go.uber.org/zap"
)

type EditReq struct {
	Ticker           string         `json:"ticker"`
	Period           string         `json:"period"`
	NewFinancialData map[string]any `json:"new_financial_data"`
}

func Edit(w http.ResponseWriter, r *http.Request, d *dynamodb.Client) {
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

	var req EditReq
	if err := json.Unmarshal(body, &req); err != nil {
		logger.Log.Error("Failed to unmarshal request body", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ticker := req.Ticker
	fullPeriod := req.Period
	financialData := req.NewFinancialData

	ticker = sanitize.Trim(ticker, "u")
	fullPeriod = sanitize.Trim(fullPeriod, "u")

	if !sanitize.Ticker(ticker) || !sanitize.Period(fullPeriod) {
		logger.Log.Error("Invalid ticker or period", zap.String("ticker", ticker), zap.String("period", fullPeriod))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	year, err := strconv.Atoi(fullPeriod[:4])
	if err != nil {
		logger.Log.Error("Failed to parse year from period", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	periodType := fullPeriod[5:]

	if financialData == nil {
		logger.Log.Error("Financial data is nil")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if !sanitize.FinancialData(financialData) {
		logger.Log.Error("Invalid financial data", zap.Any("financial_data", financialData))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Process financial data
	newFinancialDataInt := make(map[string]*int64)
	var eps *float64

	for field, value := range financialData {
		if value == nil {
			if field == "eps" {
				eps = nil
			} else {
				newFinancialDataInt[field] = nil
			}
			continue
		}

		// Handle different types of values
		switch v := value.(type) {
		case float64:
			if field == "eps" {
				tmp := v
				eps = &tmp
			} else {
				tmp := int64(v)
				newFinancialDataInt[field] = &tmp
			}
		case json.Number:
			if field == "eps" {
				if f, err := v.Float64(); err == nil {
					eps = &f
				} else {
					eps = nil
				}
			} else {
				if i, err := v.Int64(); err == nil {
					newFinancialDataInt[field] = &i
				} else {
					newFinancialDataInt[field] = nil
				}
			}
		case string:
			if v == "" {
				if field == "eps" {
					eps = nil
				} else {
					newFinancialDataInt[field] = nil
				}
			} else {
				if field == "eps" {
					if f, err := strconv.ParseFloat(v, 64); err == nil {
						eps = &f
					} else {
						eps = nil
					}
				} else {
					if i, err := strconv.ParseInt(v, 10, 64); err == nil {
						newFinancialDataInt[field] = &i
					} else {
						newFinancialDataInt[field] = nil
					}
				}
			}
		default:
			if field == "eps" {
				eps = nil
			} else {
				newFinancialDataInt[field] = nil
			}
		}
	}

	financeSK, err := buildFinanceSortKey(ticker, year, periodType)
	if err != nil {
		logger.Log.Error("Failed to build finance sort key", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var setExpressions []string
	var removeExpressions []string
	expressionAttributeValues := make(map[string]dynamoTypes.AttributeValue)

	// Helper function to add update expression
	addUpdateExpression := func(field string, value interface{}) {
		if value == nil {
			// Remove attribute if nil
			removeExpressions = append(removeExpressions, field)
		} else {
			placeholder := fmt.Sprintf(":val_%s", field)
			setExpressions = append(setExpressions, fmt.Sprintf("%s = %s", field, placeholder))
			switch v := value.(type) {
			case *int64:
				if v != nil {
					expressionAttributeValues[placeholder] = &dynamoTypes.AttributeValueMemberN{
						Value: strconv.FormatInt(*v, 10),
					}
				}
			case *float64:
				if v != nil {
					expressionAttributeValues[placeholder] = &dynamoTypes.AttributeValueMemberN{
						Value: fmt.Sprintf("%.4f", *v),
					}
				}
			}
		}
	}

	// Add all financial fields
	addUpdateExpression("current_assets", newFinancialDataInt["current_assets"])
	addUpdateExpression("non_current_assets", newFinancialDataInt["non_current_assets"])
	addUpdateExpression("eps", eps)
	addUpdateExpression("cash_and_equivalents", newFinancialDataInt["cash_and_equivalents"])
	addUpdateExpression("cash_flow_from_financing", newFinancialDataInt["cash_flow_from_financing"])
	addUpdateExpression("cash_flow_from_investing", newFinancialDataInt["cash_flow_from_investing"])
	addUpdateExpression("cash_flow_from_operations", newFinancialDataInt["cash_flow_from_operations"])
	addUpdateExpression("revenue", newFinancialDataInt["revenue"])
	addUpdateExpression("current_liabilities", newFinancialDataInt["current_liabilities"])
	addUpdateExpression("non_current_liabilities", newFinancialDataInt["non_current_liabilities"])
	addUpdateExpression("net_income", newFinancialDataInt["net_income"])

	// Build proper DynamoDB UpdateExpression syntax
	var updateParts []string
	if len(setExpressions) > 0 {
		updateParts = append(updateParts, "SET "+strings.Join(setExpressions, ", "))
	}
	if len(removeExpressions) > 0 {
		updateParts = append(updateParts, "REMOVE "+strings.Join(removeExpressions, ", "))
	}
	updateExpression := strings.Join(updateParts, " ")

	// Execute DynamoDB UpdateItem
	updateInput := &dynamodb.UpdateItemInput{
		TableName: aws.String("nodofinance_table"),
		Key: map[string]dynamoTypes.AttributeValue{
			"username":     &dynamoTypes.AttributeValueMemberS{Value: username},
			"composite_sk": &dynamoTypes.AttributeValueMemberS{Value: financeSK},
		},
		UpdateExpression:    aws.String(updateExpression),
		ConditionExpression: aws.String("attribute_exists(composite_sk) AND attribute_exists(username)"),
		ReturnValues:        dynamoTypes.ReturnValueNone,
	}

	// Only add ExpressionAttributeValues if we have any
	if len(expressionAttributeValues) > 0 {
		updateInput.ExpressionAttributeValues = expressionAttributeValues
	}

	_, err = d.UpdateItem(ctx, updateInput)
	if err != nil {
		// Check if it's a condition check failure (record doesn't exist)
		var conditionCheckFailed *dynamoTypes.ConditionalCheckFailedException
		if errors.As(err, &conditionCheckFailed) {
			logger.Log.Warn("No rows affected, possibly invalid ticker or period", zap.String("ticker", ticker), zap.String("period", fullPeriod))
			w.WriteHeader(http.StatusNotFound)
			return
		}

		logger.Log.Error("Failed to update financial data", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	logger.Log.Info("User edited result", zap.String("username", username), zap.String("ticker", ticker), zap.String("period", fullPeriod))
	w.WriteHeader(http.StatusOK)
}
