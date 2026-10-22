FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
  -trimpath \
  -ldflags="-s -w" \
  -o /bin/leadflow-api ./cmd/api

RUN CGO_ENABLED=0 GOOS=linux go build \
  -trimpath \
  -ldflags="-s -w" \
  -o /bin/leadflow-worker ./cmd/worker

FROM alpine:3.22 AS api

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -g 10001 leadflow && \
    adduser -u 10001 -G leadflow -S leadflow

WORKDIR /app

COPY --from=builder /bin/leadflow-api /bin/leadflow-api
COPY --from=builder /app/templates ./templates

USER 10001:10001

EXPOSE 8080

CMD ["/bin/leadflow-api"]

FROM alpine:3.22 AS worker

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -g 10001 leadflow && \
    adduser -u 10001 -G leadflow -S leadflow

WORKDIR /app

COPY --from=builder /bin/leadflow-worker /bin/leadflow-worker

USER 10001:10001

CMD ["/bin/leadflow-worker"]