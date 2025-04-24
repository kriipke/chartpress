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
RUN go build -o chartpress .

# Use a minimal base image for the final container
FROM go:1.23-bullseye AS setup

# Set the working directory inside the container
WORKDIR /app

# Copy the built binary from the builder stage
COPY --from=builder /app/chartpress .

# Copy the templates directory
COPY ./templates ./templates

# Expose the port the service will run on
EXPOSE 8080

# Command to run the application
CMD ["./chartpress"]


FROM setup

# Copy the built binary from the builder stage
RUN apt-get update -y && apt-get install -y libc6 \
 && rm -rf /var/lib/apt/lists/*
 
# Expose the port the service will run on
EXPOSE 8080
WORKDIR /app
# Command to run the application
CMD ["./chartpress"]
