# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -o main ./cmd/main.go

# Runtime stage
FROM alpine:latest

WORKDIR /app

# Install required dependencies for SQLite
RUN apk add --no-cache sqlite-libs

# Copy the binary from builder
COPY --from=builder /app/main .
COPY --from=builder /app/template ./template
COPY --from=builder /app/static ./static
COPY --from=builder /app/.env .

# Expose port 8080
EXPOSE 8080

# Run the application
CMD ["./main"]