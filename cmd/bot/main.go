package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Goahead-cell/telegram-bot/internal/config"
	"github.com/Goahead-cell/telegram-bot/internal/handlers"
	telegramserver "github.com/Goahead-cell/telegram-bot/internal/telegram"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	router := handlers.NewRouter(cfg.AdminUserID)
	server, err := telegramserver.New(cfg, router.HandleUpdate)
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := server.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
