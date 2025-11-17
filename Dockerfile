# Multi-stage build for both master and worker bots
FROM golang:1.24-alpine AS builder

# Install build dependencies
# gcc, musl-dev: Required for CGO
# opus-dev: Required for Discord voice connections
RUN apk add --no-cache git ca-certificates gcc musl-dev opus-dev

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build master binary (no voice, CGO disabled for smaller size)
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-w -s' -o master ./cmd/master

# Build worker binary (with voice support, CGO enabled for opus)
RUN CGO_ENABLED=1 GOOS=linux go build -a -ldflags '-w -s' -o worker ./cmd/worker

# Final stage
FROM alpine:latest

# Install runtime dependencies for voice connections
# - ca-certificates: HTTPS support
# - opus: Audio codec for Discord voice (runtime library)
# - ffmpeg: Audio processing utilities
# - libstdc++: C++ standard library (required by some audio libs)
RUN apk --no-cache add ca-certificates opus ffmpeg libstdc++

# Create app directory
WORKDIR /app

# Copy binaries from builder
COPY --from=builder /build/master /app/master
COPY --from=builder /build/worker /app/worker

# Copy translation files
COPY --from=builder /build/internal/core/i18n/translations /app/internal/core/i18n/translations

# Copy database migration files
COPY --from=builder /build/internal/core/database/migrations /app/internal/core/database/migrations

# Copy audio files for onboarding
COPY --from=builder /build/audio /app/audio

# Copy image assets for onboarding guides
COPY --from=builder /build/assets /app/assets

# Default command (can be overridden)
CMD ["/app/master"]