package sanitize

import (
	"encoding/base64"
	"encoding/json"
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// Compile at the package level to avoid re-compilation on each function call
var (
	emailRegex      = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	specialChars    = "@$!%*?&"
	blockedPatterns = []string{
		"localhost", "127.", "192.168.", "10.", "172.16.", "0.0.0.0", "169.254.",
	}
	validDomainChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-._"
	allowedFields    = map[string]FieldConstraint{
		"current_assets":            {0, 0, SafeMax, false},
		"non_current_assets":        {0, 0, SafeMax, false},
		"eps":                       {4, EpsMin, EpsMax, true},
		"cash_and_equivalents":      {0, 0, SafeMax, false},
		"cash_flow_from_financing":  {0, SafeMin, SafeMax, true},
		"cash_flow_from_investing":  {0, SafeMin, SafeMax, true},
		"cash_flow_from_operations": {0, SafeMin, SafeMax, true},
		"revenue":                   {0, 0, SafeMax, false},
		"current_liabilities":       {0, 0, SafeMax, false},
		"non_current_liabilities":   {0, 0, SafeMax, false},
		"net_income":                {0, SafeMin, SafeMax, true},
	}
)

type FieldConstraint struct {
	Decimals      int
	MinValue      float64
	MaxValue      float64
	AllowNegative bool
}

const (
	SafeMax = 1e14  // 100 trillion
	SafeMin = -1e14 // -100 trillion
	EpsMax  = 1e5   // $100,000 per share
	EpsMin  = -1e5  // -$100,000 per share
)

// *
// **
// ***
// ****
// ***** FORMAT
func Trim(s string, mode string) string {
	s = strings.TrimSpace(s)
	if s == "" || mode == "" {
		return s
	}

	result := strings.Builder{}
	result.Grow(len(s))

	for _, c := range s {
		if c <= 127 {
			switch mode {
			case "u":
				result.WriteRune(unicode.ToUpper(c))
			case "l":
				result.WriteRune(unicode.ToLower(c))
			default:
				result.WriteRune(c)
			}
		} else {
			result.WriteRune(c)
		}
	}
	return result.String()
}

// *
// **
// ***
// ****
// ***** VALIDATE
func Email(email string) bool {
	if len(email) < 6 {
		return false
	}

	if idx := strings.IndexByte(email, '@'); idx <= 0 || idx == len(email)-1 {
		return false
	}

	return emailRegex.MatchString(email)
}

func Bool(value string) bool {
	return value == "true" || value == "false"
}

func Language(value string) bool {
	return value == "ES" || value == "EN"
}

func Units(value int64) bool {
	return value == 0 || value == 1000 || value == 1000000
}

func Currency(value string) bool {
	return value == "USD" || value == "EUR" || value == "GBP" || value == "ND"
}

func Password(password string) bool {
	if len(password) < 8 || strings.ContainsAny(password, " \t\n\r\f\v") {
		return false
	}

	hasLower, hasUpper, hasDigit, hasSpecial := false, false, false, false

	for _, ch := range password {
		switch {
		case unicode.IsLower(ch):
			hasLower = true
		case unicode.IsUpper(ch):
			hasUpper = true
		case unicode.IsDigit(ch):
			hasDigit = true
		case strings.ContainsRune(specialChars, ch):
			hasSpecial = true
		}

		if hasLower && hasUpper && hasDigit && hasSpecial {
			break
		}
	}

	return hasLower && hasUpper && hasDigit && hasSpecial
}

func Code(code string) bool {
	if len(code) != 6 || strings.ContainsAny(code, " \t\n\r\f\v") {
		return false
	}

	for _, ch := range code {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

func Timestamp(timestamp string) bool {
	if len(timestamp) != 10 {
		return false
	}

	for _, ch := range timestamp {
		if !unicode.IsDigit(ch) {
			return false
		}
	}

	return true
}

func Ticker(ticker string) bool {
	if len(ticker) == 0 || len(ticker) > 12 {
		return false
	}

	if ticker[0] == '.' || ticker[len(ticker)-1] == '.' {
		return false
	}

	for _, ch := range ticker {
		if !unicode.IsUpper(ch) && !unicode.IsDigit(ch) && ch != '.' {
			return false
		}
	}

	return true
}

func Hex(token string) bool {
	if len(token) != 32 {
		return false
	}

	for _, ch := range token {
		if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') {
			return false
		}
	}

	return true
}

func Period(period string) bool {
	if len(period) != 6 && len(period) != 7 {
		return false
	}

	for i := range 4 {
		if period[i] < '0' || period[i] > '9' {
			return false
		}
	}

	year, err := strconv.Atoi(period[:4])
	if err != nil || year < 1900 {
		return false
	}

	if period[4] != '-' {
		return false
	}

	switch period[5] {
	case 'Y':
		return len(period) == 6
	case 'Q':
		return len(period) == 7 && period[6] >= '1' && period[6] <= '4'
	case 'S':
		return len(period) == 7 && period[6] >= '1' && period[6] <= '2'
	default:
		return false
	}
}

func FinancialData(financialData map[string]any) bool {
	if len(financialData) != 11 {
		return false
	}

	for fieldName, value := range financialData {
		constraints, ok := allowedFields[fieldName]
		if !ok {
			return false
		}

		if value == nil {
			continue
		}

		var numValue float64

		switch v := value.(type) {
		case string:
			if v == "" {
				continue
			}

			if len(v) > 25 {
				return false
			}

			var builder strings.Builder
			builder.Grow(len(v))

			dotCount := 0
			minusCount := 0

			for i, c := range v {
				if unicode.IsSpace(c) {
					continue // Skip whitespace
				}
				if c == ',' {
					continue // Skip commas
				}

				if c == '.' {
					dotCount++
					if dotCount > 1 {
						return false // More than one dot
					}
				} else if c == '-' {
					minusCount++
					if minusCount > 1 || i > 0 {
						return false // Multiple minus signs or minus not at beginning
					}
				} else if !unicode.IsDigit(c) {
					return false // Invalid character
				}

				builder.WriteRune(c)
			}

			valStr := builder.String()
			if valStr == "" {
				continue // If after processing it's empty, treat as null
			}

			// Validate leading zeros
			if fieldName != "eps" && constraints.Decimals == 0 {
				if len(valStr) > 1 && valStr[0] == '0' {
					return false // Leading zero in regular field
				}
				if len(valStr) > 2 && valStr[0] == '-' && valStr[1] == '0' {
					return false // Leading zero after minus
				}
			} else if fieldName == "eps" {
				dotPos := strings.Index(valStr, ".")
				var integerPart string
				if dotPos != -1 {
					integerPart = valStr[:dotPos]
				} else {
					integerPart = valStr
				}

				if len(integerPart) > 1 && integerPart[0] == '0' {
					return false // Leading zero in eps
				}
				if len(integerPart) > 2 && integerPart[0] == '-' && integerPart[1] == '0' {
					return false // Leading zero after minus in eps
				}
			}

			var err error
			numValue, err = strconv.ParseFloat(valStr, 64)
			if err != nil {
				return false
			}

		case float64:
			numValue = v

		default:
			return false // Not a string or number
		}

		// Check for NaN or Inf
		if math.IsNaN(numValue) || math.IsInf(numValue, 0) {
			return false
		}

		// Check for negative values
		if !constraints.AllowNegative && numValue < 0 {
			return false
		}

		// Check range
		if numValue < constraints.MinValue || numValue > constraints.MaxValue {
			return false
		}

		// Check decimal places
		factor := math.Pow(10.0, float64(constraints.Decimals))
		temp := numValue * factor
		if math.Abs(temp-math.Round(temp)) > 1e-6 {
			return false // More decimal places than allowed
		}
	}

	return true
}

func URL(value string) bool {
	const minLength = 3
	const maxLength = 2048

	// Length check
	if len(value) < minLength || len(value) > maxLength {
		return false
	}

	// Block localhost and private networks
	for _, pattern := range blockedPatterns {
		if strings.Contains(value, pattern) {
			return false
		}
	}

	// Verify protocol
	if strings.HasPrefix(value, "file://") {
		return false
	}

	hasProtocol := strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://")

	// Find domain start
	domainStart := 0
	if hasProtocol {
		protoSep := strings.Index(value, "://")
		if protoSep == -1 {
			return false
		}
		domainStart = protoSep + 3
	}

	if domainStart >= len(value) {
		return false
	}

	// Find domain end (first slash after domain or end of string)
	remainder := value[domainStart:]
	slashIndex := strings.IndexByte(remainder, '/')
	var domain string
	if slashIndex == -1 {
		domain = remainder
	} else {
		domain = remainder[:slashIndex]
	}

	// Domain basic checks
	if len(domain) == 0 || len(domain) > 255 {
		return false
	}

	// Check for valid domain characters
	for _, char := range domain {
		if !strings.ContainsRune(validDomainChars, char) {
			return false
		}
	}

	// Domain must have at least one dot
	if !strings.Contains(domain, ".") {
		return false
	}

	// Domain can't start or end with dot or hyphen
	if domain[0] == '.' || domain[len(domain)-1] == '.' ||
		domain[0] == '-' || domain[len(domain)-1] == '-' {
		return false
	}

	// Validate TLD (at least 2 characters)
	lastDot := strings.LastIndex(domain, ".")
	if lastDot == -1 || lastDot >= len(domain)-2 {
		return false
	}

	return true
}

func Cursor(value string) bool {
	// Basic length check
	if len(value) == 0 || len(value) > 1024 {
		return false
	}

	// Must be valid base64 URL encoding
	jsonData, err := base64.URLEncoding.DecodeString(value)
	if err != nil {
		return false
	}

	// Must be valid JSON
	var cursorData map[string]interface{}
	if err := json.Unmarshal(jsonData, &cursorData); err != nil {
		return false
	}

	// Basic structure check - should have our expected keys
	if _, hasUsername := cursorData["username"]; !hasUsername {
		return false
	}

	if _, hasCompositeKey := cursorData["composite_sk"]; !hasCompositeKey {
		return false
	}

	return true
}
