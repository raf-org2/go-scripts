FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install git for go mod dependencies
RUN apk add --no-cache git

# Copy go mod file first
COPY go.mod ./

# Copy go.sum if it exists, otherwise create empty file
COPY go.su[m] ./
RUN touch go.sum

# Copy source code
COPY *.go ./

# Download dependencies and tidy
RUN go mod tidy && go mod download

# Build all applications
RUN go build -o organization-check organization-check.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy all binaries from builder stage
COPY --from=builder /app/organization-check .
