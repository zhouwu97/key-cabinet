FROM golang:1.26-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git make

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/api cmd/api/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/migrate cmd/migrate/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/bin/api .
COPY --from=builder /app/bin/migrate .
COPY --from=builder /app/internal/config/config.example.yaml ./internal/config/config.yaml
COPY --from=builder /app/migrations ./migrations

EXPOSE 8080

CMD ["sh", "-c", "./migrate -path migrations -database \"$DATABASE_URL\" -command up && exec ./api"]
