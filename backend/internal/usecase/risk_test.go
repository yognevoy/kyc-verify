package usecase

import (
	"testing"
	"time"

	"kyc-verify/internal/domain"
)

func TestScoreRisk(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name  string
		input RiskInput
		want  domain.RiskLevel
	}{
		{name: "adult from an ordinary country", input: RiskInput{Country: "US", BirthDate: now.AddDate(-30, 0, 0)}, want: domain.RiskLow},
		{name: "minor from an ordinary country", input: RiskInput{Country: "US", BirthDate: now.AddDate(-10, 0, 0)}, want: domain.RiskHigh},
		{name: "adult from a sanctioned country", input: RiskInput{Country: "KP", BirthDate: now.AddDate(-30, 0, 0)}, want: domain.RiskHigh},
		{name: "another sanctioned country", input: RiskInput{Country: "IR", BirthDate: now.AddDate(-30, 0, 0)}, want: domain.RiskHigh},
		{name: "minor from a sanctioned country", input: RiskInput{Country: "SY", BirthDate: now.AddDate(-10, 0, 0)}, want: domain.RiskHigh},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ScoreRisk(tt.input); got != tt.want {
				t.Fatalf("ScoreRisk() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestAgeInYears(t *testing.T) {
	date := func(y int, m time.Month, d int) time.Time {
		return time.Date(y, m, d, 12, 0, 0, 0, time.UTC)
	}

	tests := []struct {
		name  string
		birth time.Time
		at    time.Time
		want  int
	}{
		{name: "day before the birthday", birth: date(2000, time.June, 15), at: date(2018, time.June, 14), want: 17},
		{name: "on the birthday", birth: date(2000, time.June, 15), at: date(2018, time.June, 15), want: 18},
		{name: "day after the birthday", birth: date(2000, time.June, 15), at: date(2018, time.June, 16), want: 18},
		{name: "birthday in a later month", birth: date(2000, time.December, 31), at: date(2018, time.January, 1), want: 17},
		{name: "born after Feb 29 in a leap year, birthday in a common year", birth: date(2000, time.March, 1), at: date(2018, time.March, 1), want: 18},
		{name: "born after Feb 29 in a common year, day before birthday in a leap year", birth: date(2002, time.March, 1), at: date(2020, time.February, 29), want: 17},
		{name: "born on Feb 29, before the birthday in a common year", birth: date(2004, time.February, 29), at: date(2022, time.February, 28), want: 17},
		{name: "born on Feb 29, on Mar 1 of a common year", birth: date(2004, time.February, 29), at: date(2022, time.March, 1), want: 18},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ageInYears(tt.birth, tt.at); got != tt.want {
				t.Fatalf("ageInYears(%s, %s) = %d, want %d", tt.birth.Format(time.DateOnly), tt.at.Format(time.DateOnly), got, tt.want)
			}
		})
	}
}
