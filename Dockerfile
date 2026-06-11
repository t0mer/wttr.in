# ========================
# Builder stage (Ubuntu/Debian-based)
# ========================
FROM golang:1.26-bookworm AS builder

WORKDIR /app

# Install build dependencies + fonts (Debian packages)
RUN apt-get update && apt-get install -y --no-install-recommends \
    bash \
    rsync \
    gcc \
    libc6-dev \
    fontconfig \
    fonts-dejavu-core \
    fonts-noto-core \
    fonts-noto-cjk \
    fonts-wqy-zenhei \
    fonts-symbola \
    fonts-motoya-l-cedar \
    fonts-lexi-gulim \
    && rm -rf /var/lib/apt/lists/*

# Cache Go modules
COPY go.mod go.sum ./
RUN go mod tidy && go mod download

# Copy source code
COPY . .

# Run the official build process
RUN bash build.sh build

# ========================
# Runtime stage (Debian to match builder glibc)
# ========================
FROM debian:bookworm-slim

WORKDIR /app

# Minimal runtime dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates curl && \
    rm -rf /var/lib/apt/lists/* && \
    useradd -r -u 1000 -m wttr && \
    mkdir -p /app/cache /app/data /app/log && \
    chown -R wttr:wttr /app

# Copy the built binary and entrypoint
COPY --from=builder /app/srv /app/bin/srv
COPY scripts/entrypoint.sh /app/bin/entrypoint.sh

# Environment variables
ENV WTTR_MYDIR="/app"
ENV WTTR_LISTEN_HOST="0.0.0.0"
ENV WTTR_LISTEN_PORT="8080"

USER wttr

EXPOSE 8080

ENTRYPOINT ["/app/bin/entrypoint.sh"]
CMD ["/app/bin/srv", "srv", "/app/config/config.yaml"]
