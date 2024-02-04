# Start from the official Golang image
FROM golang:alpine

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files to the working directory
COPY go.mod go.sum ./

# Download and install any dependencies defined in go.mod
RUN go mod download

# Copy the rest of the application source code to the working directory
COPY . .

# Copy .env to /app directiory
COPY .env /app/.env

# Build the Go application
RUN go build -o go-yukngaji cmd/main.go

# Command to run the executable
CMD ["./go-yukngaji"]