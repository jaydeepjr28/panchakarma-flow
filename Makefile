.PHONY: all build test proto lint clean infra-up infra-down infra-status seed backend-build backend-test frontend-install frontend-dev frontend-build

SHELL := /bin/bash

all: backend-build frontend-build

## ====================================================================
## Backend Commands (Go Microservices)
## ====================================================================
backend-build:
	@echo "==> Building all backend microservices..."
	@cd backend && $(MAKE) build

backend-test:
	@echo "==> Running backend automated tests..."
	@cd backend && $(MAKE) test

run-backend:
	@echo "==> Starting all 7 backend services..."
	@powershell -ExecutionPolicy Bypass -File ./run-backend.ps1

stop-backend:
	@echo "==> Stopping all backend services..."
	@powershell -ExecutionPolicy Bypass -File ./stop-backend.ps1 2>/dev/null || true

build: backend-build
test: backend-test
proto:
	@cd backend && $(MAKE) proto
lint:
	@cd backend && $(MAKE) lint

## ====================================================================
## Frontend Commands (Vite + React + TypeScript + Tailwind)
## ====================================================================
frontend-install:
	@echo "==> Installing frontend dependencies..."
	@cd frontend && npm install

frontend-dev:
	@echo "==> Starting frontend development server..."
	@cd frontend && npm run dev

frontend-build:
	@echo "==> Building frontend for production..."
	@cd frontend && npm run build

## ====================================================================
## Infrastructure (PostgreSQL 16, Redis 7, Apache Kafka 3.7)
## ====================================================================
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

seed:
	@echo "==> Seeding baseline clinical data..."
	@docker compose exec -T postgres psql -U panchakarma -d clinic_db -f - < backend/scripts/seed_data.sql
	@echo "==> Seed data applied successfully."
