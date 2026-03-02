FROM golang:1.22-bullseye AS builder

# Install libvips for bimg
RUN apt-get update && apt-get install -y \
    libvips-dev \
    pkg-config \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o /dam ./cmd/server

# Runtime image
FROM debian:bullseye-slim

RUN apt-get update && apt-get install -y \
    libvips \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /dam /app/dam
COPY migrations/ /app/migrations/

EXPOSE 8080

CMD ["/app/dam"]
