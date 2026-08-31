package handlers

import "testing"

func TestIsStartCommand(t *testing.T) {
	tests := []struct {
		text string
		want bool
	}{
		{text: "/start", want: true},
		{text: "/start payload", want: true},
		{text: "/start@example_bot", want: true},
		{text: "/starter", want: false},
		{text: "hello", want: false},
		{text: "", want: false},
	}

	for _, tt := range tests {
		if got := isStartCommand(tt.text); got != tt.want {
			t.Errorf("isStartCommand(%q) = %v, want %v", tt.text, got, tt.want)
		}
	}
}
