package handlers

import (
	"context"
	"strconv"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// ID shows the Telegram user ID of the message sender.
func ID(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message.From == nil {
		return
	}

	userID := strconv.FormatInt(update.Message.From.ID, 10)
	sendText(ctx, b, update.Message.Chat.ID, "你的 Telegram 用户 ID："+userID)
}
