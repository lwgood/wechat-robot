package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"text/template"
	"time"
)

const (
	maxPayloadSize  = 1 << 20
	maxResponseSize = 1 << 20
	wechatEndpoint  = "https://qyapi.weixin.qq.com/cgi-bin/webhook/send"
)

var wechatClient = &http.Client{Timeout: 5 * time.Second}

type wechatMessage struct {
	MsgType string `json:"msgtype"`
	Markdown struct {
		Content string `json:"content"`
	} `json:"markdown"`
}

type alert struct {
	Status      string            `json:"status"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	StartsAt    string            `json:"startsAt"`
	EndsAt      string            `json:"endsAt"`
}

type webhookPayload struct {
	Alerts []alert `json:"alerts"`
}

type templateData struct {
	Alerts struct {
		Firing   []alert
		Resolved []alert
	}
}

// WechatHandler renders an Alertmanager webhook and forwards it to WeChat.
func WechatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "only POST supported", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()

	webhookID := strings.TrimSpace(r.URL.Query().Get("key"))
	if webhookID == "" {
		webhookID = strings.TrimSpace(os.Getenv("WEBHOOK_ID"))
	}
	if webhookID == "" {
		http.Error(w, "webhook key is not configured", http.StatusInternalServerError)
		return
	}

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxPayloadSize))
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	var payload webhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "invalid Alertmanager payload", http.StatusBadRequest)
		return
	}
	if len(payload.Alerts) == 0 {
		http.Error(w, "payload contains no alerts", http.StatusBadRequest)
		return
	}

	var data templateData
	for _, alert := range payload.Alerts {
		switch alert.Status {
		case "firing":
			data.Alerts.Firing = append(data.Alerts.Firing, alert)
		case "resolved":
			data.Alerts.Resolved = append(data.Alerts.Resolved, alert)
		}
	}
	if len(data.Alerts.Firing)+len(data.Alerts.Resolved) == 0 {
		http.Error(w, "payload contains no firing or resolved alerts", http.StatusBadRequest)
		return
	}

	tmpl, err := template.ParseFiles("templates/wechat.tmpl")
	if err != nil {
		log.Printf("parse WeChat template failed: %v", err)
		http.Error(w, "failed to load message template", http.StatusInternalServerError)
		return
	}
	var content bytes.Buffer
	if err := tmpl.Execute(&content, data); err != nil {
		log.Printf("render WeChat template failed: %v", err)
		http.Error(w, "failed to render alert message", http.StatusInternalServerError)
		return
	}
	message := wechatMessage{MsgType: "markdown"}
	message.Markdown.Content = content.String()
	wechatBody, err := json.Marshal(message)
	if err != nil {
		log.Printf("encode WeChat message failed: %v", err)
		http.Error(w, "failed to encode alert message", http.StatusInternalServerError)
		return
	}

	wechatURL, err := url.Parse(wechatEndpoint)
	if err != nil {
		http.Error(w, "failed to configure webhook", http.StatusInternalServerError)
		return
	}
	query := wechatURL.Query()
	query.Set("key", webhookID)
	wechatURL.RawQuery = query.Encode()

	var lastStatus int
	for attempt := 1; attempt <= 3; attempt++ {
		req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, wechatURL.String(), bytes.NewReader(wechatBody))
		if err != nil {
			log.Print("create WeChat request failed")
			break
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := wechatClient.Do(req)
		if err == nil {
			lastStatus = resp.StatusCode
			responseBody, readErr := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize+1))
			_ = resp.Body.Close()
			if readErr == nil && len(responseBody) <= maxResponseSize && resp.StatusCode >= 200 && resp.StatusCode < 300 {
				var result struct {
					ErrCode int `json:"errcode"`
				}
				if json.Unmarshal(responseBody, &result) == nil && result.ErrCode == 0 {
					respondOK(w)
					return
				}
			}
		}

		// Do not log the upstream error: the request URL contains the bot key.
		log.Printf("WeChat delivery failed (attempt %d/3)", attempt)
		if attempt < 3 {
			timer := time.NewTimer(time.Duration(attempt) * time.Second)
			select {
			case <-r.Context().Done():
				timer.Stop()
				http.Error(w, "request canceled", http.StatusRequestTimeout)
				return
			case <-timer.C:
			}
		}
	}

	if lastStatus != 0 {
		log.Printf("WeChat delivery failed after retries (HTTP status %d)", lastStatus)
	}
	http.Error(w, "failed to deliver alert to WeChat", http.StatusBadGateway)
}

func respondOK(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func HealthzHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "only GET supported", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodGet {
		_, _ = w.Write([]byte("ok"))
	}
}
