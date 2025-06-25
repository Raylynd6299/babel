# docker/progress-service.Dockerfile
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o progress-service ./cmd/progress-service

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/

# Copy the binary
COPY --from=builder /app/progress-service .

# Expose port
EXPOSE 8003

# Run the application
CMD ["./progress-service"]
