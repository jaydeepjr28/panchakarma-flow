.PHONY: all build test proto infra-up infra-down infra-status seed lint clean run-backend stop-backend

SHELL := /bin/bash
BIN_DIR := bin

SERVICES := api-gateway identity-service clinic-service appointment-service therapy-service notification-service feedback-analytics-service

all: proto build test

## Protocol Buffers Compilation
proto:
	@echo "==> Compiling Protocol Buffers..."
	@protoc -I. \
		--go_out=. --go_opt=module=github.com/jaydeepjr28/panchakarma-flow \
		--go-grpc_out=. --go-grpc_opt=module=github.com/jaydeepjr28/panchakarma-flow \
		proto/clinic/v1/clinic_service.proto \
		proto/appointment/v1/appointment_service.proto \
		proto/identity/v1/identity_service.proto
	@echo "==> Protobuf compilation complete."

## Build All Standalone Microservice Binaries
build: proto
	@echo "==> Building all microservices..."
	@mkdir -p $(BIN_DIR)
	@for svc in $(SERVICES); do \
		echo "Building $$svc..."; \
		go build -o $(BIN_DIR)/$$svc ./cmd/$$svc || exit 1; \
	done
	@echo "==> All microservices built successfully in $(BIN_DIR)/"

## Run Automated Unit Tests
test:
	@echo "==> Running automated tests across all packages..."
	@go test -v -race ./pkg/...

## Linting and Code Hygiene
lint:
	@echo "==> Checking code with go vet..."
	@go vet ./...

## Local Infrastructure (PostgreSQL 16, Redis 7, Apache Kafka 3.7)
infra-up:
	@echo "==> Starting local infrastructure with Docker Compose..."
	@docker compose up -d
	@echo "==> Infrastructure starting. Check status with: make infra-status"

infra-down:
	@echo "==> Stopping local infrastructure..."
	@docker compose down

infra-status:
	@echo "==> Checking infrastructure health..."
	@docker compose ps

## Seed Baseline Data (Clinics, Rooms, Doctors, Packages)
seed:
	@echo "==> Seeding baseline clinical data..."
	@docker compose exec -T postgres psql -U panchakarma -d clinic_db -f - < scripts/seed_data.sql
	@echo "==> Seed data applied successfully."

## Clean Build Artifacts
clean:
	@echo "==> Cleaning build artifacts..."
	@rm -rf $(BIN_DIR)
	@echo "==> Clean complete."

## Run / Stop Backend Services
run-backend:
	@echo "==> Starting all 7 backend services..."
	@powershell -ExecutionPolicy Bypass -File ../run-backend.ps1

stop-backend:
	@echo "==> Stopping all backend services..."
	@powershell -ExecutionPolicy Bypass -File ../stop-backend.ps1 2>/dev/null || true

