package handlers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadTrafficStateAndReport(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "state.json")
	data := `{
		"version": 1,
		"period": "2026-09",
		"clear_generation": 2,
		"nodes": {
			"Node-B": {
				"last_seen_unix": 1788691022,
				"fault": "disk warning",
				"users": {
					"Alice": {"rx": 1024, "tx": 2048}
				}
			},
			"Node-A": {
				"last_seen_unix": 1788691000,
				"fault": "no",
				"users": {
					"Alice": {"rx": 1024, "tx": 0},
					"Bob": {"rx": 512, "tx": 512}
				}
			}
		}
	}`
	if err := os.WriteFile(stateFile, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}

	state, err := loadTrafficState(stateFile)
	if err != nil {
		t.Fatal(err)
	}
	report := trafficReport(state)

	for _, expected := range []string{
		"周期：2026-09",
		"清零代次：2",
		"节点：2",
		"用户：2",
		"总 RX：2.50 KiB",
		"总 TX：2.50 KiB",
		"总流量：5.00 KiB",
		"最近采集：2026-09-06 10:37:02 UTC",
		"✅ Node-A：2 用户",
		"⚠️ Node-B（故障：disk warning）：1 用户",
		"Alice：RX 2.00 KiB，TX 2.00 KiB，合计 4.00 KiB",
		"Bob：RX 512 B，TX 512 B，合计 1.00 KiB",
	} {
		if !strings.Contains(report, expected) {
			t.Errorf("report does not contain %q:\n%s", expected, report)
		}
	}

	if strings.Index(report, "Node-A") > strings.Index(report, "Node-B") {
		t.Errorf("nodes are not sorted by name:\n%s", report)
	}
	if strings.Index(report, "Alice：") > strings.Index(report, "Bob：") {
		t.Errorf("users are not sorted by total traffic:\n%s", report)
	}
}

func TestLoadTrafficStateRejectsInvalidData(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{name: "invalid JSON", data: `{`},
		{name: "missing version", data: `{"period":"2026-09","nodes":{}}`},
		{name: "missing period", data: `{"version":1,"nodes":{}}`},
		{name: "missing nodes", data: `{"version":1,"period":"2026-09"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stateFile := filepath.Join(t.TempDir(), "state.json")
			if err := os.WriteFile(stateFile, []byte(tt.data), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := loadTrafficState(stateFile); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func TestLoadTrafficStateRejectsOversizedFile(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "state.json")
	data := strings.Repeat("x", maxLocalJSONFileBytes+1)
	if err := os.WriteFile(stateFile, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadTrafficState(stateFile); err == nil {
		t.Fatal("expected an oversized-file error")
	}
}

func TestOneLine(t *testing.T) {
	if got := oneLine("line one\nline two", 12); got != "line one li…" {
		t.Fatalf("unexpected sanitized text: %q", got)
	}
}
