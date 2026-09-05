package handlers

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Help handles the /help command.
func Help(ctx context.Context, b *bot.Bot, update *models.Update) {
	sendText(ctx, b, update.Message.Chat.ID, "Help函数OK")
}
