package telegram

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAuthenticatedWebhook(t *testing.T) {
	secret := strings.Repeat("b", 32)
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	handler := authenticatedWebhook(secret, next)

	t.Run("accepts valid Telegram header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/telegram/webhook", strings.NewReader(`{}`))
		req.Header.Set("X-Telegram-Bot-Api-Secret-Token", secret)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		if response.Code != http.StatusNoContent {
			t.Fatalf("unexpected status: %d", response.Code)
		}
	})

	t.Run("rejects invalid Telegram header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/telegram/webhook", strings.NewReader(`{}`))
		req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "wrong")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("unexpected status: %d", response.Code)
		}
	})

	t.Run("rejects non-POST requests", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/telegram/webhook", nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		if response.Code != http.StatusMethodNotAllowed {
			t.Fatalf("unexpected status: %d", response.Code)
		}
	})
}
