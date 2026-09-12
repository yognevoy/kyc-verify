package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"kyc-verify/internal/auth"
	"kyc-verify/internal/usecase"
)

type Deps struct {
	AuthUsecase *usecase.AuthUsecase
	JWTIssuer   *auth.JWTIssuer
}

func NewRouter(deps Deps) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/healthz", handleHealthz)

	authH := &authHandler{auth: deps.AuthUsecase}

	r.Route("/api", func(r chi.Router) {
		r.Post("/auth/register", authH.register)
		r.Post("/auth/login", authH.login)

		r.Group(func(r chi.Router) {
			r.Use(RequireAuth(deps.JWTIssuer))
			r.Get("/me", handleMe)
		})
	})

	return r
}
