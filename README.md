# Loan Simulator — Brazilian Consumer Credit Calculator

A lightweight HTTP API in Go that simulates Brazilian consumer credit operations — calculating installments, IOF tax, and CET (Total Effective Cost).

## Background

This is a simplified, public demo of a loan calculation engine inspired by a production system I built at a Brazilian fintech specializing in payroll credit (*Consignado Privado*) and Banking as a Service.

The production version powers a platform processing **220,000+ contracts/month** as an AWS Lambda function, handling payroll credit, personal credit (CP), direct consumer credit (CDC), and FGTS credit.

---

## How It Works

### 1. Amortization — Price Table

The simulator uses the **French Amortization System** (Tabela Price), the standard for consumer credit in Brazil. Each installment is fixed and calculated with the PMT formula:

```
PMT = PV × [i × (1+i)^n] / [(1+i)^n - 1]
```

- `PV` — present value (funded amount, adjusted for grace period)
- `i` — monthly interest rate (decimal)
- `n` — number of installments

### 2. Interest Rate Conversion

Rates are converted between periods using **compound interest**, as required by BACEN (Resolução CMN nº 4.881/2020):

```
yearly_rate = (1 + monthly_rate)^12 - 1
```

### 3. Grace Period

The grace period capitalizes the debt before amortization begins:

```
PV_adjusted = Funded_Value × (1 + monthly_rate)^(grace_days / 30)
```

This covers the interval between contract signing and the first due date.

### 4. IOF Calculation

IOF is a federal tax on credit operations (Decreto nº 6.306/2007) with two components:

| Component | Rate | Base |
|---|---|---|
| Term IOF | 0.0082%/day | Amortization of each installment × elapsed days |
| Complementary IOF | 0.38% flat | Total funded amount |

The simulator builds a full amortization table and, for each installment:

1. Separates interest from amortization (principal)
2. Counts days from issue date to due date (capped at 365)
3. Applies the daily IOF rate on the amortization portion

IOF is then **financed** (embedded in the loan amount):

```
funded_taxes = (loan_value × in_cash_iof) / (loan_value - in_cash_iof)
funded_value = loan_value + funded_taxes
```

### 5. CET — Total Effective Cost

The CET represents the true annual cost of credit. It is mandatory per Resolução CMN nº 4.881/2020 and calculated as the **IRR** of the cash flows:

```
0 = -Loan_Value + Σ [Installment / (1 + CET)^t]
```

Solved via **Newton-Raphson** iteration, then annualized:

```
CET_yearly = (1 + CET_monthly)^12 - 1
```

The CET is based on the **disbursed amount** (what the client actually receives), not the funded value.

---

## API Reference

### Endpoints

| Method | Path | Description |
|---|---|---|
| `POST` | `/simulate` | Run a loan simulation |
| `GET` | `/health` | Health check |

### Request

```bash
curl -X POST http://localhost:8080/simulate \
  -H "Content-Type: application/json" \
  -d '{
    "loanValue": 10000,
    "monthlyRatePct": 2.5,
    "installments": 12,
    "gracePeriodDays": 31
  }'
```

| Parameter | Type | Description |
|---|---|---|
| `loanValue` | float | Loan amount requested (R$) |
| `monthlyRatePct` | float | Monthly interest rate (%) |
| `installments` | int | Number of monthly payments |
| `gracePeriodDays` | int | Days until first due date (default: 30) |

### Response

```json
{
  "loanValue": 10000,
  "fundedValue": 10214.81,
  "fundedValueWithGrace": 10008.23,
  "installmentValue": 996.63,
  "totalAmount": 11959.56,
  "totalInterest": 1959.56,
  "monthlyRatePct": 2.5,
  "yearlyRatePct": 34.49,
  "monthlyCetPct": 2.85,
  "yearlyCetPct": 40.12,
  "iof": {
    "termIof": 172.29,
    "complementaryIof": 38,
    "inCashIof": 210.29,
    "fundedIof": 214.81
  }
}
```

| Field | Description |
|---|---|
| `loanValue` | Loan amount disbursed to the client |
| `fundedValue` | Total financed (loan + financed IOF) |
| `fundedValueWithGrace` | Debt balance at first due date (after grace capitalization) |
| `installmentValue` | Fixed monthly payment |
| `totalAmount` | Sum of all installments |
| `totalInterest` | Total cost over the loan term (interest + IOF) |
| `monthlyRatePct` | Monthly interest rate (%) |
| `yearlyRatePct` | Equivalent yearly rate (%, compound) |
| `monthlyCetPct` | Monthly CET (%) |
| `yearlyCetPct` | Yearly CET (%) |
| `iof.termIof` | IOF accrued by term (daily rate × days) |
| `iof.complementaryIof` | Flat-rate IOF (0.38%) |
| `iof.inCashIof` | Total IOF if paid upfront |
| `iof.fundedIof` | IOF portion financed within the loan |

---

## Getting Started

### Prerequisites

- Go 1.21+
- Node.js (optional — for Bruno CLI integration tests)
- Make

### Quick start

```bash
make run                   # Start server in background on :8080
make test                  # Run unit tests
make check                 # Run unit + integration tests
make stop                  # Stop the server
```

---

## Commands

All project operations are available through `make`. Run `make` or `make help` to see the full list.

| Command | Description |
|---|---|
| `make help` | Show available commands |
| `make build` | Compile binary to `bin/loan-simulator` |
| `make test` | Run unit tests |
| `make run` | Start server on :8080 (background) |
| `make stop` | Stop the running server |
| `make integration-test` | Run Bruno collection (server must be running) |
| `make integration-test-full` | Start server → run Bruno tests → stop server |
| `make check` | Run unit tests + integration tests (full pipeline) |
| `make clean` | Remove build artifacts |

---

## Project Structure

```
loan-simulator-go/
├── main.go                     # HTTP server entry point
├── Makefile                    # Build, test, and integration targets
├── api/
│   ├── simulator.go            # Handler + simulation logic
│   └── simulator_test.go       # Unit tests
├── financial/
│   ├── iofconfig/iof.go        # IOF tax calculation
│   ├── pmtconfig/pmt.go        # PMT (installment) formula
│   └── rateconfig/rate.go      # Interest rate conversions
└── bruno-collection/           # API test collection (Bruno)
```

---

## Limitations

This is a **demonstration project**. Simplifications relative to a production engine:

| Area | What's simplified | Impact |
|---|---|---|
| Calendar | Fixed 30-day months | ~0.2% IOF variance |
| Rounding | `math.Round` (not banker's rounding) | ≤ R$0.01 per operation |
| IOF rule | Only `Funded365` (standard PF) | None for typical use |
| TC | Not supported | CET excludes origination fee |
| Insurance | Not supported | CET excludes credit insurance |
| CET scope | Interest + IOF only | Production must include all costs |
| IOF rates | Hardcoded for PF (individuals) | PJ rates differ |
| Calculation mode | Forward only (rate → installment) | No reverse rate finding |

⚠️ IOF rates may change by presidential decree. Verify current legislation before production use.

---

## Production Architecture

The production system this demo is based on includes:

- **AWS Lambda** + SAM (Serverless Application Model)
- **JWT authentication** via Amazon Cognito
- Multiple calculation modes (by loan value, installment, funded value, reverse rate via Regula Falsi)
- **XIRR** with real calendar dates for precise CET
- **FGTS credit** withdrawals simulation
- Configurable IOF parameters per product
- **API Gateway** with custom domain

---

## Glossary

| Term | Portuguese | Description |
|---|---|---|
| **CET** | Custo Efetivo Total | True annual cost of credit (IRR of all cash flows). Mandatory disclosure in Brazil. |
| **IOF** | Imposto sobre Operações Financeiras | Federal tax on credit operations. Has a daily component and a flat complementary rate. |
| **Term IOF** | IOF por prazo | IOF calculated as daily rate × number of days from issue to each installment's due date. |
| **Complementary IOF** | IOF complementar | Flat 0.38% tax on the total funded amount. |
| **Funded Value** | Valor Financiado | Total amount financed: loan + taxes embedded in the operation. |
| **Funded IOF** | IOF financiado | The IOF amount that is financed (added to the loan) rather than paid upfront. |
| **In Cash IOF** | IOF à vista | The IOF amount if it were paid upfront, before being financed. |
| **Grace Period** | Carência | Days between contract signing and the first installment due date. |
| **PMT** | Prestação / Parcela | Fixed monthly payment calculated via the Price table formula. |
| **Price Table** | Tabela Price | French amortization system — fixed installments with decreasing interest and increasing principal over time. |
| **TC** | Tarifa de Cadastro | Origination/registration fee charged at loan inception. |
| **XIRR** | TIR com datas | Extended Internal Rate of Return using actual calendar dates (used in production for precise CET). |
| **PF** | Pessoa Física | Individual (natural person). IOF rate: 0.0082%/day. |
| **PJ** | Pessoa Jurídica | Legal entity (company). IOF rate: 0.0041%/day. |
| **BACEN** | Banco Central do Brasil | Brazilian Central Bank — regulates credit operations and CET disclosure. |

---

## Regulatory References

| Regulation | Subject |
|---|---|
| Decreto nº 6.306/2007 | IOF on credit operations |
| Resolução CMN nº 4.881/2020 | CET methodology and mandatory disclosure |
| Circular BACEN nº 3.274/2005 | CET calculation methodology |
| Resolução CMN nº 4.949/2021 | General conditions for credit operations |

---

## Author

**Luiz Lima** — Principal Engineer & Engineering Manager  
[LinkedIn](https://linkedin.com/in/luiz-lima-1a133144)

## License

MIT — see [LICENSE](LICENSE) for details.
