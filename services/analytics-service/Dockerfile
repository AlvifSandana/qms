FROM golang:1.22-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/analytics-service ./cmd/analytics-service

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /bin/analytics-service /app/analytics-service
EXPOSE 8084
ENTRYPOINT ["/app/analytics-service"]
