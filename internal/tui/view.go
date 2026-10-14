package tui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"fmt"
	"github.com/charmbracelet/x/ansi"
	"strings"
	"tunnelctl/internal/domain"
)

func (m model) View() tea.View {
	width, height := m.width, m.height
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 24
	}
	if width < 16 || height < 5 {
		return tea.NewView("Terminal too small\nEsc: cancel/back\nq: exit options")
	}
	title := lipgloss.NewStyle()
	if m.color {
		title = title.Bold(true).Foreground(lipgloss.Color("6"))
	}
	lines := []string{title.Render("tunnelctl | terminal proxy"), ""}
	budget := max(1, height-9)
	// All dynamic text is sanitized even if a future service forgets to do so.
	text := func(s string) { lines = append(lines, domain.SafeName(s)) }
	options := func(items []string) {
		start := max(0, m.cursor-budget+1)
		for i := start; i < len(items) && i < start+budget; i++ {
			mark := "  "
			if i == m.cursor {
				mark = "> "
			}
			text(mark + items[i])
		}
		if len(items) == 0 {
			text("Empty. No items available.")
		}
	}
	if m.form != nil {
		text(m.form.title)
		start := max(0, m.form.cursor-budget+2)
		for i := start; i < len(m.form.fields) && i < start+max(1, budget-1); i++ {
			f := m.form.fields[i]
			mark := "  "
			if i == m.form.cursor {
				mark = "> "
			}
			value := f.value
			if f.hidden {
				value = "[hidden; input is never displayed]"
			}
			// Probe destinations may contain pasted credentials until validation: do
			// not echo them either. Auth paths and all other values are control-safe.
			if m.form.action == "probe" {
				value = "[destination input hidden]"
			}
			text(mark + f.label + ": " + value)
		}
		text("Enter: next/save | Tab/arrows: fields | Ctrl+U: clear | Esc: discard")
	} else if m.exitPrompt {
		text("Detached session is running. Leaving the menu does not stop it.")
		items := []string{"Leave running and exit", "Disconnect confirmed session and exit", "Return to menu"}
		for i, item := range items {
			mark := "  "
			if i == m.exitCursor {
				mark = "> "
			}
			text(mark + item)
		}
	} else {
		switch m.page {
		case menuPage:
			text("Main menu")
			options(menu)
		case overview, sessionsPage:
			text("Session: " + m.state.State)
			status := "Disconnected"
			if m.connectFailed {
				status = "Failed: last connection attempt"
			}
			if m.state.ProcessRunning {
				status = "Connecting"
				if m.state.ListenerReady {
					status = "Listener ready; upstream unverified"
				}
			}
			if m.state.State == "cleanup_required" {
				status = "Cleanup required; use Stop"
			}
			if m.probe != nil && m.probeSession == m.state.ID && m.state.ListenerReady {
				status = "Connected: explicit HTTPS probe passed in this menu"
			}
			if m.probeFailed && m.probeSession == m.state.ID && m.state.ListenerReady {
				status = "Failed: last HTTPS probe; listener remains ready"
			}
			text("Connection: " + status)
			text(fmt.Sprintf("Listener: %s | Ready: %t", m.state.Listener, m.state.ListenerReady))
			text(fmt.Sprintf("Core: %s | Traffic: Unavailable | Latency: Unavailable", m.state.CoreVersion))
			if m.state.Deadline != nil {
				text("Deadline: " + m.state.Deadline.Local().Format("15:04:05"))
			}
			text("Proxy only. Host DNS, routes and firewall unchanged.")
			text("Detached sessions survive menu/terminal exit. x: stop with confirmation.")
			if m.page == overview {
				text("Enter: connection options")
			} else {
				text("Enter: stop with confirmation")
			}
		case profilesPage:
			text("Profiles | /: search | i: hidden import | f: private file")
			text("Enter: details | r: rename | d: remove | Filter: " + m.filter)
			items := []string{}
			for _, p := range m.visibleProfiles() {
				selected := ""
				if p.Selected {
					selected = " [selected]"
				}
				items = append(items, p.Name+selected+" | "+p.Method)
			}
			options(items)
		case sourcesPage:
			text("Subscriptions | a: add hidden source | p/u: preview update")
			text("Enter: details | r: rename | d: remove (retain profiles)")
			items := []string{}
			for _, s := range m.sources {
				items = append(items, fmt.Sprintf("%s | %d profiles", s.Name, s.Profiles))
			}
			options(items)
		case detailsPage:
			text("Profile: " + m.profile.Name)
			text(fmt.Sprintf("%s / TCP / %s | Credentials: [REDACTED]", m.profile.Protocol, m.profile.Method))
			text(fmt.Sprintf("Endpoint: %s:%d", m.profile.Host, m.profile.Port))
			text("Capability: supported normalized profile; core/config check available below.")
			text(fmt.Sprintf("Favorite: %t | Tags: %s", m.profile.Favorite, strings.Join(m.profile.Tags, ", ")))
			options([]string{"Select (active session unchanged)", "Validate with approved core (no connection)", "Rename", "Remove inactive profile", "Edit tags/favorite"})
		case sourceDetailsPage:
			text("Subscription: " + m.source.Name + " | URL: [REDACTED]")
			text(fmt.Sprintf("Profiles: %d | Revision: %d", m.source.Profiles, m.source.Revision))
			if m.source.UpdatedAt != nil {
				text("Last commit: " + m.source.UpdatedAt.UTC().Format("2006-01-02 15:04:05 UTC"))
			}
			options([]string{"Fetch preview (Esc cancels)", "Rename", "Remove source; keep standalone profiles"})
		case connectPage:
			name := "None (select in Profiles)"
			for _, p := range m.profiles {
				if p.Selected {
					name = p.Name
				}
			}
			text("Connect | Mode: Proxy | TUN: Unavailable (no certified recovery)")
			text("Detached only: exit leaves connection running; duration is enforced independently.")
			auth := "None; accessible to other local users"
			if m.authFile != "" {
				auth = "Private authentication file configured"
			}
			probe := "Not configured"
			if m.probeTarget != "" {
				probe = "Configured; explicit request only"
			}
			options([]string{"Choose profile: " + name, fmt.Sprintf("Loopback port: %d", m.listenPort()), "Duration: " + durationLabel(m.duration), "Listener authentication: " + auth, "Connect (confirm)", "Stop (confirm)", "Probe destination: " + probe, "Test current connection (confirm)"})
		case settingsPage:
			text("Persistent defaults shared with CLI. Enter opens one Save/discard form.")
			text("Listener/core changes need explicit reconnect. No global network settings.")
			options([]string{fmt.Sprintf("Loopback port: %d", m.listenPort()), "Duration: " + durationLabel(m.duration), "Private listener authentication file", "Explicit HTTPS probe destination"})
		case logsPage:
			text("Logs: bounded state events, not raw core output.")
			if len(m.events) == 0 {
				text("Empty. No session events.")
			}
			for _, event := range m.events[max(0, len(m.events)-budget):] {
				text(event.At.Format("15:04:05") + " " + event.State)
			}
		case diagnosticsPage:
			text("Diagnostics | Enter: repeat checks")
			if m.diagnostic != nil {
				text("Platform: " + m.diagnostic.Platform + " | Core: " + m.diagnostic.CoreVersion)
				text("Storage: " + m.diagnostic.Storage)
				text("Capabilities: Shadowsocks / TCP / Proxy; statistics Unavailable")
			} else {
				text("No successful diagnostic result yet.")
			}
			text("No unsolicited network test. Use Connect to explicitly probe.")
			text("TUN, Xray and service management: Unavailable")
		case helpPage:
			for _, line := range []string{"Help", "Arrows/j/k: move | Enter: open/confirm | Esc: cancel/back", "Profiles: i hidden import, f file, / search, r rename, d remove", "Subscriptions: a add, p/u preview, r rename, d remove", "s: connection options | x: confirm stop | q: exit options", "While loading: Esc/Ctrl+C cancel; q waits for cleanup then exit options.", "Forms: Enter next/save, Tab/arrows fields, Ctrl+U clear, Esc discard.", "All connections started here are detached; no silent stop on exit.", "Supported: Shadowsocks SIP002 / TCP only; TUN/UDP unavailable.", "TLS stays verified. Secrets and raw core logs are never displayed.", "CLI works independently: tunnelctl --help, status, stop, doctor."} {
				text(line)
			}
		}
	}
	if m.preview != nil {
		text(fmt.Sprintf("Preview: +%d / -%d / unchanged %d; token not displayed", len(m.preview.Added), len(m.preview.Removed), m.preview.Unchanged))
	}
	status := []string{}
	statusText := func(s string) {
		for _, part := range strings.Split(s, "\n") {
			status = append(status, strings.Split(ansi.Hardwrap(domain.SafeName(part), width, true), "\n")...)
		}
	}
	if m.busy {
		statusText("Loading: " + m.message + " | Esc cancels")
	}
	if m.confirmation != "" {
		statusText(m.confirmation)
	} else if m.message != "" && !m.busy {
		statusText(m.message)
	}
	if m.searching {
		statusText("Search: " + m.filter + " | Enter: finish | Esc: return")
	}
	maxStatus := max(1, height-4)
	if len(status) > maxStatus {
		if maxStatus < 3 {
			status = status[:maxStatus]
		} else {
			status = append(append(status[:maxStatus-2], "... resize for full message ..."), status[len(status)-1])
		}
	}
	footer := "Enter open | Esc back | ? help | q exit"
	if width < 40 {
		footer = "Enter Esc ? q"
	}
	if m.form != nil {
		footer = "Enter save | Esc discard"
	}
	if m.searching {
		footer = "Enter done | Esc back"
	}
	lines = append(lines[:min(len(lines), max(1, height-len(status)-1))], status...)
	lines = append(lines, footer)
	for i := range lines {
		lines[i] = ansi.Truncate(lines[i], width, "")
	}
	return tea.NewView(strings.Join(lines, "\n"))
}
