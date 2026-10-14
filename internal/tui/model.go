package tui

import (
	tea "charm.land/bubbletea/v2"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
	"tunnelctl/internal/app"
	"tunnelctl/internal/core"
	"tunnelctl/internal/domain"
	"tunnelctl/internal/session"
	"tunnelctl/internal/subscription"
)

const (
	overview = iota
	profilesPage
	sourcesPage
	logsPage
	menuPage
	connectPage
	sessionsPage
	diagnosticsPage
	settingsPage
	helpPage
	detailsPage
	sourceDetailsPage
)

var menu = []string{"Overview", "Connect", "Profiles", "Subscriptions", "Sessions", "Logs", "Diagnostics", "Settings", "Help", "Exit"}
var menuPages = []int{overview, connectPage, profilesPage, sourcesPage, sessionsPage, logsPage, diagnosticsPage, settingsPage, helpPage}

type model struct {
	service                          *app.Service
	ctx                              context.Context
	page, cursor, width, height      int
	color, busy, searching           bool
	filter, message, confirmation    string
	confirmLabel                     string
	profiles                         []domain.PublicProfile
	sources                          []domain.PublicSubscription
	events                           []session.Event
	state                            domain.Session
	preview                          *subscription.Diff
	previewSource                    string
	profile                          domain.PublicProfile
	source                           domain.PublicSubscription
	form                             *form
	confirmAction                    func(context.Context) completed
	opCancel                         context.CancelFunc
	opID                             uint64
	canceling, exitAfter, exitPrompt bool
	exitCursor                       int
	port                             uint16
	duration                         time.Duration
	authFile, probeTarget            string
	diagnostic                       *app.Doctor
	probe                            *app.ProbeResult
	probeSession                     string
	connectFailed, probeFailed       bool
	operations                       *sync.WaitGroup
	colorAllowed                     bool
	coreOverride                     bool
	timeoutOverride                  bool
}
type loaded struct {
	profiles []domain.PublicProfile
	sources  []domain.PublicSubscription
	events   []session.Event
	state    domain.Session
	err      error
	settings app.PreferencesSnapshot
}
type tick struct{}
type completed struct {
	id           uint64
	message      string
	err          error
	preview      *subscription.Diff
	source       string
	state        *domain.Session
	doctor       *app.Doctor
	probe        *app.ProbeResult
	probeSession string
	quit         bool
	connection   bool
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
	if e != nil {
		return loaded{err: e}
	}
	settings, e := m.service.Settings()
	return loaded{profiles: p, sources: s, events: events, state: state, err: e, settings: settings}
}
func refresh() tea.Cmd        { return tea.Tick(time.Second, func(time.Time) tea.Msg { return tick{} }) }
func (m model) Init() tea.Cmd { return tea.Batch(m.load, refresh()) }
func safeError(e error) string {
	if errors.Is(e, context.Canceled) {
		return "Canceled. No automatic connection switch was requested."
	}
	var de *domain.Error
	if errors.As(e, &de) {
		return "Error: " + domain.SafeName(de.Message) + "\nRemedy: " + domain.SafeName(de.Remediation)
	}
	return "Error: Operation failed. Use Diagnostics or the CLI doctor command."
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
func (m *model) begin(label string, action func(context.Context) completed) tea.Cmd {
	ctx, cancel := context.WithCancel(m.ctx)
	m.opCancel = cancel
	m.opID++
	id := m.opID
	m.busy = true
	m.canceling = false
	m.message = label
	ch := make(chan completed, 1)
	operations := m.operations
	if operations != nil {
		operations.Add(1)
	}
	go func() {
		if operations != nil {
			defer operations.Done()
		}
		v := action(ctx)
		v.id = id
		cancel()
		ch <- v
	}()
	return func() tea.Msg { return <-ch }
}
func (m *model) confirm(text string, action func(context.Context) completed) {
	m.confirmation = text + " Enter/y: confirm | Esc/n: return"
	m.confirmAction = action
	m.confirmLabel = "Applying confirmed operation..."
}
func (m *model) leave() tea.Cmd {
	m.form = nil
	m.searching = false
	m.confirmation = ""
	m.preview = nil
	if m.busy {
		m.exitAfter = true
		m.canceling = true
		if m.opCancel != nil {
			m.opCancel()
		}
		m.message = "Canceling; waiting for owned operation cleanup before leaving."
		return nil
	}
	// Refresh at exit; do not infer lifetime from an old status snapshot.
	return m.begin("Checking session before exit...", func(context.Context) completed {
		state, e := m.service.Status()
		return completed{state: &state, err: e, quit: e == nil && !state.ProcessRunning}
	})
}
func (m *model) count() int {
	switch m.page {
	case menuPage:
		return len(menu)
	case profilesPage:
		return len(m.visibleProfiles())
	case sourcesPage:
		return len(m.sources)
	case connectPage:
		return 8
	case settingsPage:
		return 4
	case detailsPage:
		return 5
	case sourceDetailsPage:
		return 3
	}
	return 1
}
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case loaded:
		if msg.err != nil {
			if !m.busy {
				m.message = safeError(msg.err)
			}
		} else {
			m.profiles = msg.profiles
			m.sources = msg.sources
			m.events = msg.events
			m.state = msg.state
			m.applySettings(msg.settings)
			for _, p := range m.profiles {
				if p.ID == m.profile.ID {
					m.profile = p
					break
				}
			}
			for _, s := range m.sources {
				if s.ID == m.source.ID {
					m.source = s
					break
				}
			}
			m.cursor = min(m.cursor, max(0, m.count()-1))
		}
	case tick:
		if !m.busy {
			return m, tea.Batch(m.load, refresh())
		}
		return m, refresh()
	case completed:
		if msg.id != m.opID {
			return m, nil
		}
		m.busy = false
		m.opCancel = nil
		canceled := m.canceling
		m.canceling = false
		if msg.err != nil {
			m.message = safeError(msg.err)
		} else {
			m.message = "Success: " + msg.message
		}
		if msg.state != nil {
			m.state = *msg.state
			if msg.state.ProcessRunning && !m.exitAfter {
				m.exitPrompt = true
				m.exitCursor = 0
			}
		}
		if msg.doctor != nil && msg.err == nil {
			m.diagnostic = msg.doctor
		}
		if msg.connection {
			m.connectFailed = msg.err != nil
		}
		if msg.probeSession != "" {
			m.probeSession = msg.probeSession
			m.probeFailed = msg.err != nil
			m.probe = nil
			if msg.err == nil {
				m.probe = msg.probe
			}
		}
		if msg.preview != nil && msg.err == nil && !canceled {
			m.preview = msg.preview
			m.previewSource = msg.source
			source, token := msg.source, msg.preview.Token
			m.confirm(fmt.Sprintf("Apply snapshot? Added %d; removed %d; retain selected %d.", len(msg.preview.Added), len(msg.preview.Removed), len(msg.preview.RetainedSelected)), func(ctx context.Context) completed {
				_, e := m.service.Subscriptions().Update(ctx, source, false, true, token)
				return completed{message: "Snapshot committed; active session was not switched.", err: e}
			})
		}
		if msg.quit && msg.err == nil {
			return m, tea.Quit
		}
		if m.exitAfter {
			m.exitAfter = false
			return m, m.leave()
		}
		return m, m.load
	case tea.PasteMsg:
		if m.form != nil {
			m.form.append(msg.Content)
		} else if m.searching {
			m.filter = domain.SafeName(m.filter + msg.Content)
			if len(m.filter) > 512 {
				m.filter = ""
			}
			m.cursor = 0
		}
	case tea.KeyPressMsg:
		key := msg.String()
		if m.busy {
			if key == "esc" || key == "ctrl+c" {
				m.canceling = true
				if m.opCancel != nil {
					m.opCancel()
				}
				m.message = "Canceling; waiting for cleanup..."
			}
			if key == "q" {
				return m, m.leave()
			}
			return m, nil
		}
		if m.exitPrompt {
			switch key {
			case "esc", "n":
				m.exitPrompt = false
			case "up", "k":
				m.exitCursor = max(0, m.exitCursor-1)
			case "down", "j":
				m.exitCursor = min(2, m.exitCursor+1)
			case "enter":
				m.exitPrompt = false
				if m.exitCursor == 0 {
					return m, tea.Quit
				}
				if m.exitCursor == 1 {
					id := m.state.ID
					return m, m.begin("Stopping confirmed session before exit...", func(context.Context) completed {
						return completed{message: "Session stopped.", err: m.service.StopSession(id), quit: true}
					})
				}
			}
			return m, nil
		}
		if m.confirmation != "" {
			action := m.confirmAction
			if key == "enter" || key == "y" {
				m.confirmation = ""
				m.preview = nil
				m.confirmAction = nil
				return m, m.begin(m.confirmLabel, action)
			}
			if key == "esc" || key == "n" || key == "q" || key == "ctrl+c" {
				m.confirmation = ""
				m.preview = nil
				m.confirmAction = nil
				m.message = "Canceled; no confirmed action applied."
			}
			return m, nil
		}
		if m.form != nil {
			return m, m.formKey(msg)
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
			case "ctrl+c":
				m.searching = false
			default:
				if msg.Text != "" && len(m.filter)+len(msg.Text) <= 512 {
					m.filter += domain.SafeName(msg.Text)
				}
			}
			m.cursor = 0
			return m, nil
		}
		switch key {
		case "q", "ctrl+c":
			return m, m.leave()
		case "esc":
			switch m.page {
			case detailsPage:
				m.page = profilesPage
			case sourceDetailsPage:
				m.page = sourcesPage
			default:
				m.page = menuPage
			}
			m.cursor = 0
			m.message = ""
		case "tab", "right":
			if m.page <= logsPage {
				m.page = (m.page + 1) % 4
				m.cursor = 0
			}
		case "left":
			if m.page <= logsPage {
				m.page = (m.page + 3) % 4
				m.cursor = 0
			}
		case "1", "2", "3", "4":
			m.page = int(key[0] - '1')
			m.cursor = 0
		case "up", "k":
			m.cursor = max(0, m.cursor-1)
		case "down", "j":
			m.cursor = min(max(0, m.count()-1), m.cursor+1)
		case "/":
			if m.page == profilesPage {
				m.searching = true
			}
		case "?":
			m.page = helpPage
			m.cursor = 0
		case "s":
			m.page = connectPage
			m.cursor = 0
		case "x":
			m.stopConfirmation()
		case "i":
			if m.page == profilesPage {
				m.openForm("import-link", "Import a share link (hidden)", field{"Share link", "", true, 64 * 1024})
			}
		case "f":
			if m.page == profilesPage {
				m.openForm("import-file", "Import from owned 0600 file", field{"File path", "", false, 4096})
			}
		case "a":
			if m.page == sourcesPage {
				m.addSourceForm()
			}
		case "p", "u":
			if m.page == sourcesPage && m.cursor < len(m.sources) {
				return m, m.updateSource(m.sources[m.cursor].ID)
			}
			if m.page == sourceDetailsPage {
				return m, m.updateSource(m.source.ID)
			}
		case "r":
			m.renameForm()
		case "d":
			m.removeConfirmation()
		case "enter":
			return m, m.enter()
		}
	}
	return m, nil
}
func (m *model) stopConfirmation() {
	id := m.state.ID
	m.confirm("Stop this owned session?", func(context.Context) completed {
		return completed{message: "Session stopped and owned resources reconciled.", err: m.service.StopSession(id)}
	})
}
func (m *model) updateSource(id string) tea.Cmd {
	return m.begin("Loading subscription preview; Esc cancels...", func(ctx context.Context) completed {
		d, e := m.service.Subscriptions().Update(ctx, id, true, false, "")
		if d.NotModified {
			return completed{message: "Source returned Not Modified; healthy snapshot retained.", err: e}
		}
		return completed{message: "Preview ready.", err: e, preview: &d, source: id}
	})
}
func (m *model) enter() tea.Cmd {
	switch m.page {
	case menuPage:
		if m.cursor == len(menu)-1 {
			return m.leave()
		}
		m.page = menuPages[m.cursor]
		m.cursor = 0
		if m.page == diagnosticsPage {
			return m.doctor()
		}
	case overview:
		m.page = connectPage
		m.cursor = 0
	case profilesPage:
		p := m.visibleProfiles()
		if m.cursor < len(p) {
			m.profile = p[m.cursor]
			m.page = detailsPage
			m.cursor = 0
		}
	case sourcesPage:
		if m.cursor < len(m.sources) {
			m.source = m.sources[m.cursor]
			m.page = sourceDetailsPage
			m.cursor = 0
		}
	case detailsPage:
		switch m.cursor {
		case 0:
			id := m.profile.ID
			return m.begin("Selecting profile...", func(context.Context) completed {
				_, e := m.service.Select(id)
				return completed{message: "Profile selected; an active session is unchanged.", err: e}
			})
		case 1:
			id := m.profile.ID
			return m.begin("Validating generated configuration; no upstream request...", func(ctx context.Context) completed {
				return completed{message: "Approved core accepted generated configuration; connectivity is unverified.", err: m.service.CheckProfile(ctx, id, core.Listener{Port: m.listenPort()})}
			})
		case 2:
			m.renameForm()
		case 3:
			m.removeConfirmation()
		case 4:
			p, revision, e := m.service.ProfileSettings(m.profile.ID)
			if e != nil {
				m.message = safeError(e)
				return nil
			}
			m.openForm("metadata", "Profile metadata; Save both fields together", field{"Tags (comma separated)", strings.Join(p.Tags, ","), false, 2048}, field{"Favorite (true/false)", fmt.Sprint(p.Favorite), false, 5}, field{"Save", "", false, 0})
			m.form.key = p.ID
			m.form.revision = revision
		}
	case sourceDetailsPage:
		switch m.cursor {
		case 0:
			return m.updateSource(m.source.ID)
		case 1:
			m.renameForm()
		case 2:
			m.removeConfirmation()
		}
	case sessionsPage:
		m.stopConfirmation()
	case connectPage:
		switch m.cursor {
		case 0:
			m.page = profilesPage
			m.cursor = 0
		case 1:
			m.settingForm("port")
		case 2:
			m.settingForm("duration")
		case 3:
			m.settingForm("auth")
		case 4:
			m.connectConfirmation()
		case 5:
			m.stopConfirmation()
		case 6:
			m.settingForm("probe")
		case 7:
			return m.testConnection()
		}
	case settingsPage:
		m.settingForm([]string{"port", "duration", "auth", "probe"}[m.cursor])
	case diagnosticsPage:
		return m.doctor()
	}
	return nil
}
func (m model) listenPort() uint16 {
	if m.port == 0 {
		return 2080
	}
	return m.port
}
func (m *model) connectConfirmation() {
	id := ""
	for _, p := range m.profiles {
		if p.Selected {
			id = p.ID
			break
		}
	}
	if id == "" {
		m.message = "Error: Select a profile in Profiles before connecting."
		return
	}
	port, duration, path := m.listenPort(), m.duration, m.authFile
	m.confirm(fmt.Sprintf("Start detached Proxy on 127.0.0.1:%d for %s? Exit leaves it running. Upstream unverified.", port, durationLabel(duration)), func(ctx context.Context) completed {
		l, e := app.AuthFile(path, core.Listener{Port: port})
		if e != nil {
			return completed{err: e, connection: true}
		}
		_, e = m.service.Start(ctx, id, l, duration)
		return completed{message: "Listener ready (upstream unverified). Detached session survives menu exit.", err: e, connection: true}
	})
	m.confirmLabel = "Connecting; validating config and waiting for owned listener..."
}
func (m *model) doctor() tea.Cmd {
	return m.begin("Checking storage/core capabilities...", func(ctx context.Context) completed {
		d, e := m.service.Doctor(ctx)
		return completed{message: "Diagnostics complete; no host network changes or unsolicited probe.", err: e, doctor: &d}
	})
}
func (m *model) testConnection() tea.Cmd {
	if m.probeTarget == "" {
		m.message = "Error: Set an explicit HTTPS probe destination first."
		return nil
	}
	state, target, path := m.state, m.probeTarget, m.authFile
	if !state.ListenerReady {
		m.message = "Error: No ready session. Connect first."
		return nil
	}
	m.confirm("Send one bounded TLS-verified request to the configured destination via the current proxy?", func(ctx context.Context) completed {
		p, e := m.service.ProbeSession(ctx, state.ID, target, path)
		return completed{message: "Real HTTPS proxy request passed. This result is local to this menu, not a traffic counter.", err: e, probe: &p, probeSession: state.ID}
	})
	return nil
}
func durationLabel(d time.Duration) string {
	if d == 0 {
		return "No deadline"
	}
	return d.String()
}
func (m *model) applySettings(v app.PreferencesSnapshot) {
	m.port = v.Preferences.ListenerPort
	m.duration, _ = time.ParseDuration(v.Preferences.Duration)
	m.authFile = v.Preferences.AuthFile
	m.probeTarget = v.Preferences.ProbeTarget
	m.color = m.colorAllowed && v.Preferences.Color && os.Getenv("NO_COLOR") == ""
	m.service = m.service.WithPreferences(v.Preferences, m.coreOverride, m.timeoutOverride)
}

type Options struct {
	ColorAllowed    bool
	CoreOverride    bool
	TimeoutOverride bool
}

func Run(ctx context.Context, s *app.Service, in io.Reader, out io.Writer, options Options) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	operations := &sync.WaitGroup{}
	v, e := s.Settings()
	if e != nil {
		return e
	}
	m := model{service: s, ctx: ctx, operations: operations, page: menuPage, colorAllowed: options.ColorAllowed, coreOverride: options.CoreOverride, timeoutOverride: options.TimeoutOverride, state: domain.Session{State: "idle"}}
	m.applySettings(v)
	final, e := tea.NewProgram(m, tea.WithContext(ctx), tea.WithInput(in), tea.WithOutput(out)).Run()
	wasCanceled := ctx.Err() != nil
	cancel()
	if last, ok := final.(model); ok && last.opCancel != nil {
		last.opCancel()
	}
	// Even terminal loss waits for cancellation of in-flight owned startup/fetch.
	operations.Wait()
	if e != nil && !wasCanceled {
		return domain.Fail(1, "The terminal interface failed.", "Use CLI commands; an established detached session remains running.")
	}
	return nil
}
