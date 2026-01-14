FROM golang:1.22-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/queue-service ./cmd/queue-service

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /bin/queue-service /app/queue-service
EXPOSE 8080
ENTRYPOINT ["/app/queue-service"]
