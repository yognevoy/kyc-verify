package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
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
	ctx := context.Background()
	if err := run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cfg := config.Load()

	if err := repository.Migrate(cfg.DatabaseURL); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	dbCtx, dbCancel := context.WithTimeout(ctx, 5*time.Second)
	pool, err := repository.Connect(dbCtx, cfg.DatabaseURL)
	dbCancel()
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer pool.Close()

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
	casePool := worker.NewPool(256, func(ctx context.Context, caseID uuid.UUID) {
		verificationUsecase.Process(ctx, caseID)
	})
	verificationUsecase = usecase.NewVerificationUsecase(caseRepo, documentRepo, applicantRepo, mockProvider, casePool)
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
			CORSAllowedOrigin:   cfg.CORSAllowedOrigin,
		}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("api listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)
	case <-ctx.Done():
		log.Println("shutting down")
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown failed: %w", err)
	}

	casePool.Stop()
	return nil
}
