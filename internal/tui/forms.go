package tui

import (
	tea "charm.land/bubbletea/v2"
	"context"
	"strconv"
	"strings"
	"tunnelctl/internal/app"
	"tunnelctl/internal/domain"
)

type field struct {
	label, value string
	hidden       bool
	limit        int
}
type form struct {
	action, title, key string
	fields             []field
	cursor             int
	revision           uint64
	draft              domain.Preferences
}

func (f *form) append(text string) {
	field := &f.fields[f.cursor]
	if len(field.value)+len(text) <= field.limit {
		field.value += text
	}
}
func (m *model) openForm(action, title string, fields ...field) {
	m.form = &form{action: action, title: title, fields: fields}
	m.message = ""
}
func (m *model) addSourceForm() {
	m.openForm("add-source", "Add source; no download until manual preview", field{"Name", "", false, 256}, field{"HTTPS URL", "", true, 16384}, field{"Private CIDRs (optional, comma separated)", "", false, 512}, field{"Private CA file (optional)", "", false, 4096})
}
func (m *model) settingForm(setting string) {
	v, e := m.service.Settings()
	if e != nil {
		m.message = safeError(e)
		return
	}
	p := v.Preferences
	m.openForm("settings", "Persistent defaults; Save commits all fields; active session unchanged",
		field{"Loopback port", strconv.Itoa(int(p.ListenerPort)), false, 5},
		field{"Duration (0s = no deadline)", p.Duration, false, 32},
		field{"Private authentication file (absolute)", p.AuthFile, false, 4096},
		field{"Explicit HTTPS probe destination", p.ProbeTarget, true, 4096},
		field{"Core path (absolute; blank = environment/default)", p.CorePath, false, 4096},
		field{"Core timeout", p.Timeout, false, 32},
		field{"Color (true/false)", strconv.FormatBool(p.Color), false, 5},
		field{"Save", "", false, 0})
	m.form.revision = v.Revision
	m.form.draft = p
	if i, ok := map[string]int{"port": 0, "duration": 1, "auth": 2, "probe": 3}[setting]; ok {
		m.form.cursor = i
	}
}
func (m *model) renameForm() {
	if m.page == profilesPage {
		p := m.visibleProfiles()
		if m.cursor >= len(p) {
			return
		}
		m.profile = p[m.cursor]
	}
	if m.page == sourcesPage {
		if m.cursor >= len(m.sources) {
			return
		}
		m.source = m.sources[m.cursor]
	}
	switch m.page {
	case profilesPage, detailsPage:
		m.openForm("rename-profile", "Rename profile; preserved on subscription refresh", field{"Name", m.profile.Name, false, 256})
		m.form.key = m.profile.ID
	case sourcesPage, sourceDetailsPage:
		m.openForm("rename-source", "Rename subscription", field{"Name", m.source.Name, false, 256})
		m.form.key = m.source.ID
	}
}
func (m *model) removeConfirmation() {
	if m.page == profilesPage {
		p := m.visibleProfiles()
		if m.cursor >= len(p) {
			return
		}
		m.profile = p[m.cursor]
	}
	if m.page == sourcesPage {
		if m.cursor >= len(m.sources) {
			return
		}
		m.source = m.sources[m.cursor]
	}
	switch m.page {
	case profilesPage, detailsPage:
		id := m.profile.ID
		m.confirm("Remove this profile? Active profiles cannot be removed.", func(context.Context) completed {
			return completed{message: "Profile removed.", err: m.service.RemoveProfile(id)}
		})
	case sourcesPage, sourceDetailsPage:
		id := m.source.ID
		m.confirm("Remove source? Its profiles remain as standalone; the active session continues.", func(context.Context) completed {
			return completed{message: "Source removed; profiles retained.", err: m.service.Subscriptions().Remove(id)}
		})
	}
}
func (m *model) formKey(msg tea.KeyPressMsg) tea.Cmd {
	key := msg.String()
	f := m.form
	switch key {
	case "esc", "ctrl+c":
		m.form = nil
		m.message = "Canceled; secret input discarded."
		return nil
	case "tab", "down":
		f.cursor = (f.cursor + 1) % len(f.fields)
	case "shift+tab", "up":
		f.cursor = (f.cursor + len(f.fields) - 1) % len(f.fields)
	case "backspace":
		r := []rune(f.fields[f.cursor].value)
		if len(r) > 0 {
			f.fields[f.cursor].value = string(r[:len(r)-1])
		}
	case "ctrl+u":
		f.fields[f.cursor].value = ""
	case "enter":
		if f.cursor < len(f.fields)-1 {
			f.cursor++
			return nil
		}
		return m.submit()
	default:
		if msg.Text != "" {
			f.append(msg.Text)
		}
	}
	return nil
}
func (m *model) submit() tea.Cmd {
	f := m.form
	values := make([]string, len(f.fields))
	for i := range values {
		values[i] = f.fields[i].value
	}
	m.form = nil // Never retain secrets in display state after submitting.
	switch f.action {
	case "metadata":
		var tags []string
		if strings.TrimSpace(values[0]) != "" {
			for _, tag := range strings.Split(values[0], ",") {
				tags = append(tags, strings.TrimSpace(tag))
			}
		}
		favorite, e := strconv.ParseBool(values[1])
		if e != nil {
			m.message = "Error: Favorite must be true or false. Nothing saved."
			return nil
		}
		s := m.service
		return m.begin("Saving profile metadata...", func(context.Context) completed {
			_, e := s.ProfileMetadata(f.key, tags, favorite, &f.revision)
			return completed{message: "Profile metadata saved.", err: e}
		})
	case "settings":
		n, e := strconv.ParseUint(values[0], 10, 16)
		color, ce := strconv.ParseBool(values[6])
		if e != nil || ce != nil {
			m.message = "Error: Invalid port or color value. Nothing saved."
			return nil
		}
		p := f.draft
		p.ListenerPort = uint16(n)
		p.Duration = values[1]
		p.AuthFile = values[2]
		p.ProbeTarget = values[3]
		p.CorePath = values[4]
		p.Timeout = values[5]
		p.Color = color
		if e = p.Validate(); e != nil {
			m.message = safeError(e)
			return nil
		}
		s := m.service
		return m.begin("Saving private settings...", func(ctx context.Context) completed {
			if ctx.Err() != nil {
				return completed{err: ctx.Err()}
			}
			_, e := s.SaveSettings(f.revision, p)
			return completed{message: "Settings saved durably. Reconnect explicitly to apply listener/core/deadline changes; no request sent.", err: e}
		})
	case "import-link":
		return m.begin("Parsing and storing hidden input...", func(context.Context) completed {
			_, e := m.service.Import(strings.NewReader(values[0]))
			return completed{message: "Profile imported; connection is not verified.", err: e}
		})
	case "import-file":
		return m.begin("Importing from private file...", func(context.Context) completed {
			input, e := app.ImportFile(values[0])
			if e != nil {
				return completed{err: e}
			}
			defer input.Close()
			_, e = m.service.Import(input)
			return completed{message: "Profile imported; connection is not verified.", err: e}
		})
	case "add-source":
		return m.begin("Storing protected source; no download...", func(context.Context) completed {
			var cidrs []string
			if strings.TrimSpace(values[2]) != "" {
				for _, s := range strings.Split(values[2], ",") {
					cidrs = append(cidrs, strings.TrimSpace(s))
				}
			}
			_, e := m.service.Subscriptions().Add(values[0], strings.NewReader(values[1]), cidrs, values[3])
			return completed{message: "Source stored; select Preview to fetch.", err: e}
		})
	case "rename-profile":
		return m.begin("Renaming profile...", func(context.Context) completed {
			_, e := m.service.RenameProfile(f.key, values[0])
			return completed{message: "Profile renamed.", err: e}
		})
	case "rename-source":
		return m.begin("Renaming source...", func(context.Context) completed {
			_, e := m.service.Subscriptions().Rename(f.key, values[0])
			return completed{message: "Source renamed.", err: e}
		})
	}
	return nil
}
