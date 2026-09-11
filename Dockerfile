FROM golang:1.26-bookworm AS builder

ENV GOMAXPROCS=1
WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o app ./cmd

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata wget \
    && adduser -D -u 10001 appuser

WORKDIR /app

COPY --from=builder /build/app /app/app
COPY --from=builder /build/migrations /app/migrations

USER appuser

EXPOSE 8080

ENTRYPOINT ["/app/app"]