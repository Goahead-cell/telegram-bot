package config

import (
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	values := map[string]string{
		"TELEGRAM_BOT_TOKEN":      "123456:test-token",
		"TELEGRAM_ADMIN_USER_ID":  "123456789",
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
	if cfg.AdminUserID != 123456789 {
		t.Fatalf("unexpected admin user ID: %d", cfg.AdminUserID)
	}
}

func TestLoadRejectsUnsafeValues(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{name: "missing token", key: "TELEGRAM_BOT_TOKEN", value: ""},
		{name: "missing admin user ID", key: "TELEGRAM_ADMIN_USER_ID", value: ""},
		{name: "non-numeric admin user ID", key: "TELEGRAM_ADMIN_USER_ID", value: "not-a-number"},
		{name: "negative admin user ID", key: "TELEGRAM_ADMIN_USER_ID", value: "-1"},
		{name: "short secret", key: "TELEGRAM_WEBHOOK_SECRET", value: "short"},
		{name: "secret containing punctuation", key: "TELEGRAM_WEBHOOK_SECRET", value: strings.Repeat("a", 31) + "!"},
		{name: "non HTTPS URL", key: "TELEGRAM_WEBHOOK_URL", value: "http://bot.example.com/telegram/webhook"},
		{name: "root webhook path", key: "TELEGRAM_WEBHOOK_URL", value: "https://bot.example.com/"},
		{name: "webhook URL query", key: "TELEGRAM_WEBHOOK_URL", value: "https://bot.example.com/telegram/webhook?debug=1"},
		{name: "webhook URL fragment", key: "TELEGRAM_WEBHOOK_URL", value: "https://bot.example.com/telegram/webhook#fragment"},
		{name: "webhook URL credentials", key: "TELEGRAM_WEBHOOK_URL", value: "https://user:password@bot.example.com/telegram/webhook"},
		{name: "non-loopback listener", key: "LISTEN_ADDR", value: "0.0.0.0:18082"},
		{name: "listener without port", key: "LISTEN_ADDR", value: "127.0.0.1"},
		{name: "invalid listener port", key: "LISTEN_ADDR", value: "127.0.0.1:70000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := map[string]string{
				"TELEGRAM_BOT_TOKEN":      "123456:test-token",
				"TELEGRAM_ADMIN_USER_ID":  "123456789",
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
