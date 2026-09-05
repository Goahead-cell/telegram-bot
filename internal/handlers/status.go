package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Status shows read-only VPS resource information.
func Status(ctx context.Context, b *bot.Bot, update *models.Update) {
	commandCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	sendText(ctx, b, update.Message.Chat.ID, statusText(commandCtx, executeCommand))
}

func statusText(ctx context.Context, run commandRunner) string {
	cpuCores := commandOutput(ctx, run, "nproc")
	load := loadAverage(ctx, run)
	uptime := commandOutput(ctx, run, "uptime", "-p")
	memory := commandOutput(ctx, run, "free", "-h")
	disk := commandOutput(ctx, run, "df", "-h", "/")

	return fmt.Sprintf(`服务器状态

CPU 核心数：%s
CPU/系统负载（1/5/15 分钟）：%s
运行时间：%s

内存：
%s

磁盘：
%s`, cpuCores, load, uptime, memory, disk)
}

func loadAverage(ctx context.Context, run commandRunner) string {
	text := commandOutput(ctx, run, "cat", "/proc/loadavg")
	fields := strings.Fields(text)
	if len(fields) < 3 {
		return text
	}
	return strings.Join(fields[:3], " ")
}
