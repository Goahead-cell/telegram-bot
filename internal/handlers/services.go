package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type monitoredService struct {
	name string
	unit string
}

var monitoredServices = []monitoredService{
	{name: "Caddy", unit: "caddy.service"},
	{name: "Telegram Bot", unit: "telegram-bot.service"},
	{name: "Hysteria Server", unit: "hysteria-server.service"},
}

// Services shows whether the configured systemd services are active.
func Services(ctx context.Context, b *bot.Bot, update *models.Update) {
	commandCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	sendText(ctx, b, update.Message.Chat.ID, servicesText(commandCtx, executeCommand))
}

func servicesText(ctx context.Context, run commandRunner) string {
	lines := []string{"服务状态"}
	for _, service := range monitoredServices {
		output, err := run(ctx, "systemctl", "is-active", service.unit)
		state := strings.TrimSpace(string(output))
		if state == "" && err != nil {
			state = "查询失败"
		} else if state == "" {
			state = "未知"
		}

		lines = append(lines, fmt.Sprintf("%s %s：%s", serviceIcon(state), service.name, state))
	}

	return strings.Join(lines, "\n")
}

func serviceIcon(state string) string {
	switch state {
	case "active":
		return "🟢"
	case "activating", "reloading", "deactivating":
		return "🟡"
	case "failed":
		return "🔴"
	default:
		return "⚪"
	}
}
