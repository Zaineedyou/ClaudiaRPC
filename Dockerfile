# Stage 1: Build the Go application
FROM golang:1.26-alpine AS builder

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
# We target the main file in cmd/server/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o claudia-rpc ./cmd/server/main.go

# Stage 2: Final lightweight image
FROM alpine:latest

# Install ca-certificates for secure connections
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /app/claudia-rpc .

# Copy static files
COPY --from=builder /app/static ./static

# Expose the port the app runs on (assuming default port, adjust if necessary)
EXPOSE 8080

# Command to run the executable
CMD ["./claudia-rpc"]
