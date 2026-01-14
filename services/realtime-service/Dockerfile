FROM golang:1.22-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/realtime-service ./cmd/realtime-service

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /bin/realtime-service /app/realtime-service
EXPOSE 8085
ENTRYPOINT ["/app/realtime-service"]
