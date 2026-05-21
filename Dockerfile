FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -trimpath" -o go-notepad .

FROM scratch
WORKDIR /app
COPY --from=builder /app/go-notepad .
COPY --from=builder /app/static ./static
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
VOLUME ["/app/data"]
EXPOSE 3000
CMD ["./go-notepad"]
