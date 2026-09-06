package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Goahead-cell/telegram-bot/internal/config"
	"github.com/Goahead-cell/telegram-bot/internal/handlers"
	telegramserver "github.com/Goahead-cell/telegram-bot/internal/telegram"
)

var version = "dev"

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--version" {
		fmt.Println(version)
		return
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	router := handlers.NewRouter(cfg.AdminUserID, version, cfg.TrafficStateFile, cfg.OverLimitFile)
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
