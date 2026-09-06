package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const (
	maxReportedNodes = 100
	maxReportedUsers = 100
)

type trafficState struct {
	Version         int                    `json:"version"`
	Period          string                 `json:"period"`
	ClearGeneration int                    `json:"clear_generation"`
	Nodes           map[string]trafficNode `json:"nodes"`
}

type trafficNode struct {
	LastSeenUnix int64                   `json:"last_seen_unix"`
	Fault        string                  `json:"fault"`
	Users        map[string]trafficUsage `json:"users"`
}

type trafficUsage struct {
	RX uint64 `json:"rx"`
	TX uint64 `json:"tx"`
}

type namedTrafficUsage struct {
	Name  string
	Usage trafficUsage
}

// Traffic reports the Hysteria traffic state without running external commands.
func Traffic(ctx context.Context, b *bot.Bot, update *models.Update, stateFile string) {
	state, err := loadTrafficState(stateFile)
	if err != nil {
		log.Printf("read Hysteria traffic state %q: %v", stateFile, err)
		sendText(ctx, b, update.Message.Chat.ID, "读取 Hysteria 流量统计失败，请检查文件权限和 Bot 服务日志。")
		return
	}

	sendLongText(ctx, b, update.Message.Chat.ID, trafficReport(state))
}

func loadTrafficState(stateFile string) (trafficState, error) {
	data, err := readLocalJSONFile(stateFile)
	if err != nil {
		return trafficState{}, err
	}

	var state trafficState
	if err := json.Unmarshal(data, &state); err != nil {
		return trafficState{}, fmt.Errorf("decode state file: %w", err)
	}
	if state.Version < 1 {
		return trafficState{}, errors.New("state file version must be positive")
	}
	if strings.TrimSpace(state.Period) == "" {
		return trafficState{}, errors.New("state file period is empty")
	}
	if state.Nodes == nil {
		return trafficState{}, errors.New("state file nodes are missing")
	}
	return state, nil
}

func trafficReport(state trafficState) string {
	nodeNames := make([]string, 0, len(state.Nodes))
	users := make(map[string]trafficUsage)
	var total trafficUsage
	var latestSeen int64

	for nodeName, node := range state.Nodes {
		nodeNames = append(nodeNames, nodeName)
		if node.LastSeenUnix > latestSeen {
			latestSeen = node.LastSeenUnix
		}
		for userName, usage := range node.Users {
			users[userName] = addTrafficUsage(users[userName], usage)
			total = addTrafficUsage(total, usage)
		}
	}
	sort.Strings(nodeNames)

	userRows := make([]namedTrafficUsage, 0, len(users))
	for name, usage := range users {
		userRows = append(userRows, namedTrafficUsage{Name: name, Usage: usage})
	}
	sort.Slice(userRows, func(i, j int) bool {
		iTotal := trafficTotal(userRows[i].Usage)
		jTotal := trafficTotal(userRows[j].Usage)
		if iTotal == jTotal {
			return userRows[i].Name < userRows[j].Name
		}
		return iTotal > jTotal
	})

	var report strings.Builder
	fmt.Fprintln(&report, "Hysteria 流量统计")
	fmt.Fprintf(&report, "周期：%s\n", oneLine(state.Period, 32))
	fmt.Fprintf(&report, "清零代次：%d\n", state.ClearGeneration)
	fmt.Fprintf(&report, "节点：%d\n", len(state.Nodes))
	fmt.Fprintf(&report, "用户：%d\n", len(users))
	fmt.Fprintf(&report, "总 RX：%s\n", formatBytes(total.RX))
	fmt.Fprintf(&report, "总 TX：%s\n", formatBytes(total.TX))
	fmt.Fprintf(&report, "总流量：%s\n", formatBytes(trafficTotal(total)))
	if latestSeen > 0 {
		fmt.Fprintf(&report, "最近采集：%s\n", time.Unix(latestSeen, 0).UTC().Format("2006-01-02 15:04:05 UTC"))
	}

	fmt.Fprintln(&report, "\n节点：")
	if len(nodeNames) == 0 {
		fmt.Fprintln(&report, "暂无节点")
	}
	nodeReportCount := len(nodeNames)
	if nodeReportCount > maxReportedNodes {
		nodeReportCount = maxReportedNodes
	}
	for _, nodeName := range nodeNames[:nodeReportCount] {
		node := state.Nodes[nodeName]
		var nodeTotal trafficUsage
		for _, usage := range node.Users {
			nodeTotal = addTrafficUsage(nodeTotal, usage)
		}
		faultNote := ""
		if faultMarker(node.Fault) == "⚠️" {
			faultNote = "（故障：" + oneLine(node.Fault, 48) + "）"
		}
		fmt.Fprintf(&report, "%s %s%s：%d 用户，RX %s，TX %s，合计 %s\n",
			faultMarker(node.Fault), oneLine(nodeName, 64), faultNote, len(node.Users),
			formatBytes(nodeTotal.RX), formatBytes(nodeTotal.TX), formatBytes(trafficTotal(nodeTotal)))
	}
	if hidden := len(nodeNames) - nodeReportCount; hidden > 0 {
		fmt.Fprintf(&report, "另有 %d 个节点未显示。\n", hidden)
	}

	fmt.Fprintln(&report, "\n用户汇总：")
	if len(userRows) == 0 {
		fmt.Fprintln(&report, "暂无用户流量")
	}
	reportCount := len(userRows)
	if reportCount > maxReportedUsers {
		reportCount = maxReportedUsers
	}
	for _, row := range userRows[:reportCount] {
		fmt.Fprintf(&report, "%s：RX %s，TX %s，合计 %s\n",
			oneLine(row.Name, 64), formatBytes(row.Usage.RX), formatBytes(row.Usage.TX), formatBytes(trafficTotal(row.Usage)))
	}
	if hidden := len(userRows) - reportCount; hidden > 0 {
		fmt.Fprintf(&report, "另有 %d 个用户未显示。\n", hidden)
	}

	return strings.TrimSpace(report.String())
}

func addTrafficUsage(left, right trafficUsage) trafficUsage {
	return trafficUsage{
		RX: saturatingAdd(left.RX, right.RX),
		TX: saturatingAdd(left.TX, right.TX),
	}
}

func trafficTotal(usage trafficUsage) uint64 {
	return saturatingAdd(usage.RX, usage.TX)
}

func saturatingAdd(left, right uint64) uint64 {
	if math.MaxUint64-left < right {
		return math.MaxUint64
	}
	return left + right
}

func formatBytes(value uint64) string {
	const unit = uint64(1024)
	if value < unit {
		return fmt.Sprintf("%d B", value)
	}

	units := []string{"KiB", "MiB", "GiB", "TiB", "PiB", "EiB"}
	scaled := float64(value)
	unitIndex := -1
	for scaled >= float64(unit) && unitIndex < len(units)-1 {
		scaled /= float64(unit)
		unitIndex++
	}
	return fmt.Sprintf("%.2f %s", scaled, units[unitIndex])
}

func faultMarker(fault string) string {
	switch strings.ToLower(strings.TrimSpace(fault)) {
	case "no", "none", "ok":
		return "✅"
	case "":
		return "⚪"
	default:
		return "⚠️"
	}
}

func oneLine(value string, maxRunes int) string {
	if maxRunes < 1 {
		return ""
	}
	value = strings.Join(strings.Fields(value), " ")
	if utf8.RuneCountInString(value) <= maxRunes {
		return value
	}
	runes := []rune(value)
	return string(runes[:maxRunes-1]) + "…"
}
