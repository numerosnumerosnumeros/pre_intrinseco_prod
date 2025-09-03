package payments

import (
	"fmt"
	"nodofinance/utils/env"
)

var (
	PRICE_ID        string
	BASE_URL        string
	ENDPOINT_SECRET string
)

func init() {
	env.RegisterValidator(validateVar)
}

func validateVar() error {
	var exists bool

	PRICE_ID, exists = env.Get("STRIPE_PRICE_ID")
	if !exists {
		return fmt.Errorf("missing required environment variable: STRIPE_PRICE_ID")
	}

	BASE_URL, exists = env.Get("VITE_BASE_URL")
	if !exists {
		return fmt.Errorf("missing required environment variable: VITE_BASE_URL")
	}

	ENDPOINT_SECRET, exists = env.Get("STRIPE_WEBHOOK_SECRET")
	if !exists {
		return fmt.Errorf("missing required environment variable: STRIPE_WEBHOOK_SECRET")
	}

	return nil
}
