FROM golang:1.22.0-alpine3.18 AS builder

WORKDIR /app

COPY . .

COPY go.mod go.sum ./
RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -o boomtown

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/boomtown .

EXPOSE 8080
CMD ["./boomtown"]
