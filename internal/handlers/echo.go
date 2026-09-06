package handlers

import (
	"context"
	"log"
	"strings"

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

func sendLongText(ctx context.Context, b *bot.Bot, chatID int64, text string) {
	for _, part := range splitText(text, 3500) {
		sendText(ctx, b, chatID, part)
	}
}

func splitText(text string, maxRunes int) []string {
	if maxRunes < 1 {
		return nil
	}

	remaining := []rune(strings.TrimSpace(text))
	parts := make([]string, 0, len(remaining)/maxRunes+1)
	for len(remaining) > maxRunes {
		splitAt := maxRunes
		for index := maxRunes; index > 0; index-- {
			if remaining[index-1] == '\n' {
				splitAt = index
				break
			}
		}

		part := strings.TrimSpace(string(remaining[:splitAt]))
		if part != "" {
			parts = append(parts, part)
		}
		remaining = remaining[splitAt:]
	}

	if part := strings.TrimSpace(string(remaining)); part != "" {
		parts = append(parts, part)
	}
	return parts
}
