// Package iofconfig calculates IOF (Imposto sobre Operacoes Financeiras) for Brazilian credit operations.
//
// IOF is a Brazilian federal tax applied to credit operations. It has two components:
//   - Principal IOF: charged daily on the outstanding balance (capped at 365 days)
//   - Complementary IOF: a flat percentage on the total financed amount
//
// Reference: Brazilian legislation (Decreto 6.306/2007 and updates).
package iofconfig

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
	// PrincipalIof is the daily-accrued IOF component
	PrincipalIof float64
	// ComplementaryIof is the flat-rate IOF component
	ComplementaryIof float64
	// TotalIof is the sum of both components
	TotalIof float64
}

// IofConfig holds the parameters for IOF calculation.
type IofConfig struct {
	// FundedValue is the total financed amount (loan + fees)
	FundedValue float64
	// InstallmentsNumber is the number of monthly installments
	InstallmentsNumber int
	// GracePeriod is the number of days until the first installment
	GracePeriod int
	// DailyIofRate is the IOF rate per day (default: 0.0082%)
	DailyIofRate float64
	// ComplementaryIofRate is the flat IOF rate (default: 0.38%)
	ComplementaryIofRate float64
}

// NewIofConfig creates an IofConfig with Brazilian standard IOF rates.
func NewIofConfig(fundedValue float64, installments, gracePeriod int) IofConfig {
	return IofConfig{
		FundedValue:          fundedValue,
		InstallmentsNumber:   installments,
		GracePeriod:          gracePeriod,
		DailyIofRate:         DefaultDailyIofRate,
		ComplementaryIofRate: DefaultComplementaryIofRate,
	}
}

// Calculate computes the IOF for a regular installment loan.
func (c IofConfig) Calculate() IofResult {
	principalIof := 0.0
	installmentPrincipal := c.FundedValue / float64(c.InstallmentsNumber)

	for i := 1; i <= c.InstallmentsNumber; i++ {
		daysToInstallment := c.GracePeriod + (i-1)*30
		if daysToInstallment > MaxIofDays {
			daysToInstallment = MaxIofDays
		}
		principalIof += installmentPrincipal * (c.DailyIofRate / 100) * float64(daysToInstallment)
	}

	complementaryIof := c.FundedValue * (c.ComplementaryIofRate / 100)
	totalIof := principalIof + complementaryIof

	return IofResult{
		PrincipalIof:     round2(principalIof),
		ComplementaryIof: round2(complementaryIof),
		TotalIof:         round2(totalIof),
	}
}

func round2(v float64) float64 {
	factor := 100.0
	return float64(int(v*factor+0.5)) / factor
}
