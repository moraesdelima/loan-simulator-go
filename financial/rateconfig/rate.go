// Package rateconfig provides interest rate conversion utilities.
// Supports compound interest conversion between daily, monthly, and yearly periods.
package rateconfig

import "math"

const (
	Daily     = 1.0
	Monthly   = 30.0
	Yearly    = 360.0
	Yearly365 = 365.0
)

// RateConfig holds an interest rate and its reference period.
type RateConfig struct {
	rate   float64
	period float64
}

// NewMonthlyRate creates a RateConfig from a monthly rate percentage (e.g., 2.5 for 2.5%/month).
func NewMonthlyRate(monthlyRatePct float64) RateConfig {
	return RateConfig{rate: monthlyRatePct, period: Monthly}
}

// GetRate returns the raw rate value.
func (r RateConfig) GetRate() float64 {
	return r.rate
}

// ToMonthly converts the rate to a monthly decimal factor (e.g., 0.025 for 2.5%/month).
func (r RateConfig) ToMonthly() float64 {
	return math.Pow(1+r.rate/100, Monthly/r.period) - 1
}

// ToYearly converts the rate to a yearly percentage using compound interest.
func (r RateConfig) ToYearly() float64 {
	monthly := r.ToMonthly()
	return (math.Pow(1+monthly, 12) - 1) * 100
}

// ApplyTo applies compound interest to an amount over a given number of days.
func (r RateConfig) ApplyTo(amount float64, days int) float64 {
	if r.rate == 0 {
		return amount
	}
	return amount * math.Pow(1+r.rate/100, float64(days)/r.period)
}
