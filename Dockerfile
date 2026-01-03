# Build stage
FROM golang:1.23-alpine AS builder

# Install git and ca-certificates (needed for fetching dependencies)
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build arguments for version info
ARG VERSION=dev
ARG GIT_COMMIT=none
ARG BUILD_DATE=unknown

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X github.com/lugondev/go-carbon/cmd/carbon/cmd.Version=${VERSION} \
    -X github.com/lugondev/go-carbon/cmd/carbon/cmd.GitCommit=${GIT_COMMIT} \
    -X github.com/lugondev/go-carbon/cmd/carbon/cmd.BuildDate=${BUILD_DATE}" \
    -o /app/carbon ./cmd/carbon

# Final stage
FROM alpine:3.19

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/carbon /usr/local/bin/carbon

# Create non-root user
RUN addgroup -g 1000 carbon && \
    adduser -u 1000 -G carbon -s /bin/sh -D carbon

USER carbon

ENTRYPOINT ["carbon"]
CMD ["--help"]
