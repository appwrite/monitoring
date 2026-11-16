FROM golang:1.27-alpine@sha256:70bd7ddea8b6cbe5d6943cde279d6b67feb10cc924fb9100d1d8757c1575b280 AS builder

WORKDIR /app

RUN apk add --no-cache gcc musl-dev

COPY . .

RUN go mod download

RUN go build -o monitoring .

FROM alpine:3.24@sha256:f59abe656b692d901e184be3445e084786602108599d2a51d229c0f5a48c71bf AS final

WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY --from=builder /app/monitoring /usr/local/bin/monitoring

RUN chmod +x /usr/local/bin/monitoring