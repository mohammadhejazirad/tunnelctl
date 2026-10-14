package tui

import (
	tea "charm.land/bubbletea/v2"
	"strings"
	"testing"
	"tunnelctl/internal/domain"
)

func TestCompactViewsAndSanitizedNames(t *testing.T) {
	m := model{width: 40, height: 10, page: 1, profiles: []domain.PublicProfile{{Name: "safe\x1b]52;c;injected\x07name", Method: "aes-256-gcm"}}}
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

func TestErrorRemedyRemainsVisibleWhenListIsFull(t *testing.T) {
	m := model{width: 80, height: 12, page: profilesPage, message: safeError(domain.Fail(5, "The request failed.", "Verify the TLS certificate and source policy; never disable TLS verification."))}
	for i := 0; i < 100; i++ {
		m.profiles = append(m.profiles, domain.PublicProfile{Name: "synthetic profile"})
	}
	view := m.View().Content
	if !strings.Contains(view, "Error:") || !strings.Contains(view, "Remedy:") || !strings.Contains(view, "never disable TLS") || len(strings.Split(view, "\n")) > 12 {
		t.Fatal("error/remedy hidden by list overflow")
	}
}

func TestEveryCompactSizeBoundsMessages(t *testing.T) {
	for height := 5; height <= 24; height++ {
		for _, width := range []int{16, 20, 40, 80} {
			m := model{width: width, height: height, message: strings.Repeat("synthetic long message ", 50), page: menuPage}
			if len(strings.Split(m.View().Content, "\n")) > height {
				t.Fatal("compact view overflow")
			}
		}
	}
}
