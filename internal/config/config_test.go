package config

import (
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	values := map[string]string{
		"TELEGRAM_BOT_TOKEN":      "123456:test-token",
		"TELEGRAM_WEBHOOK_SECRET": strings.Repeat("a", 32),
		"TELEGRAM_WEBHOOK_URL":    "https://bot.example.com/telegram/webhook",
	}

	cfg, err := load(func(key string) string { return values[key] })
	if err != nil {
		t.Fatalf("load returned an error: %v", err)
	}
	if cfg.ListenAddr != "127.0.0.1:18082" {
		t.Fatalf("unexpected listen address: %q", cfg.ListenAddr)
	}
	if cfg.WebhookPath != "/telegram/webhook" {
		t.Fatalf("unexpected webhook path: %q", cfg.WebhookPath)
	}
}

func TestLoadRejectsUnsafeValues(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{name: "short secret", key: "TELEGRAM_WEBHOOK_SECRET", value: "short"},
		{name: "non HTTPS URL", key: "TELEGRAM_WEBHOOK_URL", value: "http://bot.example.com/telegram/webhook"},
		{name: "non-loopback listener", key: "LISTEN_ADDR", value: "0.0.0.0:18082"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := map[string]string{
				"TELEGRAM_BOT_TOKEN":      "123456:test-token",
				"TELEGRAM_WEBHOOK_SECRET": strings.Repeat("a", 32),
				"TELEGRAM_WEBHOOK_URL":    "https://bot.example.com/telegram/webhook",
				"LISTEN_ADDR":             "127.0.0.1:18082",
			}
			values[tt.key] = tt.value
			if _, err := load(func(key string) string { return values[key] }); err == nil {
				t.Fatal("load accepted an unsafe value")
			}
		})
	}
}
