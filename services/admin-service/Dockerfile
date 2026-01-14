FROM golang:1.22-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/admin-service ./cmd/admin-service

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /bin/admin-service /app/admin-service
EXPOSE 8083
ENTRYPOINT ["/app/admin-service"]
