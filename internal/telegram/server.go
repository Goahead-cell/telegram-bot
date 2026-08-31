// Package telegram owns the go-telegram/bot and HTTP webhook integration.
// It deliberately contains no business behavior.
package telegram

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/go-telegram/bot"

	"telegram-webhook-bot/internal/config"
)

const maxWebhookBodyBytes = 1 << 20

// Server connects Caddy's local upstream to Telegram's webhook API.
type Server struct {
	cfg        config.Config
	bot        *bot.Bot
	httpServer *http.Server
}

// New creates the transport layer and accepts business behavior as a callback.
func New(cfg config.Config, handler bot.HandlerFunc) (*Server, error) {
	b, err := bot.New(
		cfg.BotToken,
		bot.WithDefaultHandler(handler),
		bot.WithWebhookSecretToken(cfg.WebhookSecret),
		bot.WithWorkers(4),
	)
	if err != nil {
		return nil, fmt.Errorf("initialize Telegram bot: %w", err)
	}

	mux := http.NewServeMux()
	mux.Handle(cfg.WebhookPath, authenticatedWebhook(cfg.WebhookSecret, b.WebhookHandler()))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})

	return &Server{
		cfg: cfg,
		bot: b,
		httpServer: &http.Server{
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       10 * time.Second,
			WriteTimeout:      10 * time.Second,
			IdleTimeout:       30 * time.Second,
		},
	}, nil
}

// Run serves webhook requests until ctx is cancelled.
func (s *Server) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	listener, err := net.Listen("tcp", s.cfg.ListenAddr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", s.cfg.ListenAddr, err)
	}
	defer listener.Close()

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- s.httpServer.Serve(listener)
	}()
	go s.bot.StartWebhook(ctx)

	registerCtx, registerCancel := context.WithTimeout(ctx, 15*time.Second)
	_, err = s.bot.SetWebhook(registerCtx, &bot.SetWebhookParams{
		URL:            s.cfg.WebhookURL,
		SecretToken:    s.cfg.WebhookSecret,
		MaxConnections: 10,
		AllowedUpdates: []string{"message"},
	})
	registerCancel()
	if err != nil {
		cancel()
		_ = s.httpServer.Close()
		return fmt.Errorf("register Telegram webhook: %w", err)
	}

	log.Printf("Telegram webhook registered at %s; listening on %s", s.cfg.WebhookURL, s.cfg.ListenAddr)

	select {
	case <-ctx.Done():
	case err = <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			cancel()
			return fmt.Errorf("serve webhook: %w", err)
		}
	}

	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := s.httpServer.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("shut down webhook server: %w", err)
	}

	return nil
}

func authenticatedWebhook(secret string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		provided := r.Header.Get("X-Telegram-Bot-Api-Secret-Token")
		if len(provided) != len(secret) || subtle.ConstantTimeCompare([]byte(provided), []byte(secret)) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, maxWebhookBodyBytes)
		next.ServeHTTP(w, r)
	})
}
