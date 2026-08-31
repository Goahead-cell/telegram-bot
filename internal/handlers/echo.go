package handlers

import (
	"context"
	"log"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Echo is the fallback business handler for ordinary text messages.
func Echo(ctx context.Context, b *bot.Bot, update *models.Update) {
	sendText(ctx, b, update.Message.Chat.ID, update.Message.Text)
}

func sendText(ctx context.Context, b *bot.Bot, chatID int64, text string) {
	if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   text,
	}); err != nil {
		log.Printf("send Telegram message: %v", err)
	}
}
