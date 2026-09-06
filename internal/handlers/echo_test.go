package handlers

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestSplitText(t *testing.T) {
	text := "第一行\n第二行比较长\n第三行"
	parts := splitText(text, 8)
	if len(parts) < 2 {
		t.Fatalf("expected multiple parts, got %q", parts)
	}

	for _, part := range parts {
		if utf8.RuneCountInString(part) > 8 {
			t.Fatalf("part exceeds limit: %q", part)
		}
	}
	if strings.Join(parts, "") != strings.ReplaceAll(text, "\n", "") {
		t.Fatalf("split content changed: %q", parts)
	}
}

func TestSplitTextRejectsInvalidLimit(t *testing.T) {
	if parts := splitText("text", 0); parts != nil {
		t.Fatalf("expected nil parts, got %q", parts)
	}
}
