FROM golang:1.25-alpine AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/api ./cmd/api

FROM alpine:3.20
RUN adduser -D -g '' appuser
WORKDIR /app

COPY --from=builder /out/api /app/api

USER appuser
EXPOSE 8080
ENTRYPOINT ["/app/api"]
