# Wechat Robot

Alertmanager 企业微信机器人转发服务。

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
  wechat-robot:1.0.0
```

## 环境变量

| 变量名          | 是否必须 | 说明                              |
| ------------ | ---- | ------------------------------- |
| `WEBHOOK_ID` | 是    | 企业微信群机器人的 webhook key；也可通过请求 URL 的 `key` 参数单次覆盖 |


## Alertmanager 配置

```yaml
templates:
  - /etc/alertmanager/templates/wechat.tmpl

receivers:
- name: wechat-robot
  webhook_configs:
  - url: http://wechat-robot:8080/wechat-robot/send
    send_resolved: true
    payload:
      msgtype: markdown
      markdown:
        content: '{{ template "wechat.tmpl" . }}'

route:
  receiver: wechat-robot
  group_by: ['alertname']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 4h
  routes:
  - matchers:
    - severity="critical"
    receiver: wechat-robot
```
