# Loan Simulator — Brazilian Consumer Credit Calculator

A simple HTTP API written in Go that simulates Brazilian consumer credit operations, calculating installments, IOF tax, and CET (Custo Efetivo Total).

## Background

This is a simplified, public demo of the loan calculation engine I built and deployed in production at [ZIPDIN](https://zipdin.com.br) — a Brazilian fintech specializing in payroll credit (*Consignado Privado*) and Banking as a Service.

The production version runs as an **AWS Lambda function** (Serverless architecture) and is part of a platform that processes **220,000+ contracts/month**. It handles multiple credit products: payroll credit, personal credit (CP), direct consumer credit (CDC), and FGTS credit.

## What it calculates

| Field | Description |
|---|---|
| `installmentValue` | Fixed monthly payment (PMT) |
| `fundedValue` | Total financed amount (loan + IOF) |
| `totalAmount` | Sum of all installments |
| `totalInterest` | Total interest paid |
| `monthlyRatePct` | Monthly interest rate (%) |
| `yearlyRatePct` | Equivalent yearly rate (%) |
| `monthlyCetPct` | Monthly CET — Custo Efetivo Total (%) |
| `yearlyCetPct` | Yearly CET (%) |
| `iof.principalIof` | Daily-accrued IOF component |
| `iof.complementaryIof` | Flat-rate IOF component (0.38%) |
| `iof.totalIof` | Total IOF tax |

### Key financial concepts

- **PMT**: Standard loan amortization formula with compound interest
- **IOF**: Brazilian federal tax on credit operations (Decreto 6.306/2007)
  - Principal IOF: 0.0082%/day on outstanding balance, capped at 365 days
  - Complementary IOF: 0.38% flat on the financed amount
- **CET**: Custo Efetivo Total — the true cost of credit including all fees and taxes, calculated as the IRR of the cash flows (required by Brazilian Central Bank regulation)
- **Grace period**: Days until the first installment (default: 30 days)

## Getting started

### Prerequisites

- Go 1.21+

### Run

```bash
go run main.go
```

### Test

```bash
go test ./...
```

### Example request

```bash
curl -X POST http://localhost:8080/simulate \
  -H "Content-Type: application/json" \
  -d '{
    "loanValue": 10000,
    "monthlyRatePct": 2.5,
    "installments": 12,
    "gracePeriodDays": 30
  }'
```

### Example response

```json
{
  "loanValue": 10000.00,
  "fundedValue": 10185.43,
  "installmentValue": 1005.17,
  "totalAmount": 12062.04,
  "totalInterest": 2062.04,
  "monthlyRatePct": 2.50,
  "yearlyRatePct": 34.49,
  "monthlyCetPct": 2.68,
  "yearlyCetPct": 37.54,
  "iof": {
    "principalIof": 109.05,
    "complementaryIof": 38.00,
    "totalIof": 147.05
  }
}
```

## Project structure

```
loan-simulator-go/
├── main.go                        # HTTP server entry point
├── api/
│   ├── simulator.go               # HTTP handler + simulation orchestration
│   └── simulator_test.go          # Unit tests
└── financial/
    ├── pmtconfig/
    │   └── pmt.go                 # PMT (installment) calculation
    ├── rateconfig/
    │   └── rate.go                # Interest rate conversion utilities
    └── iofconfig/
        └── iof.go                 # IOF tax calculation (Brazilian regulation)
```

## Production architecture (for reference)

The production version at ZIPDIN differs from this demo in the following ways:

- Deployed as **AWS Lambda** using AWS SAM (Serverless Application Model)
- Authenticated via **JWT** tokens
- Supports multiple calculation modes: by loan value, by installment value, by funded value, and reverse rate calculation
- Includes **XIRR** (Extended IRR) for irregular cash flows
- Handles **FGTS credit** withdrawals simulation
- Uses **DynamoDB** for IOF parameter configuration per product
- Integrated with **AWS API Gateway** for routing

## Author

**Luiz Lima** — Principal Engineer & Engineering Manager
[LinkedIn](https://linkedin.com/in/luiz-lima-1a133144)
