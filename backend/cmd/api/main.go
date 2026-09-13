package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"

	"kyc-verify/internal/auth"
	"kyc-verify/internal/config"
	"kyc-verify/internal/provider"
	"kyc-verify/internal/repository"
	"kyc-verify/internal/storage"
	httptransport "kyc-verify/internal/transport/http"
	"kyc-verify/internal/usecase"
	"kyc-verify/internal/worker"
)

func main() {
	cfg := config.Load()

	if err := repository.Migrate(cfg.DatabaseURL); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	dbCtx, dbCancel := context.WithTimeout(context.Background(), 5*time.Second)
	pool, err := repository.Connect(dbCtx, cfg.DatabaseURL)
	dbCancel()
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer pool.Close()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	jwtIssuer := auth.NewJWTIssuer(cfg.JWTSecret, cfg.JWTTTL)
	userRepo := repository.NewUserRepository(pool)
	refreshTokenRepo := repository.NewRefreshTokenRepository(pool)
	authUsecase := usecase.NewAuthUsecase(userRepo, refreshTokenRepo, jwtIssuer, cfg.RefreshTTL)

	applicantRepo := repository.NewApplicantRepository(pool)
	applicantUsecase := usecase.NewApplicantUsecase(applicantRepo)

	documentRepo := repository.NewDocumentRepository(pool)
	localStorage := storage.NewLocalStorage(cfg.UploadDir)
	documentUsecase := usecase.NewDocumentUsecase(documentRepo, localStorage)

	caseRepo := repository.NewVerificationCaseRepository(pool)
	mockProvider := provider.NewMockProvider(cfg.ProviderMinDelay, cfg.ProviderMaxDelay, cfg.ProviderApproveChance)

	var verificationUsecase *usecase.VerificationUsecase
	casePool := worker.NewPool[uuid.UUID](256, func(ctx context.Context, caseID uuid.UUID) {
		verificationUsecase.Process(ctx, caseID)
	})
	verificationUsecase = usecase.NewVerificationUsecase(caseRepo, documentRepo, mockProvider, casePool)
	casePool.Start(ctx, cfg.WorkerPoolSize)

	srv := &http.Server{
		Addr: ":" + cfg.HTTPPort,
		Handler: httptransport.NewRouter(httptransport.Deps{
			AuthUsecase:         authUsecase,
			ApplicantUsecase:    applicantUsecase,
			DocumentUsecase:     documentUsecase,
			VerificationUsecase: verificationUsecase,
			JWTIssuer:           jwtIssuer,
			RefreshTTL:          cfg.RefreshTTL,
			CookieSecure:        cfg.CookieSecure,
		}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("api listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("graceful shutdown failed: %v", err)
	}

	casePool.Stop()
}
