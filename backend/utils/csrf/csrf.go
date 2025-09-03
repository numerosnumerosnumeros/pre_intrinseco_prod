package csrf

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"

	"nodofinance/utils/env"
	"nodofinance/utils/logger"
)

func Generate(value string) (string, error) {
	csrfKey, exists := env.Get("CSRF_SIGNING_KEY")
	if !exists {
		logger.Log.Fatal("Failed to load CSRF signing key")
	}

	h := hmac.New(sha256.New, []byte(csrfKey))
	h.Write([]byte(value))
	csrfToken := hex.EncodeToString(h.Sum(nil))

	return csrfToken, nil
}

func Validate(r *http.Request) bool {
	headerToken := r.Header.Get("x-csrf-Token")
	cookieToken, err := r.Cookie("nodo_csrf_token")

	if headerToken == "" ||
		strings.ToLower(headerToken) == "null" ||
		cookieToken == nil || cookieToken.Value == "" ||
		err != nil {
		return false
	}

	return hmac.Equal([]byte(headerToken), []byte(cookieToken.Value))
}
