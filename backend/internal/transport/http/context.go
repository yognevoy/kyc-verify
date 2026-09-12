package http

import (
	"context"

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
