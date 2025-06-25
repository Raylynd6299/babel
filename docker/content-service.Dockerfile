# docker/content-service.Dockerfile
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o content-service ./cmd/content-service

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/

# Copy the binary
COPY --from=builder /app/content-service .

# Expose port
EXPOSE 8002

# Run the application
CMD ["./content-service"]
