package usecase

import (
	"time"

	"kyc-verify/internal/domain"
)

var sanctionedCountries = map[string]bool{
	"KP": true,
	"IR": true,
	"SY": true,
	"CU": true,
	"AF": true,
}

type RiskInput struct {
	Country   string
	BirthDate time.Time
}

func ScoreRisk(in RiskInput) domain.RiskLevel {
	if sanctionedCountries[in.Country] {
		return domain.RiskHigh
	}
	if ageInYears(in.BirthDate, time.Now()) < 18 {
		return domain.RiskHigh
	}
	return domain.RiskLow
}

func ageInYears(birthDate, at time.Time) int {
	age := at.Year() - birthDate.Year()
	if at.Month() < birthDate.Month() || (at.Month() == birthDate.Month() && at.Day() < birthDate.Day()) {
		age--
	}
	return age
}
