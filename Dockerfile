FROM harbor-vpc.diact.com/library/golang:1.20-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY main.go .
COPY api/ ./api/

RUN go build -o wechat-robot main.go

FROM harbor-vpc.diact.com/library/alpine:3.18

WORKDIR /app

RUN apk add --no-cache curl ca-certificates tzdata \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone \
    && update-ca-certificates \
    && addgroup -S app \
    && adduser -S -G app app

COPY --from=build --chown=app:app /app/wechat-robot .

USER app

EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=3s CMD curl -f http://localhost:8080/healthz || exit 1

CMD ["./wechat-robot"]
