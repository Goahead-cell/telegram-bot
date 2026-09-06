package handlers

import (
	"testing"

	"github.com/go-telegram/bot/models"
)

func TestCommandName(t *testing.T) {
	tests := []struct {
		text string
		want string
	}{
		{text: "/start", want: "start"},
		{text: "/start payload", want: "start"},
		{text: "/start@example_bot", want: "start"},
		{text: "/help", want: "help"},
		{text: "/help@example_bot", want: "help"},
		{text: "/status", want: "status"},
		{text: "/version", want: "version"},
		{text: "/services", want: "services"},
		{text: "/id", want: "id"},
		{text: "/traffic", want: "traffic"},
		{text: "/overlimit", want: "overlimit"},
		{text: "hello", want: ""},
		{text: "", want: ""},
	}

	for _, tt := range tests {
		if got := commandName(tt.text); got != tt.want {
			t.Errorf("commandName(%q) = %q, want %q", tt.text, got, tt.want)
		}
	}
}

func TestIsAdmin(t *testing.T) {
	const adminUserID int64 = 123456789

	tests := []struct {
		name   string
		update *models.Update
		want   bool
	}{
		{name: "nil update", update: nil, want: false},
		{name: "missing message", update: &models.Update{}, want: false},
		{name: "missing sender", update: &models.Update{Message: &models.Message{}}, want: false},
		{
			name: "different user",
			update: &models.Update{Message: &models.Message{
				From: &models.User{ID: 987654321},
			}},
			want: false,
		},
		{
			name: "administrator",
			update: &models.Update{Message: &models.Message{
				From: &models.User{ID: adminUserID},
			}},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isAdmin(tt.update, adminUserID); got != tt.want {
				t.Fatalf("isAdmin() = %v, want %v", got, tt.want)
			}
		})
	}
}
