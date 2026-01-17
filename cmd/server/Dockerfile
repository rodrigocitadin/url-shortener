FROM golang:1.25.5-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN go build -o api_server ./cmd/server/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/api_server .

EXPOSE 3030
CMD ["./api_server"]
