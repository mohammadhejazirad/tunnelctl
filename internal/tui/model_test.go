package tui

import (
	tea "charm.land/bubbletea/v2"
	"strings"
	"testing"
	"tunnelctl/internal/domain"
)

func TestCompactViewsAndSanitizedNames(t *testing.T) {
	m := model{width: 40, height: 10, page: 1, profiles: []domain.PublicProfile{{Name: domain.SafeName("safe\x1b]52;c;injected\x07name"), Method: "aes-256-gcm"}}}
	view := m.View()
	if strings.Contains(view.Content, "injected") || strings.Contains(view.Content, "\x1b") {
		t.Fatal("terminal content injection")
	}
	if len(strings.Split(view.Content, "\n")) > 10 {
		t.Fatal("view exceeds resized terminal")
	}
	next, _ := m.Update(tea.WindowSizeMsg{Width: 12, Height: 3})
	if !strings.Contains(next.View().Content, "too small") {
		t.Fatal("small terminal fallback missing")
	}
}
