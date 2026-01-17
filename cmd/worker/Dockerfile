FROM golang:1.25.5-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN go build -o worker_service ./cmd/worker/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/worker_service .
CMD ["./worker_service"]
