FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o go-notepad .

FROM alpine:latest
WORKDIR /app
RUN apk --no-cache add ca-certificates && mkdir -p /app/data
COPY --from=builder /app/go-notepad .
COPY --from=builder /app/static ./static
VOLUME ["/app/data"]
EXPOSE 3000
CMD ["./go-notepad"]
