package handlers

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Help handles the /help command.
func Help(ctx context.Context, b *bot.Bot, update *models.Update) {
	const text = `可用指令：
/start - 显示欢迎信息
/help - 显示指令列表
/status - CPU、内存、磁盘和运行时间
/version - 当前 Bot 版本
/services - Caddy、Bot 和 Hysteria 服务状态
/id - 当前 Telegram 用户 ID`

	sendText(ctx, b, update.Message.Chat.ID, text)
}
