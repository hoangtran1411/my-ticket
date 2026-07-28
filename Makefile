# DevTicket Pro Makefile

APP_NAME=devticket.exe
DB_FILE=tickets.db

.PHONY: all build run dev clean test fmt help reset-db

all: build

## help: Display available Makefile commands
help:
	@echo "DevTicket Pro - Available Makefile Commands:"
	@echo "  make run        - Run application directly (go run main.go)"
	@echo "  make build      - Build binary executable ($(APP_NAME))"
	@echo "  make test       - Run unit tests across all packages"
	@echo "  make fmt        - Format Go source code (go fmt)"
	@echo "  make clean      - Remove built binaries"
	@echo "  make reset-db   - Remove SQLite database to re-seed sample data on next start"

## run: Run the application
run:
	go run main.go

## dev: Alias for run
dev: run

## build: Build the production executable
build:
	@echo "Building $(APP_NAME)..."
	go build -o $(APP_NAME) .
	@echo "Build successful! Run ./$(APP_NAME) to launch."

## test: Run tests
test:
	go test -v ./...

## fmt: Format all Go files
fmt:
	go fmt ./...

## clean: Remove binaries
clean:
	@if exist $(APP_NAME) del /f /q $(APP_NAME)
	@echo "Cleaned binaries."

## reset-db: Remove SQLite DB file to re-trigger database seeding
reset-db:
	@if exist $(DB_FILE) del /f /q $(DB_FILE)
	@echo "SQLite database reset. Restart server to re-seed data."
