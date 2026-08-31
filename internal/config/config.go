// Package config loads and validates runtime configuration.
package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
)

var webhookSecretPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{32,256}$`)

// Config contains everything needed by the Telegram transport layer.
type Config struct {
	BotToken      string
	WebhookSecret string
	WebhookURL    string
	WebhookPath   string
	ListenAddr    string
}

// Load reads configuration from environment variables.
func Load() (Config, error) {
	return load(os.Getenv)
}

func load(getenv func(string) string) (Config, error) {
	cfg := Config{
		BotToken:      strings.TrimSpace(getenv("TELEGRAM_BOT_TOKEN")),
		WebhookSecret: strings.TrimSpace(getenv("TELEGRAM_WEBHOOK_SECRET")),
		WebhookURL:    strings.TrimSpace(getenv("TELEGRAM_WEBHOOK_URL")),
		ListenAddr:    strings.TrimSpace(getenv("LISTEN_ADDR")),
	}

	if cfg.ListenAddr == "" {
		cfg.ListenAddr = "127.0.0.1:18082"
	}
	if cfg.BotToken == "" {
		return Config{}, errors.New("TELEGRAM_BOT_TOKEN is required")
	}
	if !webhookSecretPattern.MatchString(cfg.WebhookSecret) {
		return Config{}, errors.New("TELEGRAM_WEBHOOK_SECRET must be 32-256 characters using only A-Z, a-z, 0-9, _ and -")
	}

	webhookURL, err := url.ParseRequestURI(cfg.WebhookURL)
	if err != nil || webhookURL.Scheme != "https" || webhookURL.Host == "" || webhookURL.User != nil {
		return Config{}, errors.New("TELEGRAM_WEBHOOK_URL must be an absolute https URL")
	}
	if webhookURL.RawQuery != "" || webhookURL.Fragment != "" || webhookURL.RawPath != "" {
		return Config{}, errors.New("TELEGRAM_WEBHOOK_URL must not contain a query, fragment, or encoded path")
	}
	if webhookURL.Path == "" || webhookURL.Path == "/" || path.Clean(webhookURL.Path) != webhookURL.Path {
		return Config{}, errors.New("TELEGRAM_WEBHOOK_URL must contain a clean, non-root path")
	}
	cfg.WebhookPath = webhookURL.Path

	host, portText, err := net.SplitHostPort(cfg.ListenAddr)
	if err != nil {
		return Config{}, fmt.Errorf("LISTEN_ADDR must be an IP and port: %w", err)
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return Config{}, errors.New("LISTEN_ADDR must use a numeric loopback address")
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return Config{}, errors.New("LISTEN_ADDR port must be between 1 and 65535")
	}

	return cfg, nil
}
