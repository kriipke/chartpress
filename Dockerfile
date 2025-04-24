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


FROM debian:12.1-slim as setup
# Install required dependencies, including glibc
#RUN apt-get update && apt-get install -y libc6 && apt-get clea
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
