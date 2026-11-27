# Wechat Robot

Alertmanager v0.24.0 企业微信机器人告警转发服务。由于 v0.24.0 的 webhook 不支持自定义 payload，本服务使用挂载到 `/app/templates/wechat.tmpl` 的模板，将标准 Alertmanager 告警渲染为企微 Markdown，再转发给群机器人。

```text
Alertmanager
    |
    | HTTP POST (Webhook)
    v
wechat-robot (:8080)
    |
    | HTTPS POST
    v
企业微信群机器人
```

## Build & Start

```bash
docker build -t wechat-robot:1.0.0 .

docker run -d --name wechat-robot -p 8080:8080 \
  -e WEBHOOK_ID=your_webhook_key \
  -v "$PWD/templates:/app/templates:ro" \
  wechat-robot:1.0.0
```

## 环境变量

| 变量名          | 是否必须 | 说明                              |
| ------------ | ---- | ------------------------------- |
| `WEBHOOK_ID` | 是    | 企业微信群机器人的 webhook key；也可通过请求 URL 的 `key` 参数单次覆盖 |


## Alertmanager 配置

```yaml
receivers:
- name: wechat-robot
  webhook_configs:
  - url: http://wechat-robot:8080/wechat-robot/send
    send_resolved: true

route:
  receiver: wechat-robot
  group_by: ['alertname']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 1h
  routes:
  - receiver: wechat-robot
    matchers:
    - severity="critical"
    group_wait: 5m
    group_interval: 10m
    repeat_interval: 30m
  - receiver: wechat-robot
    matchers:
    - severity="warning"
    group_wait: 5m
    group_interval: 10m
    repeat_interval: 6h
```
