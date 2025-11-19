package main

import (
	"log"
	"net/http"
	"time"

	"kavenegar-sms/internal/alertmanager"
	"kavenegar-sms/internal/config"
	"kavenegar-sms/internal/kavenegar"
)

func main() {
	cfg, err := config.LoadFromEnv()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	smsClient := kavenegar.NewClient(cfg.KavenegarAPIKey, cfg.Sender, httpClient)

	handler := alertmanager.NewHandler(cfg, smsClient)

	mux := http.NewServeMux()
	mux.Handle("/kavenegar", handler)

	log.Printf("kavenegar-sms listening on %s", cfg.ListenAddr)

	if err := http.ListenAndServe(cfg.ListenAddr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
