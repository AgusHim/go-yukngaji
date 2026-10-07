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

# Runner migrasi ikut dibangun supaya `migrate up` dapat dijalankan di server
# lewat container yang sama. Tanpa ini, langkah migrasi pada urutan rilis
# (docs/deployment.md) tidak punya biner untuk dijalankan.
RUN go build -o migrate ./cmd/migrate

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

# Berkas migrasi dibutuhkan `./migrate`, yang membaca `sql/migrations`
# relatif terhadap direktori kerja.
COPY --from=builder /app/migrate .
COPY --from=builder /app/sql/migrations ./sql/migrations

# Rahasia TIDAK dibakar ke dalam image. `godotenv.Load()` bersifat opsional
# (cmd/main.go), jadi variabel cukup datang dari environment saat
# `docker run` — lihat target `runimage` di MakeFile. Berkas `.env` juga
# didaftarkan di .dockerignore supaya tidak ikut lapisan build.

# Expose port if needed (optional, depending on your app)
# EXPOSE 8080

# Run the app
CMD ["./go-yukngaji"]