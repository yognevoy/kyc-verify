package http

import (
	"context"
	"net/http"

	"kyc-verify/internal/auth"
)

type contextKey int

const claimsContextKey contextKey = iota

func withClaims(ctx context.Context, claims auth.Claims) context.Context {
	return context.WithValue(ctx, claimsContextKey, claims)
}

func ClaimsFromContext(ctx context.Context) (auth.Claims, bool) {
	claims, ok := ctx.Value(claimsContextKey).(auth.Claims)
	return claims, ok
}

func requireClaims(w http.ResponseWriter, r *http.Request) (auth.Claims, bool) {
	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing claims")
	}
	return claims, ok
}
