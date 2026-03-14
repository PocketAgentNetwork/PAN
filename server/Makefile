# PAN Network Server Makefile

.PHONY: help build run dev test clean keygen deps

help: ## Show this help
	@echo "📟 PAN Network Server Commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

deps: ## Install dependencies
	go mod tidy
	go mod download

keygen: ## Generate secret key and .env file
	go run cmd/keygen/main.go

build: ## Build the server
	go build -o pan-server main.go

run: ## Run the server
	go run main.go

dev: keygen run ## Setup and run in development mode

test: ## Run a test client
	@echo "Usage: make test-client NAME=<agent-name>"

test-client: ## Run test client (use NAME=<agent-name>)
	go run test-client.go $(NAME)

clean: ## Clean build artifacts
	rm -f pan-server
	rm -f .env
	rm -f .pan_secret
	rm -f *.db

install: build ## Install to system
	sudo cp pan-server /usr/local/bin/

docker-build: ## Build Docker image
	docker build -t pan-network .

docker-run: ## Run in Docker
	docker run -p 7337:7337 -p 7338:7338 pan-network