# Stage 1: Build
FROM golang:1.24-alpine AS builder

# Install dependencies
RUN apk add --no-cache tzdata

# Set timezone
ENV TZ=Asia/Jakarta

# Set working directory
WORKDIR /app

# Copy go mod files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the Go application
RUN go build -o go-yukngaji cmd/main.go

# Stage 2: Run (minimal image)
FROM alpine:latest

# Install timezone data
RUN apk add --no-cache tzdata ca-certificates

# Set timezone
ENV TZ=Asia/Jakarta

# Set working directory
WORKDIR /app

# Copy binary and required files from builder
COPY --from=builder /app/go-yukngaji .
COPY --from=builder /app/template ./template

# Copy .env if your app uses it
COPY .env .env

# Expose port if needed (optional, depending on your app)
# EXPOSE 8080

# Run the app
CMD ["./go-yukngaji"]