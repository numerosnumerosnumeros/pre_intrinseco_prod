package submitter

import (
	"math"
	"nodofinance/utils/logger"

	"go.uber.org/zap"
)

type Postprocessed struct {
	CurrentAssets          *int64   `json:"current_assets"`
	Cash                   *int64   `json:"cash_and_equivalents"`
	NonCurrentAssets       *int64   `json:"non_current_assets"`
	CurrentLiabilities     *int64   `json:"current_liabilities"`
	NonCurrentLiabilities  *int64   `json:"non_current_liabilities"`
	CashFlowFromOperations *int64   `json:"cash_flow_from_operations"`
	CashFlowFromInvesting  *int64   `json:"cash_flow_from_investing"`
	CashFlowFromFinancing  *int64   `json:"cash_flow_from_financing"`
	EPS                    *float64 `json:"eps"`
	Revenue                *int64   `json:"revenue"`
	NetIncome              *int64   `json:"net_income"`
}

const (
	SAFEMAX int64   = 1e14  // 100 trillion
	SAFEMIN int64   = -1e14 // -100 trillion
	BPAMAX  float64 = 1e5   // $100,000 per share
	BPAMIN  float64 = -1e5  // -$100,000 per share
)

type IntegerConstraint struct {
	MinValue      int64
	MaxValue      int64
	AllowNegative bool
}

type DoubleConstraint struct {
	MinValue      float64
	MaxValue      float64
	AllowNegative bool
}

type FieldConstraint struct {
	IsDouble   bool
	Constraint interface{}
}

var Constraints = map[string]FieldConstraint{
	// {
	// decimals,
	// min, max, negative
	// }
	"current_assets": {
		IsDouble:   false,
		Constraint: IntegerConstraint{0, SAFEMAX, false},
	},
	"non_current_assets": {
		IsDouble:   false,
		Constraint: IntegerConstraint{0, SAFEMAX, false},
	},
	"eps": {
		IsDouble:   true,
		Constraint: DoubleConstraint{BPAMIN, BPAMAX, true},
	},
	"cash_and_equivalents": {
		IsDouble:   false,
		Constraint: IntegerConstraint{0, SAFEMAX, false},
	},
	"cash_flow_from_financing": {
		IsDouble:   false,
		Constraint: IntegerConstraint{SAFEMIN, SAFEMAX, true},
	},
	"cash_flow_from_investing": {
		IsDouble:   false,
		Constraint: IntegerConstraint{SAFEMIN, SAFEMAX, true},
	},
	"cash_flow_from_operations": {
		IsDouble:   false,
		Constraint: IntegerConstraint{SAFEMIN, SAFEMAX, true},
	},
	"revenue": {
		IsDouble:   false,
		Constraint: IntegerConstraint{0, SAFEMAX, false},
	},
	"current_liabilities": {
		IsDouble:   false,
		Constraint: IntegerConstraint{0, SAFEMAX, false},
	},
	"non_current_liabilities": {
		IsDouble:   false,
		Constraint: IntegerConstraint{0, SAFEMAX, false},
	},
	"net_income": {
		IsDouble:   false,
		Constraint: IntegerConstraint{SAFEMIN, SAFEMAX, true},
	},
}

// *
// **
// ***
// ****
// ***** HELPERS
func getUnitValue(units *float64) float64 {
	const maxUnits float64 = 1000000000

	if units == nil {
		return 1.0
	}

	floatVal := *units
	floatVal = math.Abs(floatVal)

	// Check that value is not zero, not too large, and not NaN/Infinity
	if floatVal < 1 ||
		floatVal > maxUnits ||
		math.IsNaN(floatVal) ||
		math.IsInf(floatVal, 0) {
		return 1.0
	}

	return floatVal
}

func addInt64Value(fieldName string, inputValue *float64, units float64) *int64 {
	if inputValue == nil {
		return nil
	}

	intConstraint := Constraints[fieldName].Constraint.(IntegerConstraint)

	if math.IsNaN(*inputValue) || math.IsInf(*inputValue, 0) {
		return nil
	}

	if *inputValue > float64(math.MaxInt64) || *inputValue < float64(math.MinInt64) {
		return nil
	}

	scaledValue := *inputValue * units

	if scaledValue > float64(math.MaxInt64) || scaledValue < float64(math.MinInt64) {
		return nil
	}

	intValue := int64(scaledValue)

	if intValue < 0 && !intConstraint.AllowNegative {
		intValue = -intValue
	}

	if intValue < intConstraint.MinValue || intValue > intConstraint.MaxValue {
		return nil
	}

	return &intValue
}

// *
// **
// ***
// ****
// *****
func Postprocessor(balance *BalanceSheet, income *IncomeStatement, cashFlow *CashFlowStatement) (Postprocessed, error) {
	balanceUnits := getUnitValue(balance.Units)
	incomeUnits := getUnitValue(income.Units)
	cashFlowUnits := getUnitValue(cashFlow.Units)

	if balanceUnits != incomeUnits {
		logger.Log.Error("Balance and Income units do not match", zap.Float64("balanceUnits", balanceUnits), zap.Float64("incomeUnits", incomeUnits))
	}
	if balanceUnits != cashFlowUnits {
		logger.Log.Error("Balance and Cash Flow units do not match", zap.Float64("balanceUnits", balanceUnits), zap.Float64("cashFlowUnits", cashFlowUnits))
	}
	if incomeUnits != cashFlowUnits {
		logger.Log.Error("Income and Cash Flow units do not match", zap.Float64("incomeUnits", incomeUnits), zap.Float64("cashFlowUnits", cashFlowUnits))
	}

	postprocessed := Postprocessed{}

	// Add float64 values
	if income.EPS != nil {
		if !math.IsNaN(*income.EPS) && !math.IsInf(*income.EPS, 0) {
			minValue := Constraints["eps"].Constraint.(DoubleConstraint).MinValue
			maxValue := Constraints["eps"].Constraint.(DoubleConstraint).MaxValue

			if *income.EPS >= minValue && *income.EPS <= maxValue {
				epsCopy := *income.EPS
				postprocessed.EPS = &epsCopy
			}
		}
	}

	// Add int64 values
	postprocessed.NetIncome = addInt64Value("net_income", income.NetIncome, incomeUnits)
	postprocessed.Revenue = addInt64Value("revenue", income.Revenue, incomeUnits)
	postprocessed.CashFlowFromOperations = addInt64Value("cash_flow_from_operations", cashFlow.CashFlowFromOperations, cashFlowUnits)
	postprocessed.CashFlowFromInvesting = addInt64Value("cash_flow_from_investing", cashFlow.CashFlowFromInvesting, cashFlowUnits)
	postprocessed.CashFlowFromFinancing = addInt64Value("cash_flow_from_financing", cashFlow.CashFlowFromFinancing, cashFlowUnits)
	postprocessed.Cash = addInt64Value("cash_and_equivalents", balance.CashAndEquivalents, balanceUnits)

	// Add int64 assets (with fallbacks). Prefer calculated values over raw values
	postprocessed.CurrentAssets = addInt64Value("current_assets", balance.CurrentAssets, balanceUnits)
	if balance.TotalAssets == nil {
		postprocessed.NonCurrentAssets = addInt64Value("non_current_assets", balance.NonCurrentAssets, balanceUnits)
	} else if balance.CurrentAssets != nil {
		nonCurrentAssets := *balance.TotalAssets - *balance.CurrentAssets
		postprocessed.NonCurrentAssets = addInt64Value("non_current_assets", &nonCurrentAssets, balanceUnits)
	}

	// Add int64 liabilities (with fallbacks)
	postprocessed.CurrentLiabilities = addInt64Value("current_liabilities", balance.CurrentLiabilities, balanceUnits)
	if balance.TotalLiabilities != nil && balance.CurrentLiabilities != nil && (math.Abs(*balance.TotalLiabilities)-math.Abs(*balance.CurrentLiabilities) > 0) {
		if balance.CurrentLiabilities != nil {
			nonCurrentLiabilities := math.Abs(*balance.TotalLiabilities) - math.Abs(*balance.CurrentLiabilities)
			postprocessed.NonCurrentLiabilities = addInt64Value("non_current_liabilities", &nonCurrentLiabilities, balanceUnits)
		}
	} else {
		if balance.NonCurrentLiabilities != nil {
			postprocessed.NonCurrentLiabilities = addInt64Value("non_current_liabilities", balance.NonCurrentLiabilities, balanceUnits)
		} else {
			totalAssets := 0.0

			if balance.TotalAssets != nil {
				totalAssets = *balance.TotalAssets
			} else {
				if balance.CurrentAssets != nil && balance.NonCurrentAssets != nil {
					totalAssets = *balance.CurrentAssets + *balance.NonCurrentAssets
				}
			}

			if balance.Equity != nil && balance.CurrentLiabilities != nil && totalAssets != 0 {
				nonCurrentLiabilities := totalAssets - *balance.Equity - math.Abs(*balance.CurrentLiabilities)
				if nonCurrentLiabilities > 0 {
					postprocessed.NonCurrentLiabilities = addInt64Value("non_current_liabilities", &nonCurrentLiabilities, balanceUnits)
				}
			}
		}
	}

	if postprocessed.NonCurrentLiabilities == nil {
		postprocessed.NonCurrentLiabilities = addInt64Value("non_current_liabilities", balance.NonCurrentLiabilities, balanceUnits)
	}

	return postprocessed, nil
}
