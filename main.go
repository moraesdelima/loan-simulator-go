// Loan Simulator — a simple HTTP API for Brazilian consumer credit simulation.
//
// This is a simplified, public demo version of the loan calculation engine
// used in production at ZIPDIN (https://zipdin.com.br), a Brazilian fintech
// specializing in payroll credit (Consignado Privado) and Banking as a Service.
//
// The production system processes 220,000+ contracts/month and is deployed
// as an AWS Lambda function written in Go with Serverless architecture.
//
// Usage:
//
//	go run main.go
//	curl -X POST http://localhost:8080/simulate \
//	  -H "Content-Type: application/json" \
//	  -d '{"loanValue": 10000, "monthlyRatePct": 2.5, "installments": 12}'
package main

import (
	"fmt"
	"loan-simulator/api"
	"net/http"
)

func main() {
	http.HandleFunc("/simulate", api.SimulateHandler)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"status":"ok"}`)
	})

	port := "8080"
	fmt.Printf("Loan Simulator running on http://localhost:%s\n", port)
	fmt.Println("POST /simulate  — run a loan simulation")
	fmt.Println("GET  /health    — health check")

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		panic(err)
	}
}
