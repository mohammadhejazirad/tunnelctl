//go:build linux

package tui

import (
	tea "charm.land/bubbletea/v2"
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"tunnelctl/internal/app"
	"tunnelctl/internal/domain"
)

func testModel(t *testing.T) model {
	t.Helper()
	root := t.TempDir()
	os.Chmod(root, 0700)
	s, e := app.New(root, "/missing/approved-core", time.Second)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Init(); e != nil {
		t.Fatal(e)
	}
	return model{service: s, ctx: context.Background(), page: menuPage, width: 80, height: 24, port: 2080, state: domain.Session{State: "idle"}}
}
func key(m model, s string) (model, tea.Cmd) {
	msg := tea.KeyPressMsg{Text: s, Code: []rune(s)[0]}
	switch s {
	case "enter":
		msg = tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		msg = tea.KeyPressMsg{Code: tea.KeyEscape}
	case "down":
		msg = tea.KeyPressMsg{Code: tea.KeyDown}
	case "up":
		msg = tea.KeyPressMsg{Code: tea.KeyUp}
	}
	next, cmd := m.Update(msg)
	return next.(model), cmd
}
func finish(t *testing.T, m model, cmd tea.Cmd) model {
	t.Helper()
	if cmd == nil {
		t.Fatal("missing asynchronous command")
	}
	next, _ := m.Update(cmd())
	m = next.(model)
	loaded, _ := m.Update(m.load())
	return loaded.(model)
}
func TestMenuHiddenImportProfileFlowsAndSettings(t *testing.T) {
	m := testModel(t)
	if !strings.Contains(m.View().Content, "Main menu") {
		t.Fatal("menu missing")
	}
	m.cursor = 2
	m, _ = key(m, "enter")
	if m.page != profilesPage {
		t.Fatal("menu navigation failed")
	}
	m, _ = key(m, "i")
	secret := "synthetic-menu-password"
	next, _ := m.Update(tea.PasteMsg{Content: "ss://aes-256-gcm:" + secret + "@127.0.0.1:8388#safe%1B%5D52%3Bc%3Binjection%07name"})
	m = next.(model)
	if strings.Contains(m.View().Content, secret) {
		t.Fatal("secret echoed by hidden form")
	}
	m, cmd := key(m, "enter")
	m = finish(t, m, cmd)
	if len(m.profiles) != 1 || m.form != nil || strings.Contains(m.View().Content, "injection") {
		t.Fatal("import/sanitization failed")
	}
	m, _ = key(m, "/")
	m, _ = key(m, "z")
	m, _ = key(m, "enter")
	if len(m.visibleProfiles()) != 0 {
		t.Fatal("filter failed")
	}
	m.filter = ""
	m, _ = key(m, "enter")
	if m.page != detailsPage {
		t.Fatal("details missing")
	}
	m, cmd = key(m, "enter")
	m = finish(t, m, cmd)
	if !m.profiles[0].Selected {
		t.Fatal("selection not persisted")
	}
	m, _ = key(m, "r")
	m.form.fields[0].value = "custom Unicode نام"
	m, cmd = key(m, "enter")
	m = finish(t, m, cmd)
	if m.profiles[0].Name != "custom Unicode نام" {
		t.Fatal("rename failed")
	}
	m.page = settingsPage
	m.cursor = 1
	m, _ = key(m, "enter")
	m.form.fields[0].value = "10m"
	m, _ = key(m, "enter")
	if m.duration != 10*time.Minute {
		t.Fatal("duration not configured")
	}
	m.page = detailsPage
	m.profile = m.profiles[0]
	m, _ = key(m, "d")
	m, cmd = key(m, "enter")
	m = finish(t, m, cmd)
	if len(m.profiles) != 0 {
		t.Fatal("confirmed removal failed")
	}
}
func TestSubscriptionMenuRealTLSPreviewCommitCancelAndRemove(t *testing.T) {
	m := testModel(t)
	blocked := make(chan struct{})
	feed := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ss://aes-256-gcm:synthetic-source@127.0.0.1:8388#source")
	}))
	defer feed.Close()
	cert, e := x509.ParseCertificate(feed.TLS.Certificates[0].Certificate[0])
	if e != nil {
		t.Fatal(e)
	}
	ca := filepath.Join(m.service.Store.Paths.Config, "ca.pem")
	os.WriteFile(ca, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}), 0600)
	m.page = sourcesPage
	m, _ = key(m, "a")
	m.form.fields[0].value = "menu-source"
	m.form.fields[1].value = feed.URL + "/?token=synthetic-source-secret"
	m.form.fields[2].value = "127.0.0.1/32"
	m.form.fields[3].value = ca
	m.form.cursor = 3
	if strings.Contains(m.View().Content, "synthetic-source-secret") {
		t.Fatal("source secret leaked")
	}
	m, cmd := key(m, "enter")
	m = finish(t, m, cmd)
	if len(m.sources) != 1 {
		t.Fatal("source not stored")
	}
	m, cmd = key(m, "p")
	m = finish(t, m, cmd)
	if m.preview == nil || len(m.profiles) != 0 {
		t.Fatal("preview mutated profiles")
	}
	m, cmd = key(m, "enter")
	m = finish(t, m, cmd)
	if len(m.profiles) != 1 {
		t.Fatal("commit failed")
	}
	before, _ := m.service.Store.Load()
	m, cmd = key(m, "p")
	m = finish(t, m, cmd)
	m, cmd = key(m, "enter")
	m = finish(t, m, cmd)
	after, _ := m.service.Store.Load()
	if len(after.Profiles) != 1 || after.Profiles[0].ID != before.Profiles[0].ID {
		t.Fatal("duplicate/stable ID failure")
	}
	// A controlled slow HTTPS source proves cancellation without blocking Update.
	slow := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { close(blocked); <-r.Context().Done() }))
	defer slow.Close()
	cert, _ = x509.ParseCertificate(slow.TLS.Certificates[0].Certificate[0])
	os.WriteFile(ca, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}), 0600)
	source, e := m.service.Subscriptions().Add("slow", strings.NewReader(slow.URL), []string{"127.0.0.1/32"}, ca)
	if e != nil {
		t.Fatal(e)
	}
	cmd = m.updateSource(source.ID)
	select {
	case <-blocked:
	case <-time.After(3 * time.Second):
		t.Fatal("fetch not started asynchronously")
	}
	m, _ = key(m, "esc")
	if !m.canceling || !m.busy {
		t.Fatal("cancellation must wait for cleanup")
	}
	m = finish(t, m, cmd)
	if m.busy {
		t.Fatal("cancellation never completed")
	}
	m.page = sourcesPage
	m.cursor = 0
	m, _ = key(m, "d")
	m, cmd = key(m, "enter")
	m = finish(t, m, cmd)
	if len(m.profiles) != 1 {
		t.Fatal("removing source lost profiles")
	}
}
func TestExitChecksRealSessionAndStaleCompletion(t *testing.T) {
	m := testModel(t)
	m, cmd := key(m, "q")
	msg := cmd().(completed)
	if !msg.quit || msg.state.State != "idle" {
		t.Fatal("idle exit blocks")
	}
	next, quit := m.Update(msg)
	if quit == nil || next.(model).busy {
		t.Fatal("idle exit failed")
	}
	m.opID = 9
	m.busy = true
	next, _ = m.Update(completed{id: 8, message: "stale"})
	if !next.(model).busy {
		t.Fatal("stale operation overwritten")
	}
	m.busy = false
	m.exitPrompt = true
	m.state = domain.Session{ID: "owned", State: "active", ProcessRunning: true}
	if !strings.Contains(m.View().Content, "Leave running and exit") {
		t.Fatal("detached exit options not explicit")
	}
}

func TestExitCancelsThenWaitsForOperationCleanup(t *testing.T) {
	m := testModel(t)
	cleaned := make(chan struct{})
	cmd := m.begin("Loading", func(ctx context.Context) completed { <-ctx.Done(); close(cleaned); return completed{err: ctx.Err()} })
	next, quit := key(m, "q")
	m = next
	if quit != nil || !m.exitAfter || !m.busy {
		t.Fatal("exit did not wait for cancellation")
	}
	completion := cmd()
	select {
	case <-cleaned:
	default:
		t.Fatal("operation cleanup not awaited")
	}
	value, check := m.Update(completion)
	m = value.(model)
	if check == nil || !m.busy {
		t.Fatal("exit failed to check real session after cancellation")
	}
	value, quit = m.Update(check())
	if quit == nil || value.(model).state.State != "idle" {
		t.Fatal("canceled idle exit did not finish")
	}
}
