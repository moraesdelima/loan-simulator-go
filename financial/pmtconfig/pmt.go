// Package pmtconfig provides PMT (Payment) calculation for loan installments.
// PMT calculates the fixed periodic payment required to fully amortize a loan
// at a given interest rate over a specified number of periods.
package pmtconfig

import "math"

// PmtConfig holds the parameters for a PMT calculation.
type PmtConfig struct {
	// Rate is the periodic interest rate (e.g., 0.02 for 2% per month)
	Rate float64
	// NPer is the total number of payment periods
	NPer int
	// PV is the present value (loan amount)
	PV float64
}

// NewPmtConfig creates a new PmtConfig.
func NewPmtConfig(rate float64, nPer int, pv float64) *PmtConfig {
	return &PmtConfig{Rate: rate, NPer: nPer, PV: pv}
}

// CalcPmt calculates the fixed installment value using the standard PMT formula:
//
//	PMT = PV * (rate * (1+rate)^n) / ((1+rate)^n - 1)
func (p PmtConfig) CalcPmt() float64 {
	if p.Rate == 0 {
		return p.PV / float64(p.NPer)
	}
	factor := math.Pow(1+p.Rate, float64(p.NPer))
	return p.PV * (p.Rate * factor) / (factor - 1)
}
