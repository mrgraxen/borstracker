FROM golang:1.22-alpine AS builder
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /bin/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /bin/price-fetcher ./cmd/price-fetcher

FROM alpine:3.19 AS api
RUN apk add --no-cache ca-certificates wget
WORKDIR /app
COPY --from=builder /bin/api /api
COPY migrations /app/migrations
EXPOSE 8080
USER nobody
ENTRYPOINT ["/api"]

FROM alpine:3.19 AS price-fetcher
RUN apk add --no-cache ca-certificates wget
WORKDIR /app
COPY --from=builder /bin/price-fetcher /price-fetcher
COPY migrations /app/migrations
EXPOSE 8081
USER nobody
ENTRYPOINT ["/price-fetcher"]
