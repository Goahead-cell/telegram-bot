package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const maxReportedOverLimitUsers = 200

type overLimitState struct {
	OverLimit []string `json:"over_limit"`
}

// OverLimit reports users recorded in the local Hysteria over-limit file.
func OverLimit(ctx context.Context, b *bot.Bot, update *models.Update, stateFile string) {
	state, err := loadOverLimitState(stateFile)
	if err != nil {
		log.Printf("read Hysteria over-limit state %q: %v", stateFile, err)
		sendText(ctx, b, update.Message.Chat.ID, "读取 Hysteria 流量超限名单失败，请检查 JSON 格式、文件权限和 Bot 服务日志。")
		return
	}

	sendLongText(ctx, b, update.Message.Chat.ID, overLimitReport(state))
}

func loadOverLimitState(stateFile string) (overLimitState, error) {
	data, err := readLocalJSONFile(stateFile)
	if err != nil {
		return overLimitState{}, err
	}

	var state overLimitState
	if err := json.Unmarshal(data, &state); err != nil {
		return overLimitState{}, fmt.Errorf("decode over-limit file: %w", err)
	}
	if state.OverLimit == nil {
		return overLimitState{}, errors.New("over_limit field is missing or null")
	}
	return state, nil
}

func overLimitReport(state overLimitState) string {
	uniqueUsers := make(map[string]struct{}, len(state.OverLimit))
	for _, name := range state.OverLimit {
		name = strings.TrimSpace(name)
		if name != "" {
			uniqueUsers[name] = struct{}{}
		}
	}

	users := make([]string, 0, len(uniqueUsers))
	for name := range uniqueUsers {
		users = append(users, name)
	}
	sort.Strings(users)

	if len(users) == 0 {
		return "Hysteria 流量超限名单\n\n当前没有流量超限用户。"
	}

	var report strings.Builder
	fmt.Fprintln(&report, "Hysteria 流量超限名单")
	fmt.Fprintf(&report, "超限用户：%d\n\n", len(users))

	reportCount := len(users)
	if reportCount > maxReportedOverLimitUsers {
		reportCount = maxReportedOverLimitUsers
	}
	for index, name := range users[:reportCount] {
		fmt.Fprintf(&report, "%d. %s\n", index+1, oneLine(name, 128))
	}
	if hidden := len(users) - reportCount; hidden > 0 {
		fmt.Fprintf(&report, "另有 %d 个超限用户未显示。\n", hidden)
	}

	return strings.TrimSpace(report.String())
}
