---
name: docker-go-devops
description: Multi-stage Docker packaging and CI/CD pipelines for Go applications.
---

# Docker & DevOps Skill for Go

This skill outlines packaging standards for single-binary Go applications with SQLite persistence.

## Key Rules

1. **Multi-Stage Build**:
   - Builder stage: `golang:1.25-alpine` (`CGO_ENABLED=0 GOOS=linux`).
   - Minimal runtime stage: `alpine:3.20` or `scratch`.

2. **Volume Persistence**:
   - Persist SQLite database file (`tickets.db`) using Docker volume mounts in `docker-compose.yml`.

3. **CI/CD Quality Gate**:
   - Run `golangci-lint` and `go test -v -race` before creating release binaries.
