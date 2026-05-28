FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go env -w GOPROXY=https://goproxy.cn,direct
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o go-notepad .

FROM alpine:latest
WORKDIR /app
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories && apk --no-cache add ca-certificates && mkdir -p /app/data
COPY --from=builder /app/go-notepad .
COPY --from=builder /app/static ./static
VOLUME ["/app/data"]
EXPOSE 3000
CMD ["./go-notepad"]
