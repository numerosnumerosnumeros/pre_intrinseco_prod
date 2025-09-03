package auth

import (
	"net/http"
	"time"
)

func SignOut(w http.ResponseWriter, r *http.Request) {
	expiredTime := time.Now().AddDate(0, 0, -1)

	cookies := []struct {
		name     string
		httpOnly bool
	}{
		{"nodo_id_token", true},
		{"nodo_access_token", true},
		{"nodo_refresh_token", true},
		{"nodo_csrf_token", false},
	}

	for _, cookie := range cookies {
		http.SetCookie(w, &http.Cookie{
			Name:     cookie.name,
			Value:    "",
			Path:     "/",
			Expires:  expiredTime,
			Secure:   true,
			HttpOnly: cookie.httpOnly,
			SameSite: http.SameSiteStrictMode,
		})
	}

	w.WriteHeader(http.StatusOK)
}
