FROM golang:1.20-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod tidy

COPY main.go .
COPY api/ ./api/
COPY model/ ./model/
COPY templates/ ./templates/

RUN go build -o wechat-robot main.go

FROM alpine:3.18

WORKDIR /app

RUN apk add --no-cache \
    curl \
    net-tools \
    busybox-extras \
    bash \
    iproute2 \
    netcat-openbsd

COPY --from=build /app/wechat-robot .
COPY --from=build /app/templates ./templates

EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=3s CMD curl -f http://localhost:8080/healthz || exit 1

CMD ["./wechat-robot"]
