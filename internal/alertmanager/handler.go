package alertmanager

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"kavenegar-sms/internal/config"
	"kavenegar-sms/internal/kavenegar"
)

type Handler struct {
	cfg     config.Config
	sms     *kavenegar.Client
	timeout time.Duration
}

func NewHandler(cfg config.Config, smsClient *kavenegar.Client) *Handler {
	return &Handler{
		cfg:     cfg,
		sms:     smsClient,
		timeout: 5 * time.Second,
	}
}

// ServeHTTP makes Handler implement http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("alertmanager: error reading request body: %v", err)
		http.Error(w, "cannot read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var payload Webhook
	if err := json.Unmarshal(body, &payload); err != nil {
		log.Printf("alertmanager: error unmarshaling JSON: %v", err)
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if len(payload.Alerts) == 0 {
		log.Println("alertmanager: received webhook with no alerts")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"msg":"no alerts"}`))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()

	for _, a := range payload.Alerts {
		if !shouldNotify(a) {
			continue
		}

		msg := buildSMSMessage(a)

		if err := h.sms.SendBulkSMS(ctx, h.cfg.Receivers, msg); err != nil {
			log.Printf(
				"alertmanager: failed to send SMS for alert=%q severity=%q instance=%q: %v",
				a.Labels["alertname"],
				a.Labels["severity"],
				a.Labels["instance"],
				err,
			)
		} else {
			log.Printf(
				"alertmanager: SMS sent for alert=%q severity=%q instance=%q to=%v",
				a.Labels["alertname"],
				a.Labels["severity"],
				a.Labels["instance"],
				h.cfg.Receivers,
			)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"msg":"alerts processed"}`))
}

func shouldNotify(a Alert) bool {
	severity := a.Labels["severity"]
	if severity != "critical" {
		return false
	}
	if a.Status != "" && a.Status != "firing" {
		return false
	}
	return true
}

func buildSMSMessage(a Alert) string {
	alertName := a.Labels["alertname"]
	instance := a.Labels["instance"]
	severity := a.Labels["severity"]

	description := a.Annotations["description"]
	if description == "" {
		description = a.Annotations["summary"]
	}
	if description == "" {
		description = "No description"
	}

	base := "ALERT " + orDefault(alertName, "unknown") +
		" [" + orDefault(severity, "n/a") + "]\n" +
		"Instance: " + orDefault(instance, "n/a") + "\n" +
		description

	const maxRunes = 500
	runes := []rune(base)
	if len(runes) > maxRunes {
		return string(runes[:maxRunes-3]) + "..."
	}
	return base
}

func orDefault(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}
