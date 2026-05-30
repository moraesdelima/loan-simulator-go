.PHONY: build test run stop integration-test integration-test-full check clean help

# Default target
.DEFAULT_GOAL := help

## help: Show available targets
help:
	@echo ""
	@echo "  Loan Simulator — Available commands"
	@echo "  ────────────────────────────────────"
	@echo ""
	@echo "  make build                 Build binary to bin/"
	@echo "  make test                  Run unit tests"
	@echo "  make run                   Start server on :8080 (background)"
	@echo "  make stop                  Stop the running server"
	@echo "  make integration-test      Run Bruno collection (server must be running)"
	@echo "  make integration-test-full Start server, run Bruno tests, stop server"
	@echo "  make check                 Run unit tests + integration tests"
	@echo "  make clean                 Remove build artifacts"
	@echo ""

## build: Compile the application
build:
	@echo "▸ Building binary..."
	@go build -o bin/loan-simulator .
	@echo "✓ Build complete → bin/loan-simulator"

## test: Run unit tests
test:
	@echo "▸ Running unit tests..."
	@go test ./... -v
	@echo ""
	@echo "✓ Unit tests passed"

## run: Start the server locally (background)
run:
	@if lsof -ti:8080 > /dev/null 2>&1; then \
		echo "✗ Port 8080 is already in use. Stop the existing process or run:"; \
		echo "  make stop"; \
		exit 1; \
	fi
	@go run main.go & echo $$! > /tmp/loan-simulator.pid
	@sleep 1
	@echo "▸ Server running on http://localhost:8080 (PID: $$(cat /tmp/loan-simulator.pid))"
	@echo "  To stop: make stop"

## stop: Stop the running server
stop:
	@if [ -f /tmp/loan-simulator.pid ]; then \
		kill $$(cat /tmp/loan-simulator.pid) 2>/dev/null && \
		rm -f /tmp/loan-simulator.pid && \
		echo "✓ Server stopped"; \
	elif lsof -ti:8080 > /dev/null 2>&1; then \
		kill $$(lsof -ti:8080) 2>/dev/null && \
		echo "✓ Server stopped"; \
	else \
		echo "▸ No server running on :8080"; \
	fi

## integration-test: Run Bruno collection (requires server on :8080)
integration-test:
	@echo "▸ Running integration tests (Bruno collection)..."
	@cd bruno-collection && npx @usebruno/cli run --env local
	@echo ""
	@echo "✓ Integration tests passed"

## integration-test-full: Start server, run tests, stop server
integration-test-full:
	@echo "▸ Starting server on :8080..."
	@lsof -ti:8080 | xargs kill 2>/dev/null || true
	@go run main.go & echo $$! > /tmp/loan-simulator.pid
	@sleep 2
	@echo "▸ Running integration tests..."
	@cd bruno-collection && npx @usebruno/cli run --env local; \
		EXIT_CODE=$$?; \
		kill $$(cat /tmp/loan-simulator.pid) 2>/dev/null; \
		rm -f /tmp/loan-simulator.pid; \
		if [ $$EXIT_CODE -eq 0 ]; then \
			echo ""; \
			echo "✓ Integration tests passed"; \
		else \
			echo ""; \
			echo "✗ Integration tests failed"; \
		fi; \
		exit $$EXIT_CODE

## check: Run all checks (unit + integration)
check:
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "  Running full check"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo ""
	@$(MAKE) --no-print-directory test
	@echo ""
	@$(MAKE) --no-print-directory integration-test-full
	@echo ""
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "  ✓ All checks passed"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

## clean: Remove build artifacts
clean:
	@echo "▸ Cleaning..."
	@rm -rf bin/
	@echo "✓ Clean complete"
