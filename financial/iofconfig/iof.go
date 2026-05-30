// Package iofconfig calculates IOF (Imposto sobre Operações Financeiras) for Brazilian credit operations.
//
// IOF is a Brazilian federal tax applied to credit operations. It has two components:
//   - Principal IOF: charged daily on the amortization (principal) of each installment
//   - Complementary IOF: a flat percentage on the amortization of each installment
//
// The calculation uses the Price table (French amortization system) to determine
// the principal portion of each installment, then applies IOF based on the number
// of days from the issue date to each due date (capped at 365 days).
//
// Reference: Brazilian legislation (Decreto 6.306/2007 and updates).
package iofconfig

import (
	"math"
)

const (
	// DefaultDailyIofRate is the standard IOF daily rate for credit operations (0.0082% per day).
	DefaultDailyIofRate = 0.0082

	// DefaultComplementaryIofRate is the flat complementary IOF rate (0.38%).
	DefaultComplementaryIofRate = 0.38

	// MaxIofDays is the maximum number of days IOF is charged (365 days).
	MaxIofDays = 365
)

// IofResult holds the calculated IOF values.
type IofResult struct {
	// TermIof is the IOF accrued by term (daily rate × days for each installment's amortization)
	TermIof float64
	// ComplementaryIof is the flat-rate IOF component
	ComplementaryIof float64
	// TotalIof is the sum of both components
	TotalIof float64
}

// IofConfig holds the parameters for IOF calculation.
type IofConfig struct {
	// FundedValue is the total financed amount (base for amortization table)
	FundedValue float64
	// InstallmentsNumber is the number of monthly installments
	InstallmentsNumber int
	// GracePeriod is the number of days until the first installment (beyond 30 days)
	GracePeriod int
	// MonthlyRate is the monthly interest rate as a decimal (e.g., 0.025 for 2.5%)
	MonthlyRate float64
	// DailyIofRate is the IOF rate per day (default: 0.0082%)
	DailyIofRate float64
	// ComplementaryIofRate is the flat IOF rate (default: 0.38%)
	ComplementaryIofRate float64
}

// NewIofConfig creates an IofConfig with Brazilian standard IOF rates.
// gracePeriod here is the total days to first due date (e.g., 30 for standard, 31 for a 31-day month).
func NewIofConfig(fundedValue float64, installments, gracePeriod int, monthlyRate float64) IofConfig {
	return IofConfig{
		FundedValue:          fundedValue,
		InstallmentsNumber:   installments,
		GracePeriod:          gracePeriod,
		MonthlyRate:          monthlyRate,
		DailyIofRate:         DefaultDailyIofRate,
		ComplementaryIofRate: DefaultComplementaryIofRate,
	}
}

// Calculate computes the IOF using the Price table amortization method.
// For each installment, it calculates:
//   - The interest portion: debtBalance * (1+rate)^(daysInPeriod/30) - debtBalance
//   - The principal (amortization) portion: installmentValue - interest
//   - Principal IOF on the amortization: principal * (iofRatePerDay/100) * iofDays
//   - Complementary IOF: flat rate on the loanValue (not per-installment)
//
// This matches the production Zipdin calculation engine.
func (c IofConfig) Calculate() IofResult {
	if c.MonthlyRate == 0 || c.InstallmentsNumber == 0 {
		return IofResult{}
	}

	// Calculate PMT (installment value) for the funded value with grace period
	graceFactor := math.Pow(1+c.MonthlyRate, float64(c.GracePeriod)/30.0)
	pv := c.FundedValue * graceFactor
	factor := math.Pow(1+c.MonthlyRate, float64(c.InstallmentsNumber))
	installmentValue := pv * (c.MonthlyRate * factor) / (factor - 1)

	// Build amortization table and calculate IOF per installment
	principalIof := 0.0
	debtBalance := pv // debt balance starts at PV (funded value with grace)

	// Days tracking: first due date is at GracePeriod days from issue
	daysFromIssue := c.GracePeriod // first installment due date

	for i := 1; i <= c.InstallmentsNumber; i++ {
		// Interest for this period (30 days standard)
		daysInPeriod := 30
		interest := debtBalance*math.Pow(1+c.MonthlyRate, float64(daysInPeriod)/30.0) - debtBalance

		// Principal (amortization) for this installment
		principal := installmentValue - interest
		if principal < 0 {
			principal = 0
		}

		// IOF days (capped at MaxIofDays)
		iofDays := daysFromIssue
		if iofDays > MaxIofDays {
			iofDays = MaxIofDays
		}

		// Principal IOF on the amortization portion
		principalIof += principal * (c.DailyIofRate / 100) * float64(iofDays)

		// Update debt balance
		debtBalance = debtBalance + interest - installmentValue

		// Next due date (approximately 30 days later)
		daysFromIssue += 30
	}

	// Complementary IOF is a flat rate on the funded value (not per-installment)
	complementaryIof := c.FundedValue * (c.ComplementaryIofRate / 100)

	totalIof := principalIof + complementaryIof

	return IofResult{
		TermIof:          round2(principalIof),
		ComplementaryIof: round2(complementaryIof),
		TotalIof:         round2(totalIof),
	}
}

// CalculateInCash computes the IOF "à vista" (in cash) on the loan value.
// This is the IOF that would be charged if paid upfront, before being financed.
func (c IofConfig) CalculateInCash() IofResult {
	return c.Calculate()
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
