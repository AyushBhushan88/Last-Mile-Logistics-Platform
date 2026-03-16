# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Install build dependencies for CGO (required by h3-go)
RUN apk add --no-cache gcc musl-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o ingestor ./cmd/ingestor
RUN go build -o spatial-indexer ./cmd/spatial-indexer

# Run stage
FROM alpine:3.19

WORKDIR /root/

# Install runtime dependencies for h3-go if needed
RUN apk add --no-cache libstdc++

COPY --from=builder /app/ingestor .
COPY --from=builder /app/spatial-indexer .
COPY config/config.yaml ./config/

EXPOSE 8080 50051

# Default command (overridden in docker-compose for spatial-indexer)
CMD ["./ingestor"]
