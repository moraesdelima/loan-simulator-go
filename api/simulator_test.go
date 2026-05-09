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
		t.Errorf("funded value should be greater than loan value")
	}
	if result.Iof.TotalIof <= 0 {
		t.Errorf("expected positive IOF")
	}
	if result.YearlyCetPct <= result.YearlyRatePct {
		t.Errorf("CET should be >= yearly rate")
	}
	t.Logf("installment=R$%.2f totalAmount=R$%.2f IOF=R$%.2f CET=%.2f%%/year",
		result.InstallmentValue, result.TotalAmount, result.Iof.TotalIof, result.YearlyCetPct)
}

func TestSimulate_DefaultGracePeriod(t *testing.T) {
	req := SimulationRequest{LoanValue: 5000, MonthlyRatePct: 1.8, Installments: 24}
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
		{"zero loan", SimulationRequest{LoanValue: 0, MonthlyRatePct: 2.0, Installments: 12}},
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
	req := SimulationRequest{LoanValue: 20000, MonthlyRatePct: 3.0, Installments: 36, GracePeriodDays: 30}
	result, err := Simulate(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := math.Round((result.Iof.PrincipalIof+result.Iof.ComplementaryIof)*100) / 100
	if math.Abs(expected-result.Iof.TotalIof) > 0.01 {
		t.Errorf("IOF components don't match total: %.2f + %.2f != %.2f",
			result.Iof.PrincipalIof, result.Iof.ComplementaryIof, result.Iof.TotalIof)
	}
}
