package handlers

import (
	"context"
	"os/exec"
	"strings"
)

type commandRunner func(ctx context.Context, name string, args ...string) ([]byte, error)

func executeCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

func commandOutput(ctx context.Context, run commandRunner, name string, args ...string) string {
	output, err := run(ctx, name, args...)
	text := strings.TrimSpace(string(output))
	if text != "" {
		return text
	}
	if err != nil {
		return "获取失败"
	}
	return "无数据"
}
