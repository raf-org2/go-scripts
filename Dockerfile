# syntax=docker/dockerfile:1

FROM golang:1.21-alpine AS dev

WORKDIR /app

RUN apk add --no-cache git

# Copy go mod/sum and download dependencies first (cache layer)
COPY go.mod ./
COPY go.sum ./
RUN go mod download

# Copy all Go source files (only this step will invalidate cache on code change)
COPY *.go ./

# Build all binaries
RUN go build -o organization-check organization-check.go && \
    go build -o get_org_repos get_org_repos.go && \
    go build -o create_org_config create_org_config.go && \
    go build -o update_org_config update_org_config.go && \
    go build -o add_repo_to_config add_repo_to_config.go

# Final minimal image (optional, for prod/test)
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=dev /app/organization-check /app/
COPY --from=dev /app/get_org_repos /app/
COPY --from=dev /app/create_org_config /app/
COPY --from=dev /app/update_org_config /app/
COPY --from=dev /app/add_repo_to_config /app/

# Set default command (edit as needed)
CMD ["./create_org_config"]