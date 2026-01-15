package main

import (
	"log"
	"net/http"
	"wechat-robot/api"
)

func main() {
	http.HandleFunc("/wechat-robot/send", api.WechatHandler)
	http.HandleFunc("/healthz", api.HealthzHandler)

	log.Println("Wechat-robot listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
