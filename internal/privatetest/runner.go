// Package privatetest is a development-only opt-in harness. It emits no source
// names, hosts, URLs, profile identifiers, credentials or raw error text.
package privatetest

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"tunnelctl/internal/app"
	"tunnelctl/internal/core"
	"tunnelctl/internal/domain"
	"tunnelctl/internal/importer"
	"tunnelctl/internal/subscription"
)

type Options struct {
	File, Core, ProbeURL                    string
	Connect                                 bool
	MaxSamples, MaxProfiles, MaxConnections int
	Timeout                                 time.Duration
}
type Stage struct {
	Name     string `json:"stage"`
	Result   string `json:"result"`
	Category string `json:"category,omitempty"`
	Count    int    `json:"count,omitempty"`
}
type Sample struct {
	ID       string   `json:"id"`
	Kind     string   `json:"kind"`
	Stages   []Stage  `json:"stages"`
	Profiles []Sample `json:"profiles,omitempty"`
}
type Report struct {
	Passed    bool     `json:"passed"`
	Samples   []Sample `json:"samples"`
	Truncated bool     `json:"truncated"`
	Cleanup   string   `json:"cleanup"`
	TUN       string   `json:"tun"`
}

func invalidOptions() error {
	return domain.Fail(2, "Invalid private-test options.", "Specify a file path, 1–20 samples, 1–20 profiles per source, 0–5 sequential connections and a 1s–60s timeout. Connecting also requires an explicit HTTPS probe URL.")
}
func (o Options) validate() error {
	if o.File == "" || o.MaxSamples < 1 || o.MaxSamples > 20 || o.MaxProfiles < 1 || o.MaxProfiles > 20 || o.MaxConnections < 0 || o.MaxConnections > 5 || o.Timeout < time.Second || o.Timeout > time.Minute || (o.Connect && (o.ProbeURL == "" || o.MaxConnections == 0)) {
		return invalidOptions()
	}
	return nil
}
func result(stage string, e error) Stage {
	if e == nil {
		return Stage{Name: stage, Result: "passed"}
	}
	return Stage{Name: stage, Result: "failed", Category: domain.FailureCategory(e)}
}
func classify(s string) string {
	switch {
	case strings.HasPrefix(s, "https://"):
		return "subscription"
	case strings.HasPrefix(s, "ss://"):
		return "share_link"
	case strings.Contains(s, "://"):
		return "unsupported_link"
	default:
		return "invalid_input"
	}
}
func input(path string) ([]byte, error) {
	// This development harness may read an explicitly selected file on a weak-mode
	// source filesystem. Normal application ImportFile permissions are unchanged.
	st, e := os.Lstat(path)
	if e != nil || !st.Mode().IsRegular() {
		return nil, domain.Fail(4, "Private test input is not a readable regular file.", "Use an explicit input path; symlinks/devices are refused.")
	}
	f, e := os.Open(path)
	if e != nil {
		return nil, domain.Fail(4, "Cannot open private test input.", "Check input permissions.")
	}
	defer f.Close()
	opened, e := f.Stat()
	if e != nil || !os.SameFile(st, opened) {
		return nil, domain.Fail(4, "Private test input changed during open.", "Retry with a stable private file.")
	}
	b, e := io.ReadAll(io.LimitReader(f, subscription.MaxWire+1))
	if e != nil || len(b) > subscription.MaxWire {
		return nil, domain.Fail(2, "Private test input exceeds its bounded size or cannot be read.", "Use at most 5 MiB and 64 KiB per line.")
	}
	return b, nil
}

// CheckRepositoryInput verifies metadata only. No private content is passed to Git.
func CheckRepositoryInput(path string) error {
	dir := filepath.Dir(path)
	root, e := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel").Output()
	if e != nil {
		return nil
	} // An out-of-repository private file is also supported.
	repo := strings.TrimSpace(string(root))
	rel, e := filepath.Rel(repo, path)
	if e != nil {
		return invalidOptions()
	}
	if exec.Command("git", "-C", repo, "ls-files", "--error-unmatch", "--", rel).Run() == nil {
		return domain.Fail(2, "Private input is tracked; testing is refused.", "Untrack without deleting the local file and review history before proceeding.")
	}
	if exec.Command("git", "-C", repo, "check-ignore", "--quiet", "--", rel).Run() != nil {
		return domain.Fail(2, "Private input is not ignored; testing is refused.", "Add an exact ignore rule before proceeding.")
	}
	return nil
}

func Run(ctx context.Context, o Options) (report Report, err error) {
	report = Report{Samples: []Sample{}, Cleanup: "not_started", TUN: "untested_in_this_environment; unsupported"}
	if e := o.validate(); e != nil {
		return report, e
	}
	path, e := filepath.Abs(o.File)
	if e != nil {
		return report, invalidOptions()
	}
	if e = CheckRepositoryInput(path); e != nil {
		return report, e
	}
	data, e := input(path)
	if e != nil {
		return report, e
	}
	root, e := os.MkdirTemp("", "tunnelctl-private-test-")
	if e != nil {
		return report, domain.Unexpected()
	}
	// Remove only the exact new temporary tree owned by this invocation.
	defer func() {
		if os.RemoveAll(root) != nil {
			report.Cleanup = "failed"
			err = errors.Join(err, domain.Fail(7, "Private test cleanup failed.", "Inspect the harness-owned temporary directory."))
		} else {
			report.Cleanup = "passed"
		}
		report.Passed = err == nil && len(report.Samples) > 0 && report.Cleanup == "passed"
		var failed func(Sample) bool
		failed = func(s Sample) bool {
			for _, stage := range s.Stages {
				if stage.Result == "failed" {
					return true
				}
			}
			for _, child := range s.Profiles {
				if failed(child) {
					return true
				}
			}
			return false
		}
		for _, sample := range report.Samples {
			if failed(sample) {
				report.Passed = false
			}
		}
	}()
	s, e := app.New(root, o.Core, min(o.Timeout, 10*time.Second))
	if e != nil {
		return report, e
	}
	if e = s.Init(); e != nil {
		return report, e
	}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 4096), importer.MaxLinkBytes+1)
	remaining := o.MaxConnections
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if len(report.Samples) >= o.MaxSamples {
			report.Truncated = true
			break
		}
		sample := Sample{ID: fmt.Sprintf("sample-%03d", len(report.Samples)+1), Kind: classify(line), Stages: []Stage{{Name: "classification", Result: "passed"}}}
		if ctx.Err() != nil {
			sample.Stages = append(sample.Stages, result("operation", ctx.Err()))
			report.Samples = append(report.Samples, sample)
			break
		}
		opctx, cancel := context.WithTimeout(ctx, o.Timeout)
		switch sample.Kind {
		case "subscription":
			source, e := s.Subscriptions().Add(sample.ID, strings.NewReader(line), nil, "")
			sample.Stages = append(sample.Stages, result("source_validation", e))
			if e == nil {
				res, e := s.Subscriptions().Fetcher.Fetch(opctx, domain.Subscription{URL: line})
				sample.Stages = append(sample.Stages, result("fetch", e))
				if e == nil {
					profiles, e := subscription.Parse(res.Body)
					stage := result("parse", e)
					stage.Count = len(profiles)
					sample.Stages = append(sample.Stages, stage)
					if e == nil {
						d, e := s.Subscriptions().Update(opctx, source.ID, true, false, "")
						sample.Stages = append(sample.Stages, result("preview", e))
						if e == nil {
							_, e = s.Subscriptions().Update(opctx, source.ID, false, false, d.Token)
							sample.Stages = append(sample.Stages, result("atomic_commit", e))
							if e == nil {
								testRefresh(opctx, s, source.ID, &sample)
							}
						}
						// Only committed profiles from this source are tested; failed commit has none.
						snapshot, e := s.Store.Load()
						if e == nil {
							n := 0
							for _, p := range snapshot.Profiles {
								if p.SubscriptionID != source.ID {
									continue
								}
								n++
								if n > o.MaxProfiles {
									report.Truncated = true
									break
								}
								child := Sample{ID: fmt.Sprintf("%s-profile-%03d", sample.ID, n), Kind: "profile", Stages: []Stage{}}
								check(opctx, s, p.ID, o, &remaining, &child)
								sample.Profiles = append(sample.Profiles, child)
							}
						}
					}
				}
			}
		case "share_link":
			p, e := s.Import(strings.NewReader(line))
			sample.Stages = append(sample.Stages, result("parse_and_import", e))
			if e == nil {
				check(opctx, s, p.ID, o, &remaining, &sample)
			}
		case "unsupported_link":
			sample.Stages = append(sample.Stages, Stage{Name: "parse", Result: "failed", Category: "unsupported"})
		default:
			sample.Stages = append(sample.Stages, Stage{Name: "parse", Result: "failed", Category: "invalid_input"})
		}
		cancel()
		report.Samples = append(report.Samples, sample)
	}
	if scanner.Err() != nil {
		return report, domain.Fail(2, "Private input contains an oversized or unreadable line.", "Keep each input below 64 KiB.")
	}
	return report, nil
}

func testRefresh(ctx context.Context, s *app.Service, source string, sample *Sample) {
	before, e := s.Store.Load()
	if e != nil {
		sample.Stages = append(sample.Stages, result("refetch", e))
		return
	}
	selectedID := ""
	for _, p := range before.Profiles {
		if p.SubscriptionID == source {
			selectedID = p.ID
			_, e = s.RenameProfile(p.ID, "private-test-label")
			if e == nil {
				_, e = s.Select(p.ID)
			}
			break
		}
	}
	if e != nil || selectedID == "" {
		if e == nil {
			e = domain.Fail(6, "No committed profile to check.", "Inspect transaction tests.")
		}
		sample.Stages = append(sample.Stages, result("refetch_prepare", e))
		return
	}
	d, e := s.Subscriptions().Update(ctx, source, true, false, "")
	sample.Stages = append(sample.Stages, result("refetch_preview", e))
	if e != nil {
		return
	}
	if !d.NotModified {
		_, e = s.Subscriptions().Update(ctx, source, false, false, d.Token)
	}
	if e == nil {
		after, le := s.Store.Load()
		e = le
		if e == nil {
			seen := map[string]bool{}
			for _, p := range after.Profiles {
				if p.SubscriptionID != source {
					continue
				}
				if seen[p.Identity()] {
					e = domain.Fail(6, "Duplicate identity.", "Inspect transaction tests.")
				}
				seen[p.Identity()] = true
			}
			for _, prior := range before.Profiles {
				if prior.SubscriptionID != source {
					continue
				}
				for _, p := range after.Profiles {
					if p.SubscriptionID == source && p.Identity() == prior.Identity() && p.ID != prior.ID {
						e = domain.Fail(6, "Identity changed.", "Inspect transaction tests.")
					}
				}
			}
			found := false
			for _, p := range after.Profiles {
				if p.ID == selectedID {
					found = true
				}
				if p.ID == selectedID && p.Name != "private-test-label" {
					e = domain.Fail(6, "Custom label changed.", "Inspect transaction tests.")
				}
			}
			if !found || after.Selected != selectedID {
				e = domain.Fail(6, "Selection changed.", "Inspect transaction tests.")
			}
		}
	}
	sample.Stages = append(sample.Stages, result("refetch_identity_and_label", e))
}

func check(ctx context.Context, s *app.Service, id string, o Options, remaining *int, sample *Sample) {
	// Reserve a random loopback port; app.Run independently collision-checks it.
	ln, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		sample.Stages = append(sample.Stages, result("listener_port", domain.Fail(6, "Local port unavailable.", "Check socket permission.")))
		return
	}
	l := core.Listener{Port: uint16(ln.Addr().(*net.TCPAddr).Port)}
	ln.Close()
	_, e = s.VerifyCore(ctx)
	stage := result("core_compatibility", e)
	if e != nil && stage.Category != "unsupported" && stage.Category != "local_permission" {
		stage.Category = "local_core"
	}
	sample.Stages = append(sample.Stages, stage)
	if e != nil {
		return
	}
	e = s.CheckProfile(ctx, id, l)
	sample.Stages = append(sample.Stages, result("generated_config_check", e))
	if e != nil {
		return
	}
	if !o.Connect || *remaining == 0 {
		sample.Stages = append(sample.Stages, Stage{Name: "proxy_probe", Result: "not_requested_or_limit"})
		return
	}
	*remaining = *remaining - 1
	runctx, cancel := context.WithCancel(ctx)
	defer cancel()
	ready := false
	e = s.Run(runctx, id, "proxy", l, 0, func(state domain.Session) error {
		ready = true
		sample.Stages = append(sample.Stages, Stage{Name: "listener_ready", Result: "passed"})
		_, probeErr := app.Probe(runctx, state.Listener, o.ProbeURL, l)
		sample.Stages = append(sample.Stages, result("real_https_proxy_request", probeErr))
		cancel()
		return nil
	})
	if !ready {
		readinessError := e
		if readinessError == nil {
			readinessError = ctx.Err()
		}
		if readinessError == nil {
			readinessError = domain.NetworkFail("upstream_or_local_unknown", "Startup ended without listener readiness.")
		}
		sample.Stages = append(sample.Stages, result("listener_ready", readinessError))
	}
	sample.Stages = append(sample.Stages, result("stop", e))
	state, se := s.Status()
	if se == nil && state.State != "idle" {
		se = domain.Fail(7, "Private session cleanup incomplete.", "Inspect cleanup tests.")
	}
	sample.Stages = append(sample.Stages, result("session_cleanup", se))
}
