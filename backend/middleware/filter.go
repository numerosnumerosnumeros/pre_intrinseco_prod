package middleware

import (
	"net/http"
	"slices"
	"strings"

	"nodofinance/utils/jwt"
)

type FilterConfig struct {
	AllowedMethods  []string
	DevMode         bool
	ValidateRequest bool
	CheckPremium    bool
}

func Filter(next http.HandlerFunc, config FilterConfig) http.HandlerFunc {
	const MAX_URL_LENGTH = 2048 // 2KB

	return func(w http.ResponseWriter, r *http.Request) {

		if len(r.URL.String()) > MAX_URL_LENGTH {
			http.Error(w, "URL too long", http.StatusBadRequest)
			return
		}

		if config.DevMode {
			w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
			w.Header().Set("Access-Control-Allow-Methods", "OPTIONS, "+strings.Join(config.AllowedMethods, ", "))
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Csrf-Token, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Max-Age", "3600")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
		}

		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")

		if !slices.Contains(config.AllowedMethods, r.Method) {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		if config.ValidateRequest {
			if !jwt.ValidateRequest(r, config.CheckPremium) {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}

		next(w, r)
	}
}
