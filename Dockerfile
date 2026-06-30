# Use the official Go image as the base image
FROM golang:1.23 AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy the Go modules manifests
COPY go.mod go.sum ./

# Download Go modules
RUN go mod download

# Copy the application source code
COPY . .

# Build statically-linked binaries so they run on a minimal runtime image
RUN CGO_ENABLED=0 go build -o chartpress ./cmd/chartpress
RUN CGO_ENABLED=0 go build -o chartpress-server ./cmd/server
RUN CGO_ENABLED=0 go build -o operator ./cmd/operator


FROM alpine:3.20
# Set the working directory inside the container
WORKDIR /app
# Copy the built binaries from the builder stage (the operator runs from the same
# image via command: ["/app/operator"]; the backend stays the default CMD)
COPY --from=builder /app/chartpress /app/chartpress-server /app/operator ./
# The server and operator read the chart templates from ./templates at runtime
COPY --from=builder /app/templates ./templates
# Expose the port the service will run on
EXPOSE 8080

# Command to run the application
CMD ["./chartpress-server"]
