// Package api provides the loan simulation HTTP handler.
package api

import (
	"encoding/json"
	"fmt"
	"loan-simulator/financial/iofconfig"
	"loan-simulator/financial/pmtconfig"
	"loan-simulator/financial/rateconfig"
	"math"
	"net/http"
)

// SimulationRequest holds the input parameters for a loan simulation.
type SimulationRequest struct {
	// LoanValue is the requested loan amount in BRL (e.g., 10000.00)
	LoanValue float64 `json:"loanValue"`
	// MonthlyRatePct is the monthly interest rate as a percentage (e.g., 2.5 for 2.5%/month)
	MonthlyRatePct float64 `json:"monthlyRatePct"`
	// Installments is the number of monthly installments (e.g., 12, 24, 36)
	Installments int `json:"installments"`
	// GracePeriodDays is the number of days until the first installment (default: 30)
	GracePeriodDays int `json:"gracePeriodDays"`
}

// SimulationResult holds the output of a loan simulation.
type SimulationResult struct {
	// LoanValue is the requested loan amount
	LoanValue float64 `json:"loanValue"`
	// FundedValue is the total financed amount (loan + IOF funded)
	FundedValue float64 `json:"fundedValue"`
	// FundedValueWithGrace is the funded value with grace period interest applied
	FundedValueWithGrace float64 `json:"fundedValueWithGrace"`
	// InstallmentValue is the fixed monthly payment
	InstallmentValue float64 `json:"installmentValue"`
	// TotalAmount is the total amount paid over all installments
	TotalAmount float64 `json:"totalAmount"`
	// TotalInterest is the total interest paid
	TotalInterest float64 `json:"totalInterest"`
	// MonthlyRatePct is the monthly interest rate (%)
	MonthlyRatePct float64 `json:"monthlyRatePct"`
	// YearlyRatePct is the equivalent yearly interest rate (%)
	YearlyRatePct float64 `json:"yearlyRatePct"`
	// MonthlyCetPct is the monthly CET - Custo Efetivo Total (%)
	MonthlyCetPct float64 `json:"monthlyCetPct"`
	// YearlyCetPct is the yearly CET - Custo Efetivo Total (%)
	YearlyCetPct float64 `json:"yearlyCetPct"`
	// Iof holds the IOF tax breakdown
	Iof IofSummary `json:"iof"`
}

// IofSummary holds the IOF tax breakdown.
type IofSummary struct {
	// TermIof is the IOF accrued by term (daily rate × days for each installment's amortization)
	TermIof float64 `json:"termIof"`
	// ComplementaryIof is the flat-rate IOF component (0.38%)
	ComplementaryIof float64 `json:"complementaryIof"`
	// InCashIof is the total IOF if paid upfront (à vista)
	InCashIof float64 `json:"inCashIof"`
	// FundedIof is the IOF portion embedded in the funded value
	FundedIof float64 `json:"fundedIof"`
}

// SimulateHandler handles POST /simulate requests.
func SimulateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SimulationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	result, err := Simulate(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("simulation error: %v", err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// Simulate performs the loan simulation calculation following the Zipdin production engine logic:
//
//  1. Calculate the PMT factor (installment per unit of funded value)
//  2. Calculate IOF "à vista" (in cash) on the loanValue using the Price table
//  3. Derive the funded value: loanValue + funded taxes (IOF + insurance financed)
//  4. Calculate the final installment value: pmtFactor * fundedValue
//  5. Calculate CET using XIRR on the actual cash flows
func Simulate(req SimulationRequest) (*SimulationResult, error) {
	if req.LoanValue <= 0 {
		return nil, fmt.Errorf("loanValue must be greater than zero")
	}
	if req.MonthlyRatePct <= 0 {
		return nil, fmt.Errorf("monthlyRatePct must be greater than zero")
	}
	if req.Installments <= 0 {
		return nil, fmt.Errorf("installments must be greater than zero")
	}
	if req.GracePeriodDays <= 0 {
		req.GracePeriodDays = 30
	}

	rate := rateconfig.NewMonthlyRate(req.MonthlyRatePct)
	monthlyRate := rate.ToMonthly() // decimal, e.g., 0.025
	yearlyRate := rate.ToYearly()   // percentage, e.g., 34.49

	// Grace period beyond the standard 30 days
	gracePeriod := req.GracePeriodDays - 30

	// Step 1: PMT factor — installment value per unit of funded value (with grace)
	pmtFactor := pmtconfig.NewPmtConfig(
		monthlyRate,
		req.Installments,
		1*math.Pow(1+monthlyRate, float64(gracePeriod)/30.0), // PV=1 adjusted for grace
	).CalcPmt()

	// Step 2: Calculate IOF "à vista" on loanValue (using Price table amortization)
	loanValueWithFees := req.LoanValue // no TC or insurance in simplified version
	inCashIofCalc := iofconfig.NewIofConfig(
		loanValueWithFees,
		req.Installments,
		req.GracePeriodDays,
		monthlyRate,
	)
	inCashIof := inCashIofCalc.Calculate()

	// Step 3: Calculate funded value
	// inCashTaxes = IOF à vista (what would be paid if not financed)
	inCashTaxes := inCashIof.TotalIof
	// fundedTaxes = taxes that get financed (proportional increase)
	fundedTaxes := (loanValueWithFees * inCashTaxes) / (loanValueWithFees - inCashTaxes)
	fundedValue := loanValueWithFees + fundedTaxes
	fundedIof := fundedValue - loanValueWithFees // the IOF portion in funded value

	// Step 4: Calculate installment value from funded value
	installmentValue := round2(pmtFactor * fundedValue)

	// Step 5: Apply grace period to get FundedValueWithGrace
	fundedValueWithGrace := round2(loanValueWithFees * math.Pow(1+monthlyRate, float64(gracePeriod)/30.0))

	totalAmount := round2(installmentValue * float64(req.Installments))
	totalInterest := round2(totalAmount - req.LoanValue)

	// Step 6: Calculate CET using XIRR approach (Newton-Raphson on cash flows)
	monthlyCet := calcCET(req.LoanValue, installmentValue, req.Installments, req.GracePeriodDays)
	yearlyCet := round2((math.Pow(1+monthlyCet, 12) - 1) * 100)
	monthlyCetPct := round2(monthlyCet * 100)

	return &SimulationResult{
		LoanValue:            round2(req.LoanValue),
		FundedValue:          round2(fundedValue),
		FundedValueWithGrace: fundedValueWithGrace,
		InstallmentValue:     installmentValue,
		TotalAmount:          totalAmount,
		TotalInterest:        totalInterest,
		MonthlyRatePct:       round2(req.MonthlyRatePct),
		YearlyRatePct:        round2(yearlyRate),
		MonthlyCetPct:        monthlyCetPct,
		YearlyCetPct:         yearlyCet,
		Iof: IofSummary{
			TermIof:          inCashIof.TermIof,
			ComplementaryIof: inCashIof.ComplementaryIof,
			InCashIof:        inCashIof.TotalIof,
			FundedIof:        round2(fundedIof),
		},
	}, nil
}

// calcCET approximates the monthly CET using Newton-Raphson method (IRR of cash flows).
// CET considers the loan value disbursed vs all installments paid.
func calcCET(loanValue, installment float64, n, graceDays int) float64 {
	// Initial guess
	rate := 0.025
	for i := 0; i < 200; i++ {
		npv := -loanValue
		dnpv := 0.0

		for t := 1; t <= n; t++ {
			// Each installment is paid at graceDays + (t-1)*30 days from issue
			// Convert to monthly periods: graceDays/30 + (t-1)
			period := float64(graceDays)/30.0 + float64(t-1)
			disc := math.Pow(1+rate, period)
			npv += installment / disc
			dnpv -= period * installment / (disc * (1 + rate))
		}

		if math.Abs(dnpv) < 1e-12 {
			break
		}
		newRate := rate - npv/dnpv
		if math.Abs(newRate-rate) < 1e-10 {
			break
		}
		rate = newRate
	}
	return rate
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
