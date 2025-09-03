package app

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"nodofinance/routes/app/submitter"
	"nodofinance/utils/jwt"
	"nodofinance/utils/logger"
	"nodofinance/utils/sanitize"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamoTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3Types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/openai/openai-go"
	"github.com/patrickmn/go-cache"
	"go.uber.org/zap"
)

// Req
type ChunkMetrics struct {
	FirstUniqueHits  int `json:"first_unique_hits"`
	SecondUniqueHits int `json:"second_unique_hits"`
	ThirdUniqueHits  int `json:"third_unique_hits"`
	FourthUniqueHits int `json:"fourth_unique_hits"`
	FifthUniqueHits  int `json:"fifth_unique_hits"`
}

type ChunkResult struct {
	Chunk      string       `json:"chunk,omitempty"`
	Cleaned    string       `json:"cleaned,omitempty"`
	Units      int64        `json:"units,omitempty"`
	Metrics    ChunkMetrics `json:"metrics"`
	Indicators []string     `json:"indicators"`
}

type PreprocessorOutput struct {
	BalanceResult  ChunkResult `json:"balance_result"`
	IncomeResult   ChunkResult `json:"income_result"`
	CashFlowResult ChunkResult `json:"cash_flow_result"`
	Language       string      `json:"language"`
}

type SubmitReq struct {
	Ticker   string             `json:"ticker"`
	Period   string             `json:"period"`
	Currency string             `json:"currency"`
	Content  PreprocessorOutput `json:"content"`
}

// S3
func convertChunkMetrics(cm ChunkMetrics) S3MetricsData {
	return S3MetricsData{
		First:  cm.FirstUniqueHits,
		Second: cm.SecondUniqueHits,
		Third:  cm.ThirdUniqueHits,
		Fourth: cm.FourthUniqueHits,
		Fifth:  cm.FifthUniqueHits,
	}
}

type S3MetricsData struct {
	First  int `json:"first"`
	Second int `json:"second"`
	Third  int `json:"third"`
	Fourth int `json:"fourth"`
	Fifth  int `json:"fifth"`
}

type S3StatementSection struct {
	Metrics         S3MetricsData `json:"metrics"`
	RawContent      string        `json:"raw_content"`
	CleanerPrompt   string        `json:"cleaner_prompt"`
	SubmitterPrompt string        `json:"submitter_prompt"`
}

type S3Document struct {
	Balance     S3StatementSection      `json:"balance"`
	Income      S3StatementSection      `json:"income"`
	CashFlow    S3StatementSection      `json:"cash_flow"`
	FinalResult submitter.Postprocessed `json:"final_result"`
}

func Submit(w http.ResponseWriter, r *http.Request, d *dynamodb.Client, s *s3.Client, ai openai.Client, dataCache *cache.Cache, devMode bool) {
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

	// 1. Request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Log.Error("Failed to read request body", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	var req SubmitReq
	if err := json.Unmarshal(body, &req); err != nil {
		logger.Log.Error("Failed to unmarshal request body", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ticker := sanitize.Trim(req.Ticker, "u")
	period := sanitize.Trim(req.Period, "u")
	currency := sanitize.Trim(req.Currency, "u")
	language := sanitize.Trim(req.Content.Language, "u")
	unitsFromClient := submitter.UnitsFromClient{
		Balance:  req.Content.BalanceResult.Units,
		Income:   req.Content.IncomeResult.Units,
		CashFlow: req.Content.CashFlowResult.Units,
	}

	if unitsFromClient.Balance == 0 || unitsFromClient.Income == 0 || unitsFromClient.CashFlow == 0 {
		logger.Log.Error("Units from client are zero", zap.Int64("balance_units", unitsFromClient.Balance), zap.Int64("income_units", unitsFromClient.Income), zap.Int64("cashflow_units", unitsFromClient.CashFlow))
	}

	if !sanitize.Ticker(ticker) || !sanitize.Period(period) || !sanitize.Currency(currency) || !sanitize.Language(language) ||
		!sanitize.Units(unitsFromClient.Balance) || !sanitize.Units(unitsFromClient.Income) || !sanitize.Units(unitsFromClient.CashFlow) {
		logger.Log.Error("Invalid parameter", zap.String("ticker", ticker), zap.String("period", period), zap.String("currency", currency), zap.String("language", language),
			zap.Int64("balance_units", unitsFromClient.Balance), zap.Int64("income_units", unitsFromClient.Income), zap.Int64("cashflow_units", unitsFromClient.CashFlow))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Handle zeros in unitsFromClient
	values := []int64{unitsFromClient.Balance, unitsFromClient.Income, unitsFromClient.CashFlow}
	nonZeros := make([]int64, 0, 3)

	for _, v := range values {
		if v > 0 {
			nonZeros = append(nonZeros, v)
		}
	}

	// Case: exactly one value is zero
	if len(nonZeros) == 2 && nonZeros[0] == nonZeros[1] {
		// Replace the zero with the common non-zero value
		if unitsFromClient.Balance == 0 {
			unitsFromClient.Balance = nonZeros[0]
		} else if unitsFromClient.Income == 0 {
			unitsFromClient.Income = nonZeros[0]
		} else if unitsFromClient.CashFlow == 0 {
			unitsFromClient.CashFlow = nonZeros[0]
		}
	} else if len(nonZeros) == 1 {
		// Case: two values are zero, set all to the non-zero value
		unitsFromClient.Balance = nonZeros[0]
		unitsFromClient.Income = nonZeros[0]
		unitsFromClient.CashFlow = nonZeros[0]
	}

	const maxChunkSize = 14 * 1024 // 14 KB

	balanceChunk := strings.TrimSpace(req.Content.BalanceResult.Chunk)
	incomeChunk := strings.TrimSpace(req.Content.IncomeResult.Chunk)
	cashFlowChunk := strings.TrimSpace(req.Content.CashFlowResult.Chunk)

	if len(balanceChunk) < 200 && len(incomeChunk) < 200 && len(cashFlowChunk) < 200 {
		logger.Log.Error("Chunks too small", zap.Int("balance_chunk", len(balanceChunk)), zap.Int("income_chunk", len(incomeChunk)), zap.Int("cashflow_chunk", len(cashFlowChunk)))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if len(balanceChunk) >= maxChunkSize ||
		len(incomeChunk) >= maxChunkSize ||
		len(cashFlowChunk) >= maxChunkSize {
		logger.Log.Error("Chunk too large",
			zap.Int("balance_chunk_size", len(balanceChunk)),
			zap.Int("income_chunk_size", len(incomeChunk)),
			zap.Int("cashflow_chunk_size", len(cashFlowChunk)))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	balanceCleaned := strings.TrimSpace(req.Content.BalanceResult.Cleaned)
	incomeCleaned := strings.TrimSpace(req.Content.IncomeResult.Cleaned)
	cashFlowCleaned := strings.TrimSpace(req.Content.CashFlowResult.Cleaned)

	if len(balanceCleaned) < 200 && len(incomeCleaned) < 200 && len(cashFlowCleaned) < 200 {
		logger.Log.Error("Cleaned chunks too small", zap.Int("balance_cleaned", len(balanceCleaned)), zap.Int("income_cleaned", len(incomeCleaned)), zap.Int("cashflow_cleaned", len(cashFlowCleaned)))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if len(balanceCleaned) >= maxChunkSize ||
		len(incomeCleaned) >= maxChunkSize ||
		len(cashFlowCleaned) >= maxChunkSize {
		logger.Log.Error("Chunk too large",
			zap.Int("balance_cleaned_size", len(balanceCleaned)),
			zap.Int("income_cleaned_size", len(incomeCleaned)),
			zap.Int("cashflow_cleaned_size", len(cashFlowCleaned)))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	const maxMetricValue = 80
	hasExcessiveMetrics := func(metrics ChunkMetrics, maxValue int) bool {
		return metrics.FirstUniqueHits >= maxValue ||
			metrics.SecondUniqueHits >= maxValue ||
			metrics.ThirdUniqueHits >= maxValue ||
			metrics.FourthUniqueHits >= maxValue ||
			metrics.FifthUniqueHits >= maxValue
	}

	if hasExcessiveMetrics(req.Content.BalanceResult.Metrics, maxMetricValue) ||
		hasExcessiveMetrics(req.Content.IncomeResult.Metrics, maxMetricValue) ||
		hasExcessiveMetrics(req.Content.CashFlowResult.Metrics, maxMetricValue) {
		logger.Log.Error("ChunkMetrics values too high")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if req.Content.BalanceResult.Metrics.FirstUniqueHits < 17 &&
		req.Content.IncomeResult.Metrics.FirstUniqueHits < 17 &&
		req.Content.CashFlowResult.Metrics.FirstUniqueHits < 17 {
		logger.Log.Error("Not enough unique hits", zap.Int("balance_hits", req.Content.BalanceResult.Metrics.FirstUniqueHits), zap.Int("income_hits", req.Content.IncomeResult.Metrics.FirstUniqueHits), zap.Int("cashflow_hits", req.Content.CashFlowResult.Metrics.FirstUniqueHits))
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("nodo error: Invalid file. Is it a financial file? Is it in Spanish or English?"))
		return
	}

	const maxIndicators = 80
	const maxIndicatorsSize = 1000 // 1 KB
	calcSize := func(indicators []string) int {
		size := 0
		for _, ind := range indicators {
			size += len(ind)
		}
		return size
	}

	if len(req.Content.BalanceResult.Indicators) >= maxIndicators ||
		len(req.Content.IncomeResult.Indicators) >= maxIndicators ||
		len(req.Content.CashFlowResult.Indicators) >= maxIndicators ||
		calcSize(req.Content.BalanceResult.Indicators) > maxIndicatorsSize ||
		calcSize(req.Content.IncomeResult.Indicators) > maxIndicatorsSize ||
		calcSize(req.Content.CashFlowResult.Indicators) > maxIndicatorsSize {
		logger.Log.Error("Indicators too large",
			zap.Int("balance_indicators_size", calcSize(req.Content.BalanceResult.Indicators)),
			zap.Int("income_indicators_size", calcSize(req.Content.IncomeResult.Indicators)),
			zap.Int("cashflow_indicators_size", calcSize(req.Content.CashFlowResult.Indicators)),
			zap.Int("balance_indicators", len(req.Content.BalanceResult.Indicators)),
			zap.Int("income_indicators", len(req.Content.IncomeResult.Indicators)),
			zap.Int("cashflow_indicators", len(req.Content.CashFlowResult.Indicators)))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 2. Consumption limits
	limitResult, err := checkConsumptionLimits(ctx, d, username, ticker)
	if err != nil {
		logger.Log.Error("Error checking consumption limits", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if !limitResult.Allowed {
		var limitTypeStr string
		switch limitResult.LimitReached {
		case LimitTypePeriods:
			limitTypeStr = "PERIODS"
		case LimitTypeTickers:
			limitTypeStr = "TICKERS"
		case LimitTypeTokens:
			limitTypeStr = "TOKENS"
		default:
			limitTypeStr = "UNKNOWN"
		}

		logger.Log.Info("Consumption limits exceeded",
			zap.String("user", username),
			zap.String("reason", limitTypeStr))

		var limitMessage string
		switch limitResult.LimitReached {
		case LimitTypePeriods:
			limitMessage = fmt.Sprintf("Max %d periods per ticker", MAX_PERIODS)
		case LimitTypeTickers:
			limitMessage = fmt.Sprintf("Max %d tickers", MAX_TICKERS)
		case LimitTypeTokens:
			limitMessage = fmt.Sprintf("You have consumed all your tokens. Current limit: %d tokens per month.", MAX_TOKENS)
		default:
			limitMessage = "Consumption limit reached"
		}

		http.Error(w, limitMessage, http.StatusForbidden)
		return
	}

	// 3. Submitter
	hits := submitter.Hits{
		Balance:  req.Content.BalanceResult.Metrics.FirstUniqueHits,
		Income:   req.Content.IncomeResult.Metrics.FirstUniqueHits,
		CashFlow: req.Content.CashFlowResult.Metrics.FirstUniqueHits,
	}

	submitterResponse, err := submitter.CallSubmitter(
		ctx, ai,
		balanceCleaned, incomeCleaned, cashFlowCleaned,
		period, language, hits,
		unitsFromClient)
	if err != nil {
		logger.Log.Error("Error calling submitter", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if submitterResponse.Balance.FinalContent == "" ||
		submitterResponse.Income.FinalContent == "" ||
		submitterResponse.CashFlow.FinalContent == "" {
		logger.Log.Error("Empty response from submitter")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	balanceData, err := submitter.ParseFinancialData("balance", submitterResponse.Balance.FinalContent)
	if err != nil {
		logger.Log.Error("Failed to parse balance sheet", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	balance, ok := balanceData.(*submitter.BalanceSheet)
	if !ok {
		logger.Log.Error("Type assertion failed for balance sheet")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if unitsFromClient.Balance != 0 {
		balanceUnitsFloat := float64(unitsFromClient.Balance)
		balance.Units = &balanceUnitsFloat
	}

	incomeData, err := submitter.ParseFinancialData("income", submitterResponse.Income.FinalContent)
	if err != nil {
		logger.Log.Error("Failed to parse income statement", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	income, ok := incomeData.(*submitter.IncomeStatement)
	if !ok {
		logger.Log.Error("Type assertion failed for income statement")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if unitsFromClient.Income != 0 {
		incomeUnitsFloat := float64(unitsFromClient.Income)
		income.Units = &incomeUnitsFloat
	}

	cashFlowData, err := submitter.ParseFinancialData("cash_flow", submitterResponse.CashFlow.FinalContent)
	if err != nil {
		logger.Log.Error("Failed to parse cash flow statement", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	cashFlow, ok := cashFlowData.(*submitter.CashFlowStatement)
	if !ok {
		logger.Log.Error("Type assertion failed for cash flow statement")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if unitsFromClient.CashFlow != 0 {
		cashFlowUnitsFloat := float64(unitsFromClient.CashFlow)
		cashFlow.Units = &cashFlowUnitsFloat
	}

	postprocessedResult, err := submitter.Postprocessor(balance, income, cashFlow)
	if err != nil {
		logger.Log.Error("Failed to postprocess financial data", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// 4. S3
	s3Doc := S3Document{}

	s3Doc.Balance.Metrics = convertChunkMetrics(req.Content.BalanceResult.Metrics)
	s3Doc.Balance.RawContent = req.Content.BalanceResult.Chunk
	s3Doc.Balance.CleanerPrompt = submitterResponse.Balance.CleanerPrompt
	s3Doc.Balance.SubmitterPrompt = submitterResponse.Balance.SubmitterPrompt

	s3Doc.Income.Metrics = convertChunkMetrics(req.Content.IncomeResult.Metrics)
	s3Doc.Income.RawContent = req.Content.IncomeResult.Chunk
	s3Doc.Income.CleanerPrompt = submitterResponse.Income.CleanerPrompt
	s3Doc.Income.SubmitterPrompt = submitterResponse.Income.SubmitterPrompt

	s3Doc.CashFlow.Metrics = convertChunkMetrics(req.Content.CashFlowResult.Metrics)
	s3Doc.CashFlow.RawContent = req.Content.CashFlowResult.Chunk
	s3Doc.CashFlow.CleanerPrompt = submitterResponse.CashFlow.CleanerPrompt
	s3Doc.CashFlow.SubmitterPrompt = submitterResponse.CashFlow.SubmitterPrompt

	s3Doc.FinalResult = postprocessedResult

	s3JSON, err := json.Marshal(s3Doc)
	if err != nil {
		logger.Log.Error("Failed to marshal S3 document", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var compressedData bytes.Buffer
	gzipWriter := gzip.NewWriter(&compressedData)
	if _, err := gzipWriter.Write(s3JSON); err != nil {
		logger.Log.Error("Failed to write to gzip writer", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err := gzipWriter.Close(); err != nil {
		logger.Log.Error("Failed to close gzip writer", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	s3BucketName := "financial-docs-outputs"
	s3FileName := username + "_" + ticker + "_" + period + ".json.gz"

	v := reflect.ValueOf(postprocessedResult)

	for i := range v.NumField() {
		field := v.Field(i)

		if field.Kind() == reflect.Ptr && field.IsNil() {
			logger.Log.Info("nil field in postprocessed result",
				zap.String("S3 file", s3FileName),
			)
			break
		}
	}

	putObjectInput := &s3.PutObjectInput{
		Bucket:          aws.String(s3BucketName),
		Key:             aws.String(s3FileName),
		Body:            bytes.NewReader(compressedData.Bytes()),
		ContentEncoding: aws.String("gzip"),
		ContentType:     aws.String("application/json"),
		StorageClass:    s3Types.StorageClassStandardIa,
		Metadata: map[string]string{
			"language": language,
		},
	}

	_, err = s.PutObject(ctx, putObjectInput)
	if err != nil {
		logger.Log.Error("Failed to upload to S3", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// 5. DynamoDB
	year, err := strconv.Atoi(period[:4])
	if err != nil {
		logger.Log.Error("Failed to parse year from period", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	periodType := period[5:]
	currentTime := time.Now().Unix()

	financeSK, err := buildFinanceSortKey(ticker, year, periodType)
	if err != nil {
		logger.Log.Error("Failed to build finance sort key", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Calculate total tokens
	var totalTokens = int64(
		float64(
			submitterResponse.Balance.PromptTokensCleaner+submitterResponse.Income.PromptTokensCleaner+submitterResponse.CashFlow.PromptTokensCleaner) +
			float64(
				submitterResponse.Balance.PromptTokensSubmitter+submitterResponse.Income.PromptTokensSubmitter+submitterResponse.CashFlow.PromptTokensSubmitter) +
			float64(
				submitterResponse.Balance.CompletionTokensCleaner+submitterResponse.Income.CompletionTokensCleaner+submitterResponse.CashFlow.CompletionTokensCleaner)*(OUTPUT_RATE_SMALL/INPUT_RATE_SMALL) +
			float64(
				submitterResponse.Balance.CompletionTokensSubmitter+submitterResponse.Income.CompletionTokensSubmitter+submitterResponse.CashFlow.CompletionTokensSubmitter)*(OUTPUT_RATE_SMALL/INPUT_RATE_SMALL),
	)

	// DynamoDB Transaction
	transactItems := []dynamoTypes.TransactWriteItem{
		// 1. FINANCE#{ticker}#{reverse_year}#{period_order}
		{
			Put: &dynamoTypes.Put{
				TableName: aws.String("nodofinance_table"),
				Item: map[string]dynamoTypes.AttributeValue{
					"username":                  &dynamoTypes.AttributeValueMemberS{Value: username},
					"composite_sk":              &dynamoTypes.AttributeValueMemberS{Value: financeSK},
					"current_assets":            &dynamoTypes.AttributeValueMemberN{Value: formatInt64Ptr(postprocessedResult.CurrentAssets)},
					"non_current_assets":        &dynamoTypes.AttributeValueMemberN{Value: formatInt64Ptr(postprocessedResult.NonCurrentAssets)},
					"eps":                       &dynamoTypes.AttributeValueMemberN{Value: formatFloat64Ptr(postprocessedResult.EPS)},
					"cash_and_equivalents":      &dynamoTypes.AttributeValueMemberN{Value: formatInt64Ptr(postprocessedResult.Cash)},
					"cash_flow_from_financing":  &dynamoTypes.AttributeValueMemberN{Value: formatInt64Ptr(postprocessedResult.CashFlowFromFinancing)},
					"cash_flow_from_investing":  &dynamoTypes.AttributeValueMemberN{Value: formatInt64Ptr(postprocessedResult.CashFlowFromInvesting)},
					"cash_flow_from_operations": &dynamoTypes.AttributeValueMemberN{Value: formatInt64Ptr(postprocessedResult.CashFlowFromOperations)},
					"revenue":                   &dynamoTypes.AttributeValueMemberN{Value: formatInt64Ptr(postprocessedResult.Revenue)},
					"current_liabilities":       &dynamoTypes.AttributeValueMemberN{Value: formatInt64Ptr(postprocessedResult.CurrentLiabilities)},
					"non_current_liabilities":   &dynamoTypes.AttributeValueMemberN{Value: formatInt64Ptr(postprocessedResult.NonCurrentLiabilities)},
					"net_income":                &dynamoTypes.AttributeValueMemberN{Value: formatInt64Ptr(postprocessedResult.NetIncome)},
				},
			},
		},
		// 2. TICKER#{ticker}
		{
			Put: &dynamoTypes.Put{
				TableName: aws.String("nodofinance_table"),
				Item: map[string]dynamoTypes.AttributeValue{
					"username":     &dynamoTypes.AttributeValueMemberS{Value: username},
					"composite_sk": &dynamoTypes.AttributeValueMemberS{Value: fmt.Sprintf("TICKER#%s", ticker)},
					"last_update":  &dynamoTypes.AttributeValueMemberN{Value: strconv.FormatInt(currentTime, 10)},
					"currency":     &dynamoTypes.AttributeValueMemberS{Value: currency},
					// analysis starts empty, can be updated later
				},
			},
		},
		// 3. User Token Update
		{
			Update: &dynamoTypes.Update{
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
			},
		},
	}

	// Execute transaction
	_, err = d.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: transactItems,
	})

	if err != nil {
		// Handle condition check failure
		var conditionCheckFailed *dynamoTypes.TransactionCanceledException
		if errors.As(err, &conditionCheckFailed) {
			logger.Log.Error("User does not exist", zap.String("username", username))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		logger.Log.Error("Transaction failed", zap.Error(err), zap.String("username", username), zap.String("ticker", ticker), zap.String("period", period))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// *
	// **
	// ***
	// ****
	// *****
	// ******
	// ******* DEVMODE ->
	if devMode {
		postprocessedResultJSON, err := json.Marshal(postprocessedResult)
		if err != nil {
			logger.Log.Error("Failed to marshal postprocessed result", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		balanceResultJSON, _ := json.Marshal(req.Content.BalanceResult.Metrics)
		incomeResultJSON, _ := json.Marshal(req.Content.IncomeResult.Metrics)
		cashFlowResultJSON, _ := json.Marshal(req.Content.CashFlowResult.Metrics)

		contentToWrite := fmt.Sprintf(">>>>>SUBMITTER_RESULT:\n\n>>>Balance:\n\n%s\n\n\n\n>>>Income\n\n%s\n\n\n\n>>>CashFlow\n\n%s\n"+
			"\n\n\n\n>>>>>FINAL_RESULT:\n%s\n\n\n\n>>>>>HITS:\nbalance:%s\nincome:%s\ncashflow:%s",
			submitterResponse.Balance.FinalContent,
			submitterResponse.Income.FinalContent,
			submitterResponse.CashFlow.FinalContent,
			string(postprocessedResultJSON),
			string(balanceResultJSON),
			string(incomeResultJSON),
			string(cashFlowResultJSON),
		)
		filePath := "/Users/vitor/desktop/nodofinance/tests/_final_result.txt"
		err = os.WriteFile(filePath, []byte(contentToWrite), 0644)
		if err != nil {
			fmt.Println("Error writing file:", err)
			return
		}

		filePath = "/Users/vitor/desktop/nodofinance/tests/balance___raw.txt"
		err = os.WriteFile(filePath, []byte(req.Content.BalanceResult.Chunk), 0644)
		if err != nil {
			fmt.Println("Error writing file:", err)
			return
		}
		filePath = "/Users/vitor/desktop/nodofinance/tests/balance__cleaner.txt"
		err = os.WriteFile(filePath, []byte(submitterResponse.Balance.CleanerPrompt), 0644)
		if err != nil {
			fmt.Println("Error writing file:", err)
			return
		}
		filePath = "/Users/vitor/desktop/nodofinance/tests/balance_submitter.txt"
		err = os.WriteFile(filePath, []byte(submitterResponse.Balance.SubmitterPrompt), 0644)
		if err != nil {
			fmt.Println("Error writing file:", err)
			return
		}

		filePath = "/Users/vitor/desktop/nodofinance/tests/income___raw.txt"
		err = os.WriteFile(filePath, []byte(req.Content.IncomeResult.Chunk), 0644)
		if err != nil {
			fmt.Println("Error writing file:", err)
			return
		}
		filePath = "/Users/vitor/desktop/nodofinance/tests/income__cleaner.txt"
		err = os.WriteFile(filePath, []byte(submitterResponse.Income.CleanerPrompt), 0644)
		if err != nil {
			fmt.Println("Error writing file:", err)
			return
		}
		filePath = "/Users/vitor/desktop/nodofinance/tests/income_submitter.txt"
		err = os.WriteFile(filePath, []byte(submitterResponse.Income.SubmitterPrompt), 0644)
		if err != nil {
			fmt.Println("Error writing file:", err)
			return
		}

		filePath = "/Users/vitor/desktop/nodofinance/tests/cashflow___raw.txt"
		err = os.WriteFile(filePath, []byte(req.Content.CashFlowResult.Chunk), 0644)
		if err != nil {
			fmt.Println("Error writing file:", err)
			return
		}
		filePath = "/Users/vitor/desktop/nodofinance/tests/cashflow__cleaner.txt"
		err = os.WriteFile(filePath, []byte(submitterResponse.CashFlow.CleanerPrompt), 0644)
		if err != nil {
			fmt.Println("Error writing file:", err)
			return
		}
		filePath = "/Users/vitor/desktop/nodofinance/tests/cashflow_submitter.txt"
		err = os.WriteFile(filePath, []byte(submitterResponse.CashFlow.SubmitterPrompt), 0644)
		if err != nil {
			fmt.Println("Error writing file:", err)
			return
		}
	}
	// ******* <- DEVMODE
	// ******
	// *****
	// ****
	// ***
	// **
	// *

	cacheKey := "tickers_" + username
	dataCache.Delete(cacheKey)

	// 6. Response
	w.WriteHeader(http.StatusOK)
}
