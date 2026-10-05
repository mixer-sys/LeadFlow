FROM golang:1.23-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /leadflow-api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o /leadflow-worker ./cmd/worker

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /leadflow-api /leadflow-api
COPY --from=builder /leadflow-worker /leadflow-worker

ENTRYPOINT ["/leadflow-api"]
