package main

import (
	"log"
	"net/http"
	"time"
	"wechat-robot/api"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/wechat-robot/send", api.WechatHandler)
	mux.HandleFunc("/healthz", api.HealthzHandler)

	log.Println("Wechat-robot listening on :8080")
	server := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      25 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Fatal(server.ListenAndServe())
}
