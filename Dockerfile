FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
# Allow newer go.mod on older toolchains
ENV GOTOOLCHAIN=auto
RUN go mod download

COPY . .

RUN go build -o sentry-go cmd/main.go

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/sentry-go .

EXPOSE 8080

CMD ["./sentry-go"]
