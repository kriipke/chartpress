# Use the official Go image as the base image
FROM golang:1.23 AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy the Go modules manifests
COPY go.mod go.sum ./

# Download Go modules
RUN go mod download

# Copy the application source code
#COPY api.go cmd/* chartpress.yaml .
COPY . .

# Build the application
RUN go build -o chartpress ./cmd/chartpress
RUN go build -o chartpress-server ./cmd/server

 
FROM golang:1.23-bookworm
# Set the working directory inside the container
WORKDIR /app
# Copy the built binary from the builder stage
COPY --from=builder /app/chartpress /app/chartpress-server .
COPY templates/umbrella templates/umbrella
COPY templates/subchart templates/subchart
# Expose the port the service will run on
EXPOSE 8080

# Command to run the application
CMD ["./chartpress-server"]
