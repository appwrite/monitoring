FROM golang:1.21-alpine@sha256:2414035b086e3c42b99654c8b26e6f5b1b1598080d65fd03c7f499552ff4dc94 AS builder

WORKDIR /app

RUN apk add --no-cache gcc musl-dev

COPY . .

RUN go mod download

RUN go build -o monitoring .

FROM alpine:3.19@sha256:6baf43584bcb78f2e5847d1de515f23499913ac9f12bdf834811a3145eb11ca1 AS final

WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY --from=builder /app/monitoring /usr/local/bin/monitoring

RUN chmod +x /usr/local/bin/monitoring