package http

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"kyc-verify/internal/auth"
	"kyc-verify/internal/usecase"
)

type Deps struct {
	AuthUsecase         *usecase.AuthUsecase
	ApplicantUsecase    *usecase.ApplicantUsecase
	DocumentUsecase     *usecase.DocumentUsecase
	VerificationUsecase *usecase.VerificationUsecase
	JWTIssuer           *auth.JWTIssuer
	RefreshTTL          time.Duration
	CookieSecure        bool
}

func NewRouter(deps Deps) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/healthz", handleHealthz)

	authH := &authHandler{
		auth:         deps.AuthUsecase,
		refreshTTL:   deps.RefreshTTL,
		cookieSecure: deps.CookieSecure,
	}
	applicantH := &applicantHandler{
		applicants: deps.ApplicantUsecase,
	}
	documentH := &documentHandler{
		applicants: deps.ApplicantUsecase,
		documents:  deps.DocumentUsecase,
	}
	verificationH := &verificationHandler{
		applicants:   deps.ApplicantUsecase,
		verification: deps.VerificationUsecase,
	}

	r.Route("/api", func(r chi.Router) {
		r.Post("/auth/register", authH.register)
		r.Post("/auth/login", authH.login)
		r.Post("/auth/refresh", authH.refresh)
		r.Post("/auth/logout", authH.logout)

		r.Group(func(r chi.Router) {
			r.Use(RequireAuth(deps.JWTIssuer))
			r.Get("/me", handleMe)
			r.Post("/applicants", applicantH.create)
			r.Get("/applicants/me", applicantH.getMe)
			r.Put("/applicants/me", applicantH.updateMe)
			r.Post("/documents", documentH.upload)
			r.Get("/documents/me", documentH.listMine)
			r.Post("/verification-cases", verificationH.submit)
			r.Get("/verification-cases/me", verificationH.getMe)
		})
	})

	return r
}
