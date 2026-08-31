package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"telegram-webhook-bot/internal/config"
	"telegram-webhook-bot/internal/handlers"
	telegramserver "telegram-webhook-bot/internal/telegram"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	server, err := telegramserver.New(cfg, handlers.HandleUpdate)
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := server.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
