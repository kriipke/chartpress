# syntax=docker/dockerfile:1.7
# SPDX-License-Identifier: Apache-2.0
FROM golang:1.23.6-bookworm@sha256:462f68e1109cc0415f58ba591f11e650b38e193fddc4a683a3b77d29be8bfb2c AS builder

ENV GOTOOLCHAIN=local

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


FROM alpine:3.20.3@sha256:1e42bbe2508154c9126d48c2b8a75420c3544343bf86fd041fb7527e017a4b4a
ARG VCS_REF=unknown
ARG SOURCE_URL=https://github.com/kriipke/chartpress
LABEL org.opencontainers.image.source=$SOURCE_URL \
      org.opencontainers.image.revision=$VCS_REF \
      org.opencontainers.image.licenses=Apache-2.0
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
