// Package handlers contains only the bot's business behavior.
package handlers

import (
	"context"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// HandleUpdate is the callback passed to go-telegram/bot.
// Add new commands here, then put their implementation in separate files.
func HandleUpdate(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update == nil || update.Message == nil || update.Message.Text == "" {
		return
	}

	if isStartCommand(update.Message.Text) {
		Start(ctx, b, update)
		return
	}

	Echo(ctx, b, update)
}

func isStartCommand(text string) bool {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return false
	}
	return fields[0] == "/start" || strings.HasPrefix(fields[0], "/start@")
}
