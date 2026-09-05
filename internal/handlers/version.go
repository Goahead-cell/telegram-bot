package handlers

import (
	"context"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Version shows the version embedded at build time.
func Version(ctx context.Context, b *bot.Bot, update *models.Update, version string) {
	sendText(ctx, b, update.Message.Chat.ID, versionText(version))
}

func versionText(version string) string {
	version = strings.TrimSpace(version)
	if version == "" {
		version = "dev"
	}

	return "Telegram Bot 版本：" + version
}
