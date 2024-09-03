# Use an official Go image as the base
FROM golang:1.20-alpine

# Install dependencies needed for Playwright, including browser binaries
RUN apk add --no-cache chromium

# Set the working directory inside the container
WORKDIR /app

# Copy the Go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application
COPY . .

# Build the Go application
RUN go build -o playwright_exporter

# Expose port for Prometheus metrics
EXPOSE 8080

# Set the entry point to run the Playwright tests
CMD ["./playwright_exporter"]
