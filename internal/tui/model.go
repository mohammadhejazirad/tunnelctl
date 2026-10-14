package tui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"context"
	"errors"
	"fmt"
	"github.com/charmbracelet/x/ansi"
	"io"
	"os"
	"strings"
	"time"
	"tunnelctl/internal/app"
	"tunnelctl/internal/core"
	"tunnelctl/internal/domain"
	"tunnelctl/internal/session"
	"tunnelctl/internal/subscription"
)

type model struct {
	service                       *app.Service
	ctx                           context.Context
	page, cursor, width, height   int
	color, busy, searching        bool
	filter, message, confirmation string
	profiles                      []domain.PublicProfile
	sources                       []domain.PublicSubscription
	events                        []session.Event
	state                         domain.Session
	preview                       *subscription.Diff
	previewSource                 string
}
type loaded struct {
	profiles []domain.PublicProfile
	sources  []domain.PublicSubscription
	events   []session.Event
	state    domain.Session
	err      error
}
type tick struct{}
type completed struct {
	message string
	err     error
	preview *subscription.Diff
	source  string
}

func (m model) load() tea.Msg {
	p, e := m.service.List()
	if e != nil {
		return loaded{err: e}
	}
	s, e := m.service.Subscriptions().List()
	if e != nil {
		return loaded{err: e}
	}
	state, e := m.service.Status()
	if e != nil {
		return loaded{err: e}
	}
	events, e := m.service.Logs()
	return loaded{p, s, events, state, e}
}
func refresh() tea.Cmd        { return tea.Tick(time.Second, func(time.Time) tea.Msg { return tick{} }) }
func (m model) Init() tea.Cmd { return tea.Batch(m.load, refresh()) }
func safeError(e error) string {
	var de *domain.Error
	if errors.As(e, &de) {
		return de.Message
	}
	return "Operation failed; use doctor for diagnostics."
}
func (m model) visibleProfiles() []domain.PublicProfile {
	out := []domain.PublicProfile{}
	for _, p := range m.profiles {
		if strings.Contains(strings.ToLower(p.Name), strings.ToLower(m.filter)) {
			out = append(out, p)
		}
	}
	return out
}
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case loaded:
		if msg.err != nil {
			m.message = safeError(msg.err)
		} else {
			m.profiles = msg.profiles
			m.sources = msg.sources
			m.events = msg.events
			m.state = msg.state
		}
		return m, nil
	case tick:
		if !m.busy {
			return m, tea.Batch(m.load, refresh())
		}
		return m, refresh()
	case completed:
		m.busy = false
		if msg.err != nil {
			m.message = safeError(msg.err)
		} else {
			m.message = msg.message
			if msg.preview != nil {
				m.preview = msg.preview
				m.previewSource = msg.source
				m.confirmation = fmt.Sprintf("Apply preview: +%d / -%d / retain selected %d? y/N", len(msg.preview.Added), len(msg.preview.Removed), len(msg.preview.RetainedSelected))
			}
		}
		return m, m.load
	case tea.KeyPressMsg:
		key := msg.String()
		if key == "ctrl+c" || (!m.searching && key == "q") {
			return m, tea.Quit
		}
		if m.confirmation != "" {
			m.confirmation = ""
			if key == "y" {
				m.busy = true
				if m.preview != nil {
					source, token := m.previewSource, m.preview.Token
					m.preview = nil
					return m, func() tea.Msg {
						_, e := m.service.Subscriptions().Update(m.ctx, source, false, true, token)
						return completed{message: "Snapshot committed; active session was not switched.", err: e}
					}
				}
				return m, func() tea.Msg { return completed{message: "Owned session stopped.", err: m.service.Stop()} }
			}
			m.preview = nil
			return m, nil
		}
		if m.searching {
			switch key {
			case "enter", "esc":
				m.searching = false
			case "backspace":
				r := []rune(m.filter)
				if len(r) > 0 {
					m.filter = string(r[:len(r)-1])
				}
			default:
				if len([]rune(key)) == 1 && len(m.filter) < 512 {
					m.filter += key
				}
			}
			m.cursor = 0
			return m, nil
		}
		if m.busy {
			return m, nil
		}
		switch key {
		case "tab", "right":
			m.page = (m.page + 1) % 4
			m.cursor = 0
		case "left":
			m.page = (m.page + 3) % 4
			m.cursor = 0
		case "1", "2", "3", "4":
			m.page = int(key[0] - '1')
			m.cursor = 0
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			m.cursor++
		case "/":
			if m.page == 1 {
				m.searching = true
			}
		case "?":
			m.message = "Tab: screens; j/k: move; Enter: select; s: detached start; x: stop; p: subscription preview; q: leave (detached session remains)."
		case "x":
			m.preview = nil
			m.confirmation = "Stop the owned session? y/N"
		case "s":
			m.busy = true
			return m, func() tea.Msg {
				_, e := m.service.Start(m.ctx, "", core.Listener{Port: 2080}, 0)
				return completed{message: "Detached proxy ready; quitting the UI leaves it running.", err: e}
			}
		case "enter":
			if m.page == 1 {
				p := m.visibleProfiles()
				if m.cursor < len(p) {
					id := p[m.cursor].ID
					m.busy = true
					return m, func() tea.Msg {
						_, e := m.service.Select(id)
						return completed{message: "Profile selected; active connection was not switched.", err: e}
					}
				}
			}
		case "p":
			if m.page == 2 && m.cursor < len(m.sources) {
				id := m.sources[m.cursor].ID
				m.busy = true
				return m, func() tea.Msg {
					d, e := m.service.Subscriptions().Update(m.ctx, id, true, false, "")
					if d.NotModified {
						return completed{message: "Source returned Not Modified.", err: e}
					}
					return completed{message: "Preview ready.", err: e, preview: &d, source: id}
				}
			}
		}
		count := len(m.visibleProfiles())
		if m.page == 2 {
			count = len(m.sources)
		}
		if m.cursor >= count {
			m.cursor = max(0, count-1)
		}
	}
	return m, nil
}
func (m model) View() tea.View {
	width, height := m.width, m.height
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 24
	}
	if width < 16 || height < 5 {
		return tea.NewView("Terminal too small\nq: leave")
	}
	title := lipgloss.NewStyle()
	if m.color {
		title = title.Bold(true).Foreground(lipgloss.Color("6"))
	}
	lines := []string{title.Render("tunnelctl  |  terminal proxy"), "1 Overview  2 Profiles  3 Sources  4 Events", ""}
	budget := max(1, height-8)
	switch m.page {
	case 0:
		lines = append(lines, fmt.Sprintf("State: %s", m.state.State), fmt.Sprintf("Listener: %s  Ready: %t", m.state.Listener, m.state.ListenerReady), fmt.Sprintf("Upstream verified: %t  Traffic: Unavailable", m.state.ConnectionTested), "TUN: Unsupported; no certified privileged adapter", "No host DNS, routes or firewall changes.")
	case 1:
		lines = append(lines, "Enter selects a profile. / filters names.")
		p := m.visibleProfiles()
		start := max(0, m.cursor-budget+1)
		for i := start; i < len(p) && i < start+budget; i++ {
			mark := "  "
			if i == m.cursor {
				mark = "> "
			}
			selected := ""
			if p[i].Selected {
				selected = " [selected]"
			}
			lines = append(lines, mark+p[i].Name+selected+"  "+p[i].Method)
		}
		if len(p) == 0 {
			lines = append(lines, "No matching profiles. Import using the CLI.")
		}
	case 2:
		lines = append(lines, "p previews a manual refresh; applying requires confirmation.")
		start := max(0, m.cursor-budget+1)
		for i := start; i < len(m.sources) && i < start+budget; i++ {
			mark := "  "
			if i == m.cursor {
				mark = "> "
			}
			lines = append(lines, fmt.Sprintf("%s%s  %d profiles", mark, m.sources[i].Name, m.sources[i].Profiles))
		}
		if len(m.sources) == 0 {
			lines = append(lines, "No sources. Add through subscription add --url-stdin.")
		}
	case 3:
		lines = append(lines, "Private state events; raw core logs are unavailable.")
		for _, event := range m.events[max(0, len(m.events)-budget):] {
			lines = append(lines, event.At.Format("15:04:05")+"  "+event.State)
		}
	}
	if m.searching {
		lines = append(lines, "Filter: "+m.filter)
	}
	if m.busy {
		lines = append(lines, "Operation in progress; Ctrl+C cancels UI operations.")
	}
	if m.confirmation != "" {
		lines = append(lines, m.confirmation)
	} else if m.message != "" {
		lines = append(lines, m.message)
	}
	lines = append(lines, "Tab screens | j/k move | s start | x stop | ? help | q leave")
	if len(lines) > height {
		lines = append(lines[:height-2], lines[len(lines)-2:]...)
	}
	for i := range lines {
		lines[i] = ansi.Truncate(lines[i], width, "")
	}
	return tea.NewView(strings.Join(lines, "\n"))
}
func Run(ctx context.Context, s *app.Service, in io.Reader, out io.Writer, color bool) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	m := model{service: s, ctx: ctx, color: color && os.Getenv("NO_COLOR") == "", state: domain.Session{State: "idle"}}
	_, e := tea.NewProgram(m, tea.WithContext(ctx), tea.WithInput(in), tea.WithOutput(out)).Run()
	if e != nil && ctx.Err() == nil {
		return domain.Fail(1, "The terminal interface failed.", "Use the CLI commands; no detached session is stopped by leaving the UI.")
	}
	return nil
}
