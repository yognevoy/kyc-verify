package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"kyc-verify/internal/domain"
)

var (
	ErrInvalidEmail        = errors.New("invalid email")
	ErrPasswordTooShort    = errors.New("password must be at least 8 characters")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
)

const minPasswordLength = 8

type TokenIssuer interface {
	Issue(userID uuid.UUID, role string) (string, error)
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

type AuthUsecase struct {
	users         domain.UserRepository
	refreshTokens domain.RefreshTokenRepository
	tokens        TokenIssuer
	refreshTTL    time.Duration
}

func NewAuthUsecase(users domain.UserRepository, refreshTokens domain.RefreshTokenRepository, tokens TokenIssuer, refreshTTL time.Duration) *AuthUsecase {
	return &AuthUsecase{users: users, refreshTokens: refreshTokens, tokens: tokens, refreshTTL: refreshTTL}
}

func (a *AuthUsecase) Register(ctx context.Context, email, password string) (TokenPair, error) {
	if _, err := mail.ParseAddress(email); err != nil {
		return TokenPair{}, ErrInvalidEmail
	}
	if len(password) < minPasswordLength {
		return TokenPair{}, ErrPasswordTooShort
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return TokenPair{}, fmt.Errorf("hash password: %w", err)
	}

	user := &domain.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: string(hash),
		Role:         domain.RoleApplicant,
	}

	if err := a.users.Create(ctx, user); err != nil {
		return TokenPair{}, err
	}

	return a.issueTokenPair(ctx, user)
}

func (a *AuthUsecase) Login(ctx context.Context, email, password string) (TokenPair, error) {
	user, err := a.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return TokenPair{}, ErrInvalidCredentials
		}
		return TokenPair{}, fmt.Errorf("get user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return TokenPair{}, ErrInvalidCredentials
	}

	return a.issueTokenPair(ctx, user)
}

func (a *AuthUsecase) Refresh(ctx context.Context, rawToken string) (TokenPair, error) {
	stored, err := a.refreshTokens.GetByHash(ctx, hashRefreshToken(rawToken))
	if err != nil {
		if errors.Is(err, domain.ErrRefreshTokenNotFound) {
			return TokenPair{}, ErrInvalidRefreshToken
		}
		return TokenPair{}, fmt.Errorf("get refresh token: %w", err)
	}

	if stored.RevokedAt != nil {
		if err := a.refreshTokens.RevokeAllForUser(ctx, stored.UserID); err != nil {
			return TokenPair{}, fmt.Errorf("revoke sessions: %w", err)
		}
		return TokenPair{}, ErrInvalidRefreshToken
	}

	if time.Now().After(stored.ExpiresAt) {
		return TokenPair{}, ErrInvalidRefreshToken
	}

	if err := a.refreshTokens.Revoke(ctx, stored.ID); err != nil {
		return TokenPair{}, fmt.Errorf("revoke refresh token: %w", err)
	}

	user, err := a.users.GetByID(ctx, stored.UserID)
	if err != nil {
		return TokenPair{}, fmt.Errorf("get user: %w", err)
	}

	return a.issueTokenPair(ctx, user)
}

func (a *AuthUsecase) Logout(ctx context.Context, rawToken string) error {
	stored, err := a.refreshTokens.GetByHash(ctx, hashRefreshToken(rawToken))
	if err != nil {
		if errors.Is(err, domain.ErrRefreshTokenNotFound) {
			return nil
		}
		return fmt.Errorf("get refresh token: %w", err)
	}
	return a.refreshTokens.Revoke(ctx, stored.ID)
}

func (a *AuthUsecase) issueTokenPair(ctx context.Context, user *domain.User) (TokenPair, error) {
	access, err := a.tokens.Issue(user.ID, string(user.Role))
	if err != nil {
		return TokenPair{}, fmt.Errorf("issue access token: %w", err)
	}

	raw, hash, err := generateRefreshToken()
	if err != nil {
		return TokenPair{}, fmt.Errorf("generate refresh token: %w", err)
	}

	rt := &domain.RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: hash,
		ExpiresAt: time.Now().Add(a.refreshTTL),
	}
	if err := a.refreshTokens.Create(ctx, rt); err != nil {
		return TokenPair{}, fmt.Errorf("store refresh token: %w", err)
	}

	return TokenPair{AccessToken: access, RefreshToken: raw}, nil
}

func generateRefreshToken() (raw, hash string, err error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", fmt.Errorf("generate random token: %w", err)
	}
	raw = base64.RawURLEncoding.EncodeToString(buf)
	return raw, hashRefreshToken(raw), nil
}

func hashRefreshToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
