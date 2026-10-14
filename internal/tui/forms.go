package tui

import (
	tea "charm.land/bubbletea/v2"
	"context"
	"strconv"
	"strings"
	"time"
	"tunnelctl/internal/app"
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
	switch setting {
	case "port":
		m.openForm("port", "Session-local listener port", field{"Port", strconv.Itoa(int(m.listenPort())), false, 5})
	case "duration":
		m.openForm("duration", "Session-local duration (blank or 0 = no deadline)", field{"Duration", m.duration.String(), false, 32})
	case "auth":
		m.openForm("auth", "Session-local private listener authentication file", field{"File (blank = unauthenticated loopback)", m.authFile, false, 4096})
	case "probe":
		m.openForm("probe", "Explicit HTTPS probe target; no automatic request", field{"HTTPS destination (no credentials/query)", m.probeTarget, false, 4096})
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
	case "port":
		n, e := strconv.ParseUint(values[0], 10, 16)
		if e != nil || n < 1024 {
			m.message = "Error: Port must be 1024–65535."
		} else {
			m.port = uint16(n)
			m.message = "Success: Session-local port updated; an active session is unchanged."
		}
	case "duration":
		var d time.Duration
		var e error
		if strings.TrimSpace(values[0]) != "" {
			d, e = time.ParseDuration(values[0])
		}
		if e != nil || d < 0 {
			m.message = "Error: Use a nonnegative duration such as 10m."
		} else {
			m.duration = d
			m.message = "Success: New-session duration updated; active deadline is unchanged."
		}
	case "auth":
		m.authFile = values[0]
		m.message = "Success: Authentication path updated; validated on next start/probe."
	case "probe":
		m.probeTarget = values[0]
		m.message = "Success: Probe destination updated; no request sent."
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
