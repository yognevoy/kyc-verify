package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"kyc-verify/internal/domain"
)

var (
	ErrFullNameRequired = errors.New("full name is required")
	ErrInvalidBirthDate = errors.New("invalid birth date")
	ErrInvalidCountry   = errors.New("country must be a 2-letter ISO code")
)

const BirthDateLayout = "2006-01-02"

type ApplicantInput struct {
	FullName  string
	BirthDate string
	Country   string
}

type ApplicantUsecase struct {
	applicants domain.ApplicantRepository
}

func NewApplicantUsecase(applicants domain.ApplicantRepository) *ApplicantUsecase {
	return &ApplicantUsecase{applicants: applicants}
}

func (a *ApplicantUsecase) Create(ctx context.Context, userID uuid.UUID, in ApplicantInput) (*domain.Applicant, error) {
	birthDate, country, err := validateApplicantInput(in)
	if err != nil {
		return nil, err
	}

	applicant := &domain.Applicant{
		ID:        uuid.New(),
		UserID:    userID,
		FullName:  in.FullName,
		BirthDate: birthDate,
		Country:   country,
		RiskLevel: ScoreRisk(RiskInput{Country: country, BirthDate: birthDate}),
	}

	if err := a.applicants.Create(ctx, applicant); err != nil {
		return nil, err
	}
	return applicant, nil
}

func (a *ApplicantUsecase) GetByID(ctx context.Context, id uuid.UUID) (*domain.Applicant, error) {
	return a.applicants.GetByID(ctx, id)
}

func (a *ApplicantUsecase) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.Applicant, error) {
	return a.applicants.GetByUserID(ctx, userID)
}

func (a *ApplicantUsecase) Update(ctx context.Context, userID uuid.UUID, in ApplicantInput) (*domain.Applicant, error) {
	birthDate, country, err := validateApplicantInput(in)
	if err != nil {
		return nil, err
	}

	applicant, err := a.applicants.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	applicant.FullName = in.FullName
	applicant.BirthDate = birthDate
	applicant.Country = country
	applicant.RiskLevel = ScoreRisk(RiskInput{Country: country, BirthDate: birthDate})

	if err := a.applicants.Update(ctx, applicant); err != nil {
		return nil, err
	}
	return applicant, nil
}

func validateApplicantInput(in ApplicantInput) (time.Time, string, error) {
	if strings.TrimSpace(in.FullName) == "" {
		return time.Time{}, "", ErrFullNameRequired
	}

	birthDate, err := time.Parse(BirthDateLayout, in.BirthDate)
	if err != nil || birthDate.After(time.Now()) {
		return time.Time{}, "", ErrInvalidBirthDate
	}

	country := strings.ToUpper(strings.TrimSpace(in.Country))
	if len(country) != 2 {
		return time.Time{}, "", ErrInvalidCountry
	}

	return birthDate, country, nil
}
