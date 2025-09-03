const isDev =
  import.meta.env.MODE === 'development' ||
  import.meta.env.VITE_USE_DEV === 'true'

export const config = {
  baseURL: isDev
    ? import.meta.env.VITE_BASE_URL_DEV
    : import.meta.env.VITE_BASE_URL,
  stripeKey: isDev
    ? import.meta.env.VITE_STRIPE_PK_TEST
    : import.meta.env.VITE_STRIPE_PK,
  contactEmail: import.meta.env.VITE_EMAIL_CONTACT,
  maxTickers: import.meta.env.VITE_MAX_TICKERS,
  maxPeriods: import.meta.env.VITE_MAX_PERIODS_PER_TICKER,
}
