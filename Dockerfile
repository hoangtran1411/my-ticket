# Step 1: Build binary using official Go 1.25 Alpine image
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source files
COPY . .

# Build lightweight CGO-free Go binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o devticket main.go

# Step 2: Minimal runtime image
FROM alpine:3.20

WORKDIR /app

# Copy ca-certificates for potential TLS requests
RUN apk --no-cache add ca-certificates tzdata

# Copy built binary & templates/static files
COPY --from=builder /app/devticket /app/devticket
COPY --from=builder /app/templates /app/templates
COPY --from=builder /app/static /app/static

EXPOSE 8080

ENV PORT=8080

CMD ["/app/devticket"]
