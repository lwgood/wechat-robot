# Wechat Robot

轻量化企业微信告警 Alertmanager webhook 并发送 Markdown 告警。

### 整体架构
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

### 环境变量

| 变量名          | 是否必须 | 说明                              |
| ------------ | ---- | ------------------------------- |
| `WEBHOOK_ID` | 否    | 企业微信机器人 webhook key，不设置则使用内置默认值 |

### 运行方式
```bash
# docker 
docker run -d \
  -p 8080:8080 \
  -e WEBHOOK_ID=0dee1696-5e24-4830-afab-8cce0b64d866 \
  --name wechat-robot \
  wechat-robot:v1.0.4

# docker-compose
version: "3.8"

services:
  wechat-robot:
    image: registry.cn-hangzhou.aliyuncs.com/basex/wechat-robot:1.0.4
    container_name: wechat-robot
    restart: unless-stopped
    user: "0"
    environment:
      WEBHOOK_ID: "0dee1696-5e24-4830-afab-8cce0b64d866"
    ports:
      - "8080:8080"
    # volumes:
    #   - ./templates:/app/templates:ro

# 健康检查
curl http://localhost:8080/healthz
```

### Alertmanager 配置示例
```yaml
receivers:
- name: wechat-robot
  webhook_configs:
  - url: http://wechat-robot:8080/wechat-robot/send?key=0dee1696-5e24-4830-afab-8cce0b64d866  # webhook_id 可选，如果未指定则为默认值: 0dee1696-5e24-4830-afab-8cce0b64d866 
    send_resolved: true

# 路由示例
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

### 测试
```bash
curl -X POST \
  -H "Content-Type: application/json" \
  --data @sample-alert.json \
  http://localhost:8080/wechat-robot/send?key=0dee1696-5e24-4830-afab-8cce0b64d866
```
