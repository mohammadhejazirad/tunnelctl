//go:build linux

package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"tunnelctl/internal/app"
	"tunnelctl/internal/domain"
)

func TestSettingsDraftSaveCancelReloadAndConflict(t *testing.T) {
	m := testModel(t)
	m.settingForm("port")
	m.form.fields[0].value = "9071"
	m.form.fields[1].value = "10m"
	before, e := m.service.Settings()
	if e != nil {
		t.Fatal(e)
	}
	m, _ = key(m, "esc")
	after, e := m.service.Settings()
	if e != nil || before != after {
		t.Fatal("canceled draft was saved")
	}
	m.settingForm("port")
	m.form.fields[0].value = "9071"
	m.form.fields[1].value = "10m"
	m.form.fields[6].value = "false"
	m.form.cursor = 7
	m, cmd := key(m, "enter")
	m = finish(t, m, cmd)
	if m.port != 9071 || m.duration != 10*time.Minute || m.color || !strings.Contains(m.message, "Success:") {
		t.Fatal("durable settings not applied after Save")
	}
	restarted, e := app.New(filepath.Dir(m.service.Paths().Config), "/missing/core", time.Second)
	if e != nil {
		t.Fatal(e)
	}
	saved, e := restarted.Settings()
	if e != nil || saved.Preferences.ListenerPort != 9071 || saved.Preferences.Duration != "10m" {
		t.Fatal("restart lost form settings")
	}
	// A CLI field patch uses the exact same service API/storage as the form.
	_, e = restarted.EditSettings(nil, func(p *domain.Preferences) error { p.ListenerPort = 9072; return nil })
	if e != nil {
		t.Fatal(e)
	}
	next, _ := m.Update(m.load())
	m = next.(model)
	if m.port != 9072 {
		t.Fatal("external saved settings not reloaded")
	}
	m.settingForm("port")
	m.form.fields[0].value = "9073"
	m.form.cursor = 7
	_, e = restarted.EditSettings(nil, func(p *domain.Preferences) error { p.Color = true; return nil })
	if e != nil {
		t.Fatal(e)
	}
	m, cmd = key(m, "enter")
	m = finish(t, m, cmd)
	saved, e = restarted.Settings()
	if e != nil || saved.Preferences.ListenerPort != 9072 || !saved.Preferences.Color || strings.Contains(m.message, "Success:") {
		t.Fatal("stale form overwrote concurrent change or claimed success")
	}
	state, e := restarted.Status()
	if e != nil || state.ProcessRunning || state.ListenerReady {
		t.Fatal("settings caused auto-connect")
	}
}

func TestSettingsWriteFailureAndInvalidFormNeverClaimSuccess(t *testing.T) {
	m := testModel(t)
	m.settingForm("port")
	m.form.fields[0].value = "80"
	m.form.cursor = 7
	m, cmd := key(m, "enter")
	if cmd != nil || !strings.Contains(m.message, "Error:") {
		t.Fatal("invalid draft accepted")
	}
	m.settingForm("port")
	m.form.fields[0].value = "9071"
	m.form.cursor = 7
	path := filepath.Join(m.service.Paths().Config, "profiles.json")
	if e := os.Chmod(path, 0644); e != nil {
		t.Fatal(e)
	}
	defer os.Chmod(path, 0600)
	m, cmd = key(m, "enter")
	m = finish(t, m, cmd)
	if strings.Contains(m.message, "Success:") || m.port == 9071 {
		t.Fatal("failed save claimed success or applied draft")
	}
}

func TestMetadataFormSaveDiscardAndStaleRevision(t *testing.T) {
	m := testModel(t)
	p, e := m.service.Import(strings.NewReader("ss://aes-256-gcm:synthetic-metadata@127.0.0.1:8388#metadata"))
	if e != nil {
		t.Fatal(e)
	}
	m.profile = p
	m.page = detailsPage
	m.cursor = 4
	m, _ = key(m, "enter")
	m.form.fields[0].value = "work,personal"
	m.form.fields[1].value = "true"
	m, _ = key(m, "esc")
	p, e = m.service.Show(p.ID)
	if e != nil || p.Favorite || len(p.Tags) != 0 {
		t.Fatal("metadata discard wrote data")
	}
	m.cursor = 4
	m, _ = key(m, "enter")
	m.form.fields[0].value = "work,personal"
	m.form.fields[1].value = "true"
	m.form.cursor = 2
	m, cmd := key(m, "enter")
	m = finish(t, m, cmd)
	p, e = m.service.Show(p.ID)
	if e != nil || !p.Favorite || len(p.Tags) != 2 {
		t.Fatal("metadata form not durably saved")
	}
	m.cursor = 4
	m, _ = key(m, "enter")
	m.form.fields[0].value = "stale"
	m.form.cursor = 2
	if _, e = m.service.RenameProfile(p.ID, "external rename"); e != nil {
		t.Fatal(e)
	}
	m, cmd = key(m, "enter")
	m = finish(t, m, cmd)
	p, e = m.service.Show(p.ID)
	if e != nil || p.Name != "external rename" || p.Tags[0] != "work" || strings.Contains(m.message, "Success:") {
		t.Fatal("stale metadata Save overwrote data")
	}
}
