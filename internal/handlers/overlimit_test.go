package handlers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadOverLimitStateAndReport(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "over-limit.json")
	data := `{
		"version": 99,
		"clear_generation": 42,
		"over_limit": ["User-B", "User-A", "User-B", "  "]
	}`
	if err := os.WriteFile(stateFile, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}

	state, err := loadOverLimitState(stateFile)
	if err != nil {
		t.Fatal(err)
	}
	report := overLimitReport(state)

	for _, expected := range []string{
		"Hysteria 流量超限名单",
		"超限用户：2",
		"1. User-A",
		"2. User-B",
	} {
		if !strings.Contains(report, expected) {
			t.Errorf("report does not contain %q:\n%s", expected, report)
		}
	}
}

func TestOverLimitReportWithoutUsers(t *testing.T) {
	report := overLimitReport(overLimitState{OverLimit: []string{}})
	if !strings.Contains(report, "当前没有流量超限用户") {
		t.Fatalf("unexpected empty report: %q", report)
	}
}

func TestLoadOverLimitStateRejectsInvalidData(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{name: "invalid JSON", data: `{"over_limit":[“User-A”]}`},
		{name: "missing field", data: `{"version":1}`},
		{name: "null field", data: `{"over_limit":null}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stateFile := filepath.Join(t.TempDir(), "over-limit.json")
			if err := os.WriteFile(stateFile, []byte(tt.data), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := loadOverLimitState(stateFile); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}
