package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"kyc-verify/internal/auth"
	"kyc-verify/internal/config"
	"kyc-verify/internal/repository"
	httptransport "kyc-verify/internal/transport/http"
	"kyc-verify/internal/usecase"
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

	jwtIssuer := auth.NewJWTIssuer(cfg.JWTSecret, cfg.JWTTTL)
	userRepo := repository.NewUserRepository(pool)
	authUsecase := usecase.NewAuthUsecase(userRepo, jwtIssuer)

	srv := &http.Server{
		Addr: ":" + cfg.HTTPPort,
		Handler: httptransport.NewRouter(httptransport.Deps{
			AuthUsecase: authUsecase,
			JWTIssuer:   jwtIssuer,
		}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

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
}
