FROM golang:1.27-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /bin/api ./cmd/api

FROM alpine:3.22
RUN apk add --no-cache ca-certificates && \
    adduser -D -u 10001 appuser
USER appuser

COPY --from=builder /bin/api /bin/api

EXPOSE 8080
ENTRYPOINT ["/bin/api"]
