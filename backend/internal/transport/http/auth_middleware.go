package http

import (
	"net/http"
	"strings"

	"kyc-verify/internal/auth"
)

func RequireAuth(issuer *auth.JWTIssuer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			token, ok := strings.CutPrefix(header, "Bearer ")
			if !ok || token == "" {
				writeError(w, http.StatusUnauthorized, "missing bearer token")
				return
			}

			claims, err := issuer.Parse(token)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			next.ServeHTTP(w, r.WithContext(withClaims(r.Context(), claims)))
		})
	}
}
