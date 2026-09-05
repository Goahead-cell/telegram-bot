package handlers

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestStatusText(t *testing.T) {
	outputs := map[string]string{
		"nproc":             "2\n",
		"cat /proc/loadavg": "0.10 0.20 0.30 1/100 1234\n",
		"uptime -p":         "up 3 days\n",
		"free -h":           "Mem: 1.9Gi 500Mi 1.4Gi\n",
		"df -h /":           "/dev/vda1 40G 8G 32G 20% /\n",
	}

	run := func(_ context.Context, name string, args ...string) ([]byte, error) {
		key := strings.Join(append([]string{name}, args...), " ")
		return []byte(outputs[key]), nil
	}

	got := statusText(context.Background(), run)
	for _, want := range []string{"CPU 核心数：2", "0.10 0.20 0.30", "运行时间：up 3 days", "Mem: 1.9Gi", "/dev/vda1 40G"} {
		if !strings.Contains(got, want) {
			t.Errorf("statusText() does not contain %q:\n%s", want, got)
		}
	}
}

func TestServicesText(t *testing.T) {
	states := map[string]struct {
		output string
		err    error
	}{
		"caddy.service":           {output: "active\n"},
		"telegram-bot.service":    {output: "failed\n", err: errors.New("exit status 3")},
		"hysteria-server.service": {output: "inactive\n", err: errors.New("exit status 3")},
	}

	run := func(_ context.Context, name string, args ...string) ([]byte, error) {
		if name != "systemctl" || len(args) != 2 || args[0] != "is-active" {
			t.Fatalf("unexpected command: %s %v", name, args)
		}
		state := states[args[1]]
		return []byte(state.output), state.err
	}

	got := servicesText(context.Background(), run)
	for _, want := range []string{
		"🟢 Caddy：active",
		"🔴 Telegram Bot：failed",
		"⚪ Hysteria Server：inactive",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("servicesText() does not contain %q:\n%s", want, got)
		}
	}
}

func TestCommandOutputFailure(t *testing.T) {
	run := func(context.Context, string, ...string) ([]byte, error) {
		return nil, errors.New("command failed")
	}

	if got := commandOutput(context.Background(), run, "missing"); got != "获取失败" {
		t.Fatalf("commandOutput() = %q, want %q", got, "获取失败")
	}
}

func TestVersionText(t *testing.T) {
	if got := versionText("v0.3.0"); got != "Telegram Bot 版本：v0.3.0" {
		t.Fatalf("versionText() = %q", got)
	}
	if got := versionText(" "); got != "Telegram Bot 版本：dev" {
		t.Fatalf("versionText() fallback = %q", got)
	}
}
