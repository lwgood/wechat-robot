package api

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"text/template"
	"time"

	"wechat-robot/model"
)

func WechatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "only POST supported", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()

	// 1. webhook key 优先级：
	// URL query ?key=xxx > ENV WEBHOOK_ID > 默认值
	webhookID := r.URL.Query().Get("key")
	if webhookID == "" {
		webhookID = os.Getenv("WEBHOOK_ID")
	}
	if webhookID == "" {
		webhookID = "0dee1696-5e24-4830-afab-8cce0b64d866"
	}

	// 2. 企业微信官方 Webhook 地址（固定）
	wechatURL := "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=" + webhookID

	// 3. 解析 Alertmanager payload
	var payload model.WebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		log.Printf("decode payload failed: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("received %d alerts", len(payload.Alerts))

	// 4. 构建模板视图
	type View struct {
		Alerts struct {
			Firing   []model.Alert
			Resolved []model.Alert
		}
	}

	var view View
	for _, a := range payload.Alerts {
		if a.Status == "firing" {
			view.Alerts.Firing = append(view.Alerts.Firing, a)
		} else if a.Status == "resolved" {
			view.Alerts.Resolved = append(view.Alerts.Resolved, a)
		}
	}

	// 5. 渲染模板
	tmpl, err := template.ParseFiles("templates/wechat.tmpl")
	if err != nil {
		log.Printf("template parse error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, view); err != nil {
		log.Printf("template execute error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 6. 构造企业微信消息体
	body := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"content": buf.String(),
		},
	}

	b, _ := json.Marshal(body)

	// 7. 超时 + 重试（最小实现）
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	var lastErr error
	for i := 1; i <= 3; i++ {
		req, err := http.NewRequest(http.MethodPost, wechatURL, bytes.NewReader(b))
		if err != nil {
			lastErr = err
			log.Printf("create request failed: %v", err)
			break
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			log.Printf("sent alerts to wechat, status=%s", resp.Status)
			respondOK(w)
			return
		}

		lastErr = err
		log.Printf("wechat send failed (retry %d/3): %v", i, err)
		time.Sleep(time.Duration(i) * time.Second)
	}

	log.Printf("wechat send finally failed: %v", lastErr)
	respondOK(w)
}

// 统一成功响应，避免 Alertmanager hang
func respondOK(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

// 健康检查
func HealthzHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
