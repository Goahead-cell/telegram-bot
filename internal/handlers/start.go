package handlers

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Start handles the /start command.
func Start(ctx context.Context, b *bot.Bot, update *models.Update) {
	sendText(ctx, b, update.Message.Chat.ID, "机器人已上线。发送 /help 查看可用指令。")
}
