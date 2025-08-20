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

# Build organization-check
RUN go build -o organization-check organization-check.go

# Build get_org_repos
RUN go build -o get_org_repos get_org_repos.go

# Build create_org_config
RUN go build -o create_org_config create_org_config.go

# Build update_org_config
RUN go build -o update_org_config update_org_config.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy all binaries from builder stage
COPY --from=builder /app/organization-check /app/
COPY --from=builder /app/get_org_repos /app/

COPY --from=builder /app/create_org_config /app/
COPY --from=builder /app/update_org_config /app/

