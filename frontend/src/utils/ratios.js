export function calculateRatiosWithPrice(price, data) {
  if (!price || !data) {
    let ratios = {
      enterpriseValue: null,
      evOcf: null,
      peRatio: null,
      pbRatio: null,
      score: null,
      evMarketCap: null,
      evNetIncome: null,
      enterpriseValue_prev: null,
      evOcf_prev: null,
      peRatio_prev: null,
      pbRatio_prev: null,
      score_prev: null,
      evMarketCap_prev: null,
      evNetIncome_prev: null,
    }
    return ratios
  }

  price = parseFloat(price)
  let sharesOutstanding = data.shares ? Math.floor(Number(data.shares)) : null
  let totalDebt = data.total_liabilities
    ? Math.floor(Number(data.total_liabilities))
    : null
  let cash = data.cash_and_equivalents
    ? Math.floor(Number(data.cash_and_equivalents))
    : null
  let ocf = data.cash_flow_from_operations
    ? Math.floor(Number(data.cash_flow_from_operations))
    : null
  let eps = data.eps ? Number(data.eps) : null
  let vc = data.book_value ? Number(data.book_value) : null
  let netIncome = data.net_income ? Math.floor(Number(data.net_income)) : null
  let score = null
  let marketCap = sharesOutstanding ? price * sharesOutstanding : null
  let enterpriseValue =
    totalDebt && cash && marketCap ? marketCap + totalDebt - cash : null
  let evOcf = ocf && enterpriseValue ? enterpriseValue / ocf : null
  let peRatio = eps ? price / eps : null
  let pbRatio = vc ? price / vc : null
  let evMarketCap =
    marketCap && enterpriseValue ? enterpriseValue / marketCap : null
  let evNetIncome =
    netIncome && enterpriseValue ? enterpriseValue / netIncome : null

  let sharesOutstanding_prev = data.shares_prev
    ? Math.floor(Number(data.shares_prev))
    : null
  let totalDebt_prev = data.total_liabilities_prev
    ? Math.floor(Number(data.total_liabilities_prev))
    : null
  let cash_prev = data.cash_and_equivalents_prev
    ? Math.floor(Number(data.cash_and_equivalents_prev))
    : null
  let ocf_prev = data.cash_flow_from_operations_prev
    ? Math.floor(Number(data.cash_flow_from_operations_prev))
    : null
  let eps_prev = data.eps_prev ? Number(data.eps_prev) : null
  let vc_prev = data.book_value_prev
    ? Math.floor(Number(data.book_value_prev))
    : null
  let netIncome_prev = data.net_income_prev
    ? Math.floor(Number(data.net_income_prev))
    : null
  let marketCap_prev = sharesOutstanding_prev
    ? price * sharesOutstanding_prev
    : null
  let enterpriseValue_prev =
    totalDebt_prev && cash_prev && marketCap_prev
      ? marketCap_prev + totalDebt_prev - cash_prev
      : null
  let evOcf_prev =
    ocf_prev && enterpriseValue_prev ? enterpriseValue_prev / ocf_prev : null
  let peRatio_prev = eps_prev ? price / eps_prev : null
  let pbRatio_prev = vc_prev ? price / vc_prev : null
  let evMarketCap_prev =
    marketCap_prev && enterpriseValue_prev
      ? enterpriseValue_prev / marketCap_prev
      : null
  let evNetIncome_prev =
    netIncome_prev && enterpriseValue_prev
      ? enterpriseValue_prev / netIncome_prev
      : null

  if (
    netIncome === null ||
    sharesOutstanding === null ||
    evMarketCap === null ||
    pbRatio === null
  ) {
    score = null
  } else if (eps <= 0 || vc <= 0 || ocf <= 0 || netIncome <= 0) {
    score = 0
  } else if (enterpriseValue <= 0) {
    score = 10
  } else {
    const reciprocalTransform = (value, max) =>
      value === null ? null : value > max ? 0 : 10 * (1 - value / max)

    const normalizedEvOcf = reciprocalTransform(evOcf, 50)
    const normalizedPeRatio = reciprocalTransform(peRatio, 50)
    const normalizedPbRatio = reciprocalTransform(pbRatio, 20)

    if (normalizedPeRatio === null || normalizedPbRatio === null) {
      score = null
    } else if (normalizedEvOcf == null) {
      const weightPeRatio = 0.5
      const weightPbRatio = 0.5
      score =
        weightPeRatio * normalizedPeRatio + weightPbRatio * normalizedPbRatio
    } else {
      const weightEvOcf = 0.4
      const weightPeRatio = 0.3
      const weightPbRatio = 0.3
      score =
        weightEvOcf * normalizedEvOcf +
        weightPeRatio * normalizedPeRatio +
        weightPbRatio * normalizedPbRatio
    }
  }

  const ratios = {
    enterpriseValue,
    evOcf,
    peRatio,
    pbRatio,
    score,
    evMarketCap,
    evNetIncome,
    enterpriseValue_prev,
    evOcf_prev,
    peRatio_prev,
    pbRatio_prev,
    evMarketCap_prev,
    evNetIncome_prev,
  }

  for (const key in ratios) {
    if (isNaN(ratios[key]) || ratios[key] === null) {
      ratios[key] = null
    } else {
      if (key === 'enterpriseValue' || key === 'enterpriseValue_prev') {
        ratios[key] = Math.floor(ratios[key])
      } else {
        ratios[key] = Math.round(ratios[key] * 100) / 100
      }
    }
  }

  return ratios
}

// Helper function for safe division
function safeDivide(numerator, denominator) {
  if (
    numerator === 0 ||
    numerator === 0.0 ||
    numerator === null ||
    numerator === undefined
  ) {
    return null
  }
  if (
    denominator === 0 ||
    denominator === 0.0 ||
    denominator === null ||
    denominator === undefined
  ) {
    return null
  }
  return Number(numerator) / Number(denominator)
}

// Helper function for safe addition
function safeAdd(a, b) {
  if (a === null || a === undefined || b === null || b === undefined) {
    return null
  }
  return Number(a) + Number(b)
}

function safeSubstract(a, b) {
  if (a === null || a === undefined || b === null || b === undefined) {
    return null
  }
  return Number(a) - Number(b)
}

export function calculateRatios(data) {
  // Early validation of required fields
  if (!data) return null

  // Safe calculations for totals
  const total_assets = safeAdd(data.current_assets, data.non_current_assets)
  const total_assets_prev = safeAdd(
    data.current_assets_prev,
    data.non_current_assets_prev,
  )

  const total_liabilities = safeAdd(
    data.current_liabilities,
    data.non_current_liabilities,
  )
  const total_liabilities_prev = safeAdd(
    data.current_liabilities_prev,
    data.non_current_liabilities_prev,
  )

  const equity = safeSubstract(total_assets, total_liabilities)
  const equity_prev = safeSubstract(total_assets_prev, total_liabilities_prev)

  // Safe calculation for shares
  const shares = safeDivide(data.net_income, data.eps)
  const shares_prev = safeDivide(data.net_income_prev, data.eps_prev)

  const working_capital = safeSubstract(
    data.current_assets,
    data.current_liabilities,
  )
  const working_capital_prev = safeSubstract(
    data.current_assets_prev,
    data.current_liabilities_prev,
  )

  // Adjusting to 2 decimal places without rounding
  const adjustToTwoDecimals = value => {
    if (value === null || value === undefined) return null
    return Math.round(value * 1000) / 1000
  }

  // Safe calculations for ratios
  const wc_ncl = adjustToTwoDecimals(
    safeDivide(working_capital, data.non_current_liabilities),
  )
  const wc_ncl_prev = adjustToTwoDecimals(
    safeDivide(working_capital_prev, data.non_current_liabilities_prev),
  )

  const liquidity = adjustToTwoDecimals(
    safeDivide(data.current_assets, data.current_liabilities),
  )
  const liquidity_prev = adjustToTwoDecimals(
    safeDivide(data.current_assets_prev, data.current_liabilities_prev),
  )

  const leverage = adjustToTwoDecimals(safeDivide(total_liabilities, equity))
  const leverage_prev = adjustToTwoDecimals(
    safeDivide(total_liabilities_prev, equity_prev),
  )

  const solvency = adjustToTwoDecimals(
    safeDivide(total_assets, total_liabilities),
  )
  const solvency_prev = adjustToTwoDecimals(
    safeDivide(total_assets_prev, total_liabilities_prev),
  )

  const net_margin = adjustToTwoDecimals(
    safeDivide(data.net_income, data.revenue),
  )
  const net_margin_prev = adjustToTwoDecimals(
    safeDivide(data.net_income_prev, data.revenue_prev),
  )

  const book_value = adjustToTwoDecimals(safeDivide(equity, shares))
  const book_value_prev = adjustToTwoDecimals(
    safeDivide(equity_prev, shares_prev),
  )

  const roa = adjustToTwoDecimals(safeDivide(data.net_income, total_assets))
  const roa_prev = adjustToTwoDecimals(
    safeDivide(data.net_income_prev, total_assets_prev),
  )

  const roe = adjustToTwoDecimals(safeDivide(data.net_income, equity))
  const roe_prev = adjustToTwoDecimals(
    safeDivide(data.net_income_prev, equity_prev),
  )

  const newData = {
    ...data,
    equity,
    equity_prev,
    total_assets,
    total_assets_prev,
    total_liabilities,
    total_liabilities_prev,
    shares,
    shares_prev,
    working_capital,
    working_capital_prev,
    wc_ncl,
    wc_ncl_prev,
    liquidity,
    liquidity_prev,
    leverage,
    leverage_prev,
    solvency,
    solvency_prev,
    net_margin,
    net_margin_prev,
    book_value,
    book_value_prev,
    roa,
    roa_prev,
    roe,
    roe_prev,
  }

  return newData
}

export const editableFields = [
  'current_assets',
  'non_current_assets',
  'cash_and_equivalents',
  'current_liabilities',
  'non_current_liabilities',
  'revenue',
  'net_income',
  'eps',
  'cash_flow_from_operations',
  'cash_flow_from_investing',
  'cash_flow_from_financing',
]

export function formatForRatios(data) {
  const formattedData = {}
  for (const key in data) {
    if (key === 'eps') {
      formattedData[key] = Number(data[key])
    } else {
      formattedData[key] = Math.floor(Number(data[key]))
    }
  }
  return formattedData
}

export function wishedPER(wishedPER, eps, currency) {
  let currencySymbol = ''
  switch (currency) {
    case 'USD':
      currencySymbol = '$'
      break
    case 'EUR':
      currencySymbol = '€'
      break
    case 'GBP':
      currencySymbol = '£'
      break
    case 'NA':
      currencySymbol = 'NA'
      break
    default:
      currencySymbol = ''
  }
  const epsNumber = Number(eps)
  const wishedPERNumber = Number(wishedPER)

  if (
    !wishedPER ||
    !eps ||
    isNaN(epsNumber) ||
    isNaN(wishedPERNumber) ||
    epsNumber <= 0 ||
    wishedPERNumber < 0
  ) {
    return currencySymbol
  }

  return `${Math.round(Number(wishedPER) * epsNumber).toLocaleString('en-US', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  })}${currencySymbol}`
}

export function revenueNeededForWishedPER(
  price,
  wishedPER,
  shares,
  currency,
  currentIncome,
) {
  if (!price || !wishedPER || !shares || !currentIncome)
    return { value: 'π', percentChange: null }

  const priceNumber = Number(price)
  const wishedPERNumber = Number(wishedPER)
  const sharesNumber = Number(shares)
  const currentIncomeNumber = Number(currentIncome)

  if (
    isNaN(priceNumber) ||
    isNaN(wishedPERNumber) ||
    isNaN(sharesNumber) ||
    wishedPERNumber <= 0 ||
    sharesNumber <= 0
  ) {
    return { value: 'π', percentChange: null }
  }

  let currencySymbol = 'NA'
  switch (currency) {
    case 'USD':
      currencySymbol = '$'
      break
    case 'EUR':
      currencySymbol = '€'
      break
    case 'GBP':
      currencySymbol = '£'
      break
    default:
      currencySymbol = 'NA'
  }

  const requiredEPS = priceNumber / wishedPERNumber
  const requiredNetIncome = Math.round(requiredEPS * sharesNumber)

  const percentChange = currentIncomeNumber
    ? Math.round(
        ((requiredNetIncome - currentIncomeNumber) /
          Math.abs(currentIncomeNumber)) *
          1000,
      ) / 10
    : null

  let formattedValue
  if (requiredNetIncome >= 1000000) {
    const millionValue = Math.round(requiredNetIncome / 1000000)
    formattedValue = `${millionValue.toLocaleString('en-US', {
      maximumFractionDigits: 0,
      useGrouping: true,
    })}M${currencySymbol}`
  } else {
    formattedValue = `${requiredNetIncome.toLocaleString('en-US', {
      maximumFractionDigits: 0,
      useGrouping: true,
    })}${currencySymbol}`
  }

  return {
    value: formattedValue,
    percentChange: percentChange,
  }
}
