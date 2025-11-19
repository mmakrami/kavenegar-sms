package config

import (
	"errors"
	"os"
	"strings"
)

type Config struct {
	KavenegarAPIKey string
	Sender          string
	Receivers       []string
	ListenAddr      string
}

func LoadFromEnv() (Config, error) {
	cfg := Config{
		KavenegarAPIKey: strings.TrimSpace(os.Getenv("KAVENEGAR_API_KEY")),
		Sender:          strings.TrimSpace(os.Getenv("KAVENEGAR_SENDER")),
		ListenAddr:      strings.TrimSpace(os.Getenv("LISTEN_ADDR")),
	}

	receiversRaw := strings.TrimSpace(os.Getenv("KAVENEGAR_RECEIVERS"))
	if receiversRaw != "" {
		parts := strings.Split(receiversRaw, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				cfg.Receivers = append(cfg.Receivers, p)
			}
		}
	}

	if cfg.ListenAddr == "" {
		cfg.ListenAddr = ":8082"
	}

	if cfg.KavenegarAPIKey == "" {
		return cfg, errors.New("KAVENEGAR_API_KEY is required")
	}
	if cfg.Sender == "" {
		return cfg, errors.New("KAVENEGAR_SENDER is required")
	}
	if len(cfg.Receivers) == 0 {
		return cfg, errors.New("KAVENEGAR_RECEIVERS must contain at least one number")
	}

	return cfg, nil
}
