FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
  -ldflags="-s -w" \
  -o /bin/leadflow-api ./cmd/api

RUN CGO_ENABLED=0 GOOS=linux go build \
  -o /bin/leadflow-worker ./cmd/worker

FROM alpine:3.22

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /bin/leadflow-api /bin/leadflow-api
COPY --from=builder /bin/leadflow-worker /bin/leadflow-worker
COPY --from=builder /app/templates ./templates

EXPOSE 8080

CMD ["/bin/leadflow-api"]