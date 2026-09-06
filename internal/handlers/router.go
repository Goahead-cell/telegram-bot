// Package handlers contains only the bot's business behavior.
package handlers

import (
	"context"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Router dispatches updates received from the configured administrator.
type Router struct {
	adminUserID      int64
	version          string
	trafficStateFile string
	overLimitFile    string
}

// NewRouter creates a Router restricted to one Telegram user.
func NewRouter(adminUserID int64, version, trafficStateFile, overLimitFile string) *Router {
	return &Router{
		adminUserID:      adminUserID,
		version:          version,
		trafficStateFile: trafficStateFile,
		overLimitFile:    overLimitFile,
	}
}

// HandleUpdate is the callback passed to go-telegram/bot.
// Add new commands here, then put their implementation in separate files.
func (r *Router) HandleUpdate(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update == nil || update.Message == nil || update.Message.Text == "" {
		return
	}
	if !isAdmin(update, r.adminUserID) {
		return
	}

	switch commandName(update.Message.Text) {
	case "start":
		Start(ctx, b, update)
	case "help":
		Help(ctx, b, update)
	case "status":
		Status(ctx, b, update)
	case "version":
		Version(ctx, b, update, r.version)
	case "services":
		Services(ctx, b, update)
	case "id":
		ID(ctx, b, update)
	case "traffic":
		Traffic(ctx, b, update, r.trafficStateFile)
	case "overlimit":
		OverLimit(ctx, b, update, r.overLimitFile)
	default:
		Echo(ctx, b, update)
	}
}

func isAdmin(update *models.Update, adminUserID int64) bool {
	return update != nil &&
		update.Message != nil &&
		update.Message.From != nil &&
		update.Message.From.ID == adminUserID
}

func commandName(text string) string {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return ""
	}

	first := fields[0]
	if !strings.HasPrefix(first, "/") {
		return ""
	}

	// /help@my_bot 转换成 /help
	command := strings.SplitN(first, "@", 2)[0]

	// 去掉开头的 /
	return strings.TrimPrefix(command, "/")
}
