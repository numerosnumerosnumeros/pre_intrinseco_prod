export function hasMinLength(password) {
  return password.length >= 8
}

export function hasUpperCase(password) {
  return /[A-Z]/.test(password)
}

export function hasLowerCase(password) {
  return /[a-z]/.test(password)
}

export function hasNumber(password) {
  return /\d/.test(password)
}

export function hasSpecialChar(password) {
  return /[@$!%*?&]/.test(password)
}

export function isValidPassword(password) {
  return /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)(?=.*[@$!%*?&])[A-Za-z\d@$!%*?&]{8,}$/.test(
    password,
  )
}

export function isValidEmail(email) {
  return /^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/.test(email)
}

export function isValidNumber(number) {
  return /^\d{6}$/.test(number)
}

// Define field constraints
const ALLOWED_FIELDS = {
  current_assets: {
    decimals: 0,
    minValue: 0,
    maxValue: 1e14,
    allowNegative: false,
  },
  non_current_assets: {
    decimals: 0,
    minValue: 0,
    maxValue: 1e14,
    allowNegative: false,
  },
  eps: { decimals: 4, minValue: -1e14, maxValue: 1e14, allowNegative: true },
  cash_and_equivalents: {
    decimals: 0,
    minValue: 0,
    maxValue: 1e14,
    allowNegative: false,
  },
  cash_flow_from_financing: {
    decimals: 0,
    minValue: -1e14,
    maxValue: 1e14,
    allowNegative: true,
  },
  cash_flow_from_investing: {
    decimals: 0,
    minValue: -1e14,
    maxValue: 1e14,
    allowNegative: true,
  },
  cash_flow_from_operations: {
    decimals: 0,
    minValue: -1e14,
    maxValue: 1e14,
    allowNegative: true,
  },
  revenue: { decimals: 0, minValue: 0, maxValue: 1e14, allowNegative: false },
  current_liabilities: {
    decimals: 0,
    minValue: 0,
    maxValue: 1e14,
    allowNegative: false,
  },
  non_current_liabilities: {
    decimals: 0,
    minValue: 0,
    maxValue: 1e14,
    allowNegative: false,
  },
  net_income: {
    decimals: 0,
    minValue: -1e14,
    maxValue: 1e14,
    allowNegative: true,
  },
}

export function isValidFinancialData(value, fieldName) {
  const constraints = ALLOWED_FIELDS[fieldName]
  if (!constraints) return false

  if (value === null || (typeof value === 'string' && value.trim() === ''))
    return true

  // Remove all whitespace
  let valStr = String(value).replace(/\s/g, '')

  if (valStr.length > 50) return false // Reject excessively long inputs

  const numValue = parseFloat(valStr)
  if (isNaN(numValue) || !isFinite(numValue)) return false

  if (!constraints.allowNegative && numValue < 0) return false

  if (numValue < constraints.minValue || numValue > constraints.maxValue)
    return false

  const factor = Math.pow(10, constraints.decimals)
  const temp = numValue * factor
  if (Math.abs(temp - Math.round(temp)) > 1e-6) {
    return false // More decimal places than allowed
  }

  return true
}

export function filterNumericValues(data) {
  if (typeof data !== 'object' || data === null) return null

  const fieldCount = Object.keys(data).length
  if (fieldCount !== 11) return null

  const numericData = {}
  for (const [key, value] of Object.entries(data)) {
    const constraints = ALLOWED_FIELDS[key]
    if (!constraints) {
      return null // Invalid field
    }

    if (value === null || value === '') {
      // Include null values
      numericData[key] = null
      continue
    }

    if (!isValidFinancialData(value, key)) {
      return null // Return null immediately if any field is invalid
    }

    let numValue = parseFloat(String(value))

    // Round to required decimal places
    const factor = Math.pow(10, constraints.decimals)
    numValue = Math.round(numValue * factor) / factor

    numericData[key] = numValue
  }

  return numericData
}

export function validatePeriod(period) {
  return /^\d{4}-(?:Q[1-4]|S[1-2]|Y)$/.test(period)
}

export function validateTicker(ticker) {
  if (ticker.length === 1) {
    return /^[A-Z0-9]$/.test(ticker)
  }
  return /^[A-Z0-9][A-Z0-9.]{0,10}[A-Z0-9]$/.test(ticker)
}

export function validateCurrency(currency) {
  return /^(GBP|USD|EUR|NA)$/.test(currency)
}
