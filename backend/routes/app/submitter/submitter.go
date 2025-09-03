package submitter

import (
	"context"
	"encoding/json"
	"fmt"
	"nodofinance/utils/logger"
	"reflect"
	"strings"

	"github.com/openai/openai-go"
	"go.uber.org/zap"
)

type BalanceSheet struct {
	Units                 *float64 `json:"units"`
	CashAndEquivalents    *float64 `json:"cash_and_equivalents"`
	CurrentAssets         *float64 `json:"current_assets"`
	NonCurrentAssets      *float64 `json:"non_current_assets"`
	TotalAssets           *float64 `json:"total_assets"`
	CurrentLiabilities    *float64 `json:"current_liabilities"`
	NonCurrentLiabilities *float64 `json:"non_current_liabilities"`
	TotalLiabilities      *float64 `json:"total_liabilities"`
	Equity                *float64 `json:"equity"`
}

type IncomeStatement struct {
	Units     *float64 `json:"units"`
	Revenue   *float64 `json:"revenue"`
	NetIncome *float64 `json:"net_income"`
	EPS       *float64 `json:"eps"`
}

type CashFlowStatement struct {
	Units                  *float64 `json:"units"`
	CashFlowFromOperations *float64 `json:"cash_flow_from_operations"`
	CashFlowFromInvesting  *float64 `json:"cash_flow_from_investing"`
	CashFlowFromFinancing  *float64 `json:"cash_flow_from_financing"`
}

func getRequiredFields(target string, units int64) []string {
	var structType reflect.Type

	switch target {
	case "balance":
		structType = reflect.TypeOf(BalanceSheet{})
	case "income":
		structType = reflect.TypeOf(IncomeStatement{})
	case "cash_flow":
		structType = reflect.TypeOf(CashFlowStatement{})
	default:
		return []string{}
	}

	fields := make([]string, 0, structType.NumField())
	for i := range structType.NumField() {
		field := structType.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag != "" {
			// Extract the field name from the JSON tag
			tagParts := strings.Split(jsonTag, ",")
			fieldName := tagParts[0]

			if fieldName == "units" && units != 0 {
				continue
			}

			fields = append(fields, fieldName)
		}
	}

	return fields
}

// Parse the content into the appropriate struct
func ParseFinancialData(target string, content string) (any, error) {
	var structPtr any
	switch target {
	case "balance":
		structPtr = &BalanceSheet{}
	case "income":
		structPtr = &IncomeStatement{}
	case "cash_flow":
		structPtr = &CashFlowStatement{}
	default:
		return nil, fmt.Errorf("unknown target: %s", target)
	}

	err := json.Unmarshal([]byte(content), structPtr)
	if err != nil {
		return nil, err
	}

	return structPtr, nil
}

func createFinancialDataSchema(fields []string) any {
	properties := map[string]any{}
	required := []string{}

	for _, field := range fields {
		properties[field] = map[string]any{
			"type": []string{"number", "null"},
		}
		required = append(required, field)
	}

	schema := map[string]any{
		"type":                 "object",
		"properties":           properties,
		"required":             required,
		"additionalProperties": false,
	}

	return schema
}

func createNullJSONResponse(target string) (string, error) {
	doc := make(map[string]any)

	fields := getRequiredFields(target, 0)

	for _, field := range fields {
		doc[field] = nil
	}

	jsonBytes, err := json.Marshal(doc)
	if err != nil {
		logger.Log.Error("Failed to marshal JSON", zap.Error(err))
		return "", err
	}

	return string(jsonBytes), nil
}

type UnitsFromClient struct {
	Balance  int64
	Income   int64
	CashFlow int64
}

type Hits struct {
	Balance  int
	Income   int
	CashFlow int
}

type AIresponse struct {
	PromptTokensCleaner       int64  `json:"prompt_tokens_cleaner"`
	CompletionTokensCleaner   int64  `json:"completion_tokens_cleaner"`
	PromptTokensSubmitter     int64  `json:"prompt_tokens_submitter"`
	CompletionTokensSubmitter int64  `json:"completion_tokens_submitter"`
	CleanerPrompt             string `json:"cleaner_prompt"`
	SubmitterPrompt           string `json:"submitter_prompt"`
	FinalContent              string `json:"final_content"`
}

type SubmitterRes struct {
	Balance  AIresponse `json:"balance"`
	Income   AIresponse `json:"income"`
	CashFlow AIresponse `json:"cash_flow"`
}

func CallSubmitter(
	ctx context.Context,
	ai openai.Client,
	balanceResult string,
	incomeResult string,
	cashFlowResult string,
	period string,
	language string,
	hits Hits,
	unitsFromClient UnitsFromClient,
) (SubmitterRes, error) {

	response := SubmitterRes{}

	type result struct {
		target   string
		response AIresponse
		err      error
	}
	results := make(chan result, 3) // Buffer size 3 to avoid blocking

	activeRoutines := 0

	if hits.Balance < 17 {
		nullJSON, err := createNullJSONResponse("balance")
		if err != nil {
			return SubmitterRes{}, err
		}
		response.Balance = AIresponse{0, 0, 0, 0, "", "", nullJSON}
	} else {
		activeRoutines++
		go func() {
			balanceResp, err := InitiatePipeline(ctx, ai, balanceResult, "balance", period, language, unitsFromClient.Balance)
			results <- result{"balance", balanceResp, err}
		}()
	}

	if hits.Income < 17 {
		nullJSON, err := createNullJSONResponse("income")
		if err != nil {
			return SubmitterRes{}, err
		}
		response.Income = AIresponse{0, 0, 0, 0, "", "", nullJSON}
	} else {
		activeRoutines++
		go func() {
			incomeResp, err := InitiatePipeline(ctx, ai, incomeResult, "income", period, language, unitsFromClient.Income)
			results <- result{"income", incomeResp, err}
		}()
	}

	if hits.CashFlow < 17 {
		nullJSON, err := createNullJSONResponse("cash_flow")
		if err != nil {
			return SubmitterRes{}, err
		}
		response.CashFlow = AIresponse{0, 0, 0, 0, "", "", nullJSON}
	} else {
		activeRoutines++
		go func() {
			cashFlowResp, err := InitiatePipeline(ctx, ai, cashFlowResult, "cash_flow", period, language, unitsFromClient.CashFlow)
			results <- result{"cash_flow", cashFlowResp, err}
		}()
	}

	for range activeRoutines {
		res := <-results
		if res.err != nil {
			return SubmitterRes{}, res.err
		}

		switch res.target {
		case "balance":
			response.Balance = res.response
		case "income":
			response.Income = res.response
		case "cash_flow":
			response.CashFlow = res.response
		}
	}

	return response, nil
}

// *
// **
// ***
// ****
// *****
func InitiatePipeline(
	ctx context.Context,
	ai openai.Client,
	contentToClean string,
	target string,
	period string,
	language string,
	units int64,
) (AIresponse, error) {

	cleanerModel := openai.ChatModelGPT4oMini
	submitterModel := openai.ChatModelGPT4oMini

	response := AIresponse{}

	cleanerPrompt := promptEngineerCleaner(contentToClean, target, units)
	cleanerSystemContent := getSystemPrompt("cleaner", target)

	chatCompletionCleaner, err := ai.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(cleanerSystemContent),
			openai.UserMessage(cleanerPrompt),
		},
		Model:       cleanerModel,
		MaxTokens:   openai.Int(5000),
		Temperature: openai.Float(0.2),
	})

	if err != nil {
		logger.Log.Error("Failed to call OpenAI API", zap.Error(err))
		return AIresponse{}, err
	}

	cleanerResult := ""

	if len(chatCompletionCleaner.Choices) > 0 {
		response.PromptTokensCleaner = int64(chatCompletionCleaner.Usage.PromptTokens)
		response.CompletionTokensCleaner = int64(chatCompletionCleaner.Usage.CompletionTokens)
		cleanerResult = chatCompletionCleaner.Choices[0].Message.Content
	} else {
		logger.Log.Error("OpenAI API returned no choices", zap.Any("response", chatCompletionCleaner))
		return AIresponse{}, fmt.Errorf("OpenAI API returned no choices")
	}

	submitterPrompt := promptEngineerSubmitter(cleanerResult, period)
	submitterSystemContent := getSystemPrompt("submitter", target)

	fields := getRequiredFields(target, units)
	financialSchema := createFinancialDataSchema(fields)

	schemaParam := openai.ResponseFormatJSONSchemaJSONSchemaParam{
		Name:   "financial_data",
		Schema: financialSchema,
		Strict: openai.Bool(true),
	}

	chatCompletionSubmitter, err := ai.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(submitterSystemContent),
			openai.UserMessage(submitterPrompt),
		},
		Model:       submitterModel,
		MaxTokens:   openai.Int(1000),
		Temperature: openai.Float(0.2),
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONSchema: &openai.ResponseFormatJSONSchemaParam{JSONSchema: schemaParam},
		},
	})

	if err != nil {
		logger.Log.Error("Failed to call OpenAI API", zap.Error(err))
		return AIresponse{}, err
	}

	if len(chatCompletionSubmitter.Choices) > 0 {
		response.PromptTokensSubmitter = int64(chatCompletionSubmitter.Usage.PromptTokens)
		response.CompletionTokensSubmitter = int64(chatCompletionSubmitter.Usage.CompletionTokens)
		response.CleanerPrompt = cleanerPrompt
		response.SubmitterPrompt = submitterPrompt
		response.FinalContent = chatCompletionSubmitter.Choices[0].Message.Content
	} else {
		logger.Log.Error("OpenAI API returned no choices", zap.Any("response", chatCompletionSubmitter))
		return AIresponse{}, fmt.Errorf("OpenAI API returned no choices")
	}

	return response, nil
}
