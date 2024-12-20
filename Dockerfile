FROM golang:1.21-alpine AS builder

WORKDIR /app

RUN apk add --no-cache gcc musl-dev

COPY . .

RUN go mod download

RUN go build -o monitoring .

FROM alpine:3.19 AS final

WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY --from=builder /app/monitoring /usr/local/bin/monitoring

RUN chmod +x /usr/local/bin/monitoring