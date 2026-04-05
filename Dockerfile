FROM golang:1.25-alpine AS builder
RUN apk add --no-cache ca-certificates tzdata git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /chatgpt2api .

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /chatgpt2api /chatgpt2api
COPY config.defaults.toml /app/config.defaults.toml
COPY _public /app/_public
RUN mkdir -p /app/data
EXPOSE 8200
ENV SERVER_HOST=0.0.0.0 SERVER_PORT=8200 TZ=Asia/Shanghai
CMD ["/chatgpt2api"]
