FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /api ./cmd/api
RUN CGO_ENABLED=0 go build -o /crawler ./cmd/crawler

FROM alpine:3.20 AS api
RUN apk --no-cache add ca-certificates
COPY --from=builder /api /api
CMD ["/api"]

FROM alpine:3.20 AS crawler
RUN apk --no-cache add ca-certificates
COPY --from=builder /crawler /crawler
CMD ["/crawler"]
