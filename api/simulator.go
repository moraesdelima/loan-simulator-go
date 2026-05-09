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
	LoanValue       float64 `json:"loanValue"`
	MonthlyRatePct  float64 `json:"monthlyRatePct"`
	Installments    int     `json:"installments"`
	GracePeriodDays int     `json:"gracePeriodDays"`
}

// SimulationResult holds the output of a loan simulation.
type SimulationResult struct {
	LoanValue        float64    `json:"loanValue"`
	FundedValue      float64    `json:"fundedValue"`
	InstallmentValue float64    `json:"installmentValue"`
	TotalAmount      float64    `json:"totalAmount"`
	TotalInterest    float64    `json:"totalInterest"`
	MonthlyRatePct   float64    `json:"monthlyRatePct"`
	YearlyRatePct    float64    `json:"yearlyRatePct"`
	MonthlyCetPct    float64    `json:"monthlyCetPct"`
	YearlyCetPct     float64    `json:"yearlyCetPct"`
	Iof              IofSummary `json:"iof"`
}

// IofSummary holds the IOF tax breakdown.
type IofSummary struct {
	PrincipalIof     float64 `json:"principalIof"`
	ComplementaryIof float64 `json:"complementaryIof"`
	TotalIof         float64 `json:"totalIof"`
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

// Simulate performs the loan simulation calculation.
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
	monthlyRate := rate.ToMonthly()
	yearlyRate := rate.ToYearly()

	iofCalc := iofconfig.NewIofConfig(req.LoanValue, req.Installments, req.GracePeriodDays)
	iofResult := iofCalc.Calculate()

	fundedValue := req.LoanValue + iofResult.TotalIof

	graceFactor := math.Pow(1+monthlyRate, float64(req.GracePeriodDays)/30.0)
	pmt := pmtconfig.NewPmtConfig(monthlyRate, req.Installments, fundedValue*graceFactor).CalcPmt()

	installmentValue := round2(pmt)
	totalAmount := round2(installmentValue * float64(req.Installments))
	totalInterest := round2(totalAmount - req.LoanValue)

	monthlyCet := calcCET(req.LoanValue, installmentValue, req.Installments, req.GracePeriodDays)
	yearlyCet := round2((math.Pow(1+monthlyCet, 12) - 1) * 100)
	monthlyCetPct := round2(monthlyCet * 100)

	return &SimulationResult{
		LoanValue:        round2(req.LoanValue),
		FundedValue:      round2(fundedValue),
		InstallmentValue: installmentValue,
		TotalAmount:      totalAmount,
		TotalInterest:    totalInterest,
		MonthlyRatePct:   round2(req.MonthlyRatePct),
		YearlyRatePct:    round2(yearlyRate),
		MonthlyCetPct:    monthlyCetPct,
		YearlyCetPct:     yearlyCet,
		Iof: IofSummary{
			PrincipalIof:     iofResult.PrincipalIof,
			ComplementaryIof: iofResult.ComplementaryIof,
			TotalIof:         iofResult.TotalIof,
		},
	}, nil
}

func calcCET(loanValue, installment float64, n, graceDays int) float64 {
	rate := 0.02
	for i := 0; i < 100; i++ {
		npv := -loanValue
		for t := 1; t <= n; t++ {
			npv += installment / math.Pow(1+rate, float64(t)+float64(graceDays)/30.0-1)
		}
		dnpv := 0.0
		for t := 1; t <= n; t++ {
			exp := float64(t) + float64(graceDays)/30.0 - 1
			dnpv -= exp * installment / math.Pow(1+rate, exp+1)
		}
		if math.Abs(dnpv) < 1e-12 {
			break
		}
		newRate := rate - npv/dnpv
		if math.Abs(newRate-rate) < 1e-8 {
			break
		}
		rate = newRate
	}
	return rate
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
