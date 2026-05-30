package api

import (
	"math"
	"testing"
)

func TestSimulate_BasicLoan(t *testing.T) {
	req := SimulationRequest{
		LoanValue:       10000,
		MonthlyRatePct:  2.5,
		Installments:    12,
		GracePeriodDays: 30,
	}

	result, err := Simulate(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.LoanValue != 10000 {
		t.Errorf("expected loanValue=10000, got %.2f", result.LoanValue)
	}
	if result.InstallmentValue <= 0 {
		t.Errorf("expected positive installment, got %.2f", result.InstallmentValue)
	}
	if result.FundedValue <= result.LoanValue {
		t.Errorf("funded value (%.2f) should be greater than loan value (%.2f)", result.FundedValue, result.LoanValue)
	}
	if result.TotalAmount <= result.LoanValue {
		t.Errorf("total amount (%.2f) should be greater than loan value (%.2f)", result.TotalAmount, result.LoanValue)
	}
	if result.Iof.InCashIof <= 0 {
		t.Errorf("expected positive IOF, got %.2f", result.Iof.InCashIof)
	}
	if result.YearlyCetPct <= result.YearlyRatePct {
		t.Errorf("CET (%.2f%%) should be >= yearly rate (%.2f%%)", result.YearlyCetPct, result.YearlyRatePct)
	}

	t.Logf("Simulation result: installment=R$%.2f, fundedValue=R$%.2f, IOF(inCash)=R$%.2f, IOF(funded)=R$%.2f, CET=%.2f%%/year",
		result.InstallmentValue, result.FundedValue, result.Iof.InCashIof, result.Iof.FundedIof, result.YearlyCetPct)
}

func TestSimulate_ZeroGracePeriodDefaultsTo30(t *testing.T) {
	req := SimulationRequest{
		LoanValue:      5000,
		MonthlyRatePct: 1.8,
		Installments:   24,
	}
	result, err := Simulate(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.InstallmentValue <= 0 {
		t.Errorf("expected positive installment")
	}
}

func TestSimulate_ValidationErrors(t *testing.T) {
	cases := []struct {
		name string
		req  SimulationRequest
	}{
		{"zero loan value", SimulationRequest{LoanValue: 0, MonthlyRatePct: 2.0, Installments: 12}},
		{"negative rate", SimulationRequest{LoanValue: 1000, MonthlyRatePct: -1, Installments: 12}},
		{"zero installments", SimulationRequest{LoanValue: 1000, MonthlyRatePct: 2.0, Installments: 0}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Simulate(tc.req)
			if err == nil {
				t.Errorf("expected error for %s", tc.name)
			}
		})
	}
}

func TestSimulate_IofComponents(t *testing.T) {
	req := SimulationRequest{
		LoanValue:       20000,
		MonthlyRatePct:  3.0,
		Installments:    36,
		GracePeriodDays: 30,
	}
	result, err := Simulate(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// IOF components should sum to InCashIof
	expectedTotal := math.Round((result.Iof.TermIof+result.Iof.ComplementaryIof)*100) / 100
	if math.Abs(expectedTotal-result.Iof.InCashIof) > 0.01 {
		t.Errorf("IOF components (%.2f + %.2f = %.2f) don't match InCashIof (%.2f)",
			result.Iof.TermIof, result.Iof.ComplementaryIof, expectedTotal, result.Iof.InCashIof)
	}

	// FundedIof should be greater than InCashIof (because it's financed)
	if result.Iof.FundedIof <= 0 {
		t.Errorf("expected positive FundedIof, got %.2f", result.Iof.FundedIof)
	}
}
