SERVICE_NAME := metering-service
BINARY       := bin/$(SERVICE_NAME)
MAIN         := ./cmd/apiserver

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-16s\033[0m %s\n", $$1, $$2}'

.PHONY: run
run: ## Run the server (no external dependencies required)
	go run $(MAIN) server

.PHONY: config
config: ## Print the resolved configuration as JSON
	go run $(MAIN) config

.PHONY: build
build: ## Build the binary
	go build -o $(BINARY) $(MAIN)

.PHONY: test
test: ## Run all tests with the race detector
	go test -race ./...

.PHONY: cover
cover: ## Run tests and report total coverage
	go test -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -n 1

.PHONY: cover-html
cover-html: cover ## Open the HTML coverage report
	go tool cover -html=coverage.out

.PHONY: stress
stress: ## Load-test the live API with vegeta (requires vegeta on PATH)
	./scripts/stress_test.sh

.PHONY: lint
lint: ## Run go vet
	go vet ./...

.PHONY: tidy
tidy: ## Tidy go modules
	go mod tidy

.PHONY: clean
clean: ## Remove build artifacts
	@rm -rf bin coverage.out
	@go clean
