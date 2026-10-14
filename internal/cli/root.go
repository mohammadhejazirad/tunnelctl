package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"io"
	"os/signal"
	"time"
	"tunnelctl/internal/app"
	"tunnelctl/internal/buildinfo"
	"tunnelctl/internal/core"
	"tunnelctl/internal/domain"
	"tunnelctl/internal/platform"
)

type envelope struct {
	SchemaVersion int           `json:"schema_version"`
	OK            bool          `json:"ok"`
	Data          any           `json:"data"`
	Error         *domain.Error `json:"error,omitempty"`
}
type options struct {
	root           string
	core           string
	json           bool
	plain          bool
	noColor        bool
	nonInteractive bool
	timeout        time.Duration
	command        string
}

func Execute(args []string, in io.Reader, out, diagnostics io.Writer) int {
	o := &options{}
	cmd := newRoot(o, in, out)
	cmd.SetErr(diagnostics)
	cmd.SetArgs(args)
	if e := cmd.Execute(); e != nil {
		var childExit *app.ChildExit
		if errors.As(e, &childExit) && cleanupError(e, nil) == nil {
			if o.json {
				json.NewEncoder(out).Encode(envelope{domain.OutputSchemaVersion, false, map[string]int{"child_exit_code": childExit.Code}, &domain.Error{Code: childExit.Code, Message: "The child command exited unsuccessfully.", Remediation: "Check the child command."}})
			}
			return childExit.Code
		}
		var de *domain.Error
		if !errors.As(e, &de) {
			de = &domain.Error{Code: 2, Message: "Invalid command or arguments.", Remediation: "Run tunnelctl --help and check the command syntax."}
		} // Never echo Cobra errors: they can contain secret-bearing user arguments.
		// A joined cleanup failure takes precedence over a preceding failure.
		de = cleanupError(e, de)
		if o.json {
			if json.NewEncoder(out).Encode(envelope{domain.OutputSchemaVersion, false, nil, de}) != nil {
				return 1
			}
		} else {
			fmt.Fprintf(diagnostics, "Error: %s\nRemediation: %s\n", de.Message, de.Remediation)
		}
		return de.Code
	}
	return 0
}
func cleanupError(e error, fallback *domain.Error) *domain.Error {
	if v, ok := e.(*domain.Error); ok && v.Code == 7 {
		return v
	}
	if v, ok := e.(interface{ Unwrap() []error }); ok {
		for _, part := range v.Unwrap() {
			if found := cleanupError(part, nil); found != nil {
				return found
			}
		}
	}
	return fallback
}
func newRoot(o *options, in io.Reader, out io.Writer) *cobra.Command {
	root := &cobra.Command{Use: "tunnelctl", Short: "A terminal-only proxy client", SilenceErrors: true, SilenceUsage: true, Args: cobra.NoArgs}
	root.SetIn(in)
	root.SetOut(out)
	root.SetErr(io.Discard)
	root.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		o.command = cmd.Name()
		_, e := buildinfo.Current()
		return e
	}
	root.CompletionOptions.DisableDefaultCmd = true
	root.PersistentFlags().StringVar(&o.root, "data-dir", "", "Private storage root override (0700)")
	root.PersistentFlags().StringVar(&o.core, "core", app.DefaultCore(), "Absolute path to the approved sing-box executable")
	root.PersistentFlags().BoolVar(&o.json, "json", false, "Write schema-versioned JSON results")
	root.PersistentFlags().BoolVar(&o.plain, "plain", false, "Disable TUI styling; CLI output is always plain")
	root.PersistentFlags().BoolVar(&o.noColor, "no-color", false, "Disable TUI color")
	root.PersistentFlags().BoolVar(&o.nonInteractive, "non-interactive", false, "Never prompt; refuse interactive TUI")
	root.PersistentFlags().DurationVar(&o.timeout, "timeout", 10*time.Second, "Core preflight and readiness timeout")
	service := func() (*app.Service, error) {
		if o.timeout <= 0 || o.timeout > 2*time.Minute {
			return nil, domain.Fail(2, "Timeout must be greater than zero and at most two minutes.", "Pass --timeout 10s.")
		}
		s, e := app.New(o.root, o.core, o.timeout)
		if e != nil {
			return nil, e
		}
		// Recovery must not depend on a readable preferences snapshot.
		if o.command == "stop" || o.command == "status" || o.command == "logs" {
			return s, nil
		}
		v, e := s.Settings()
		if e != nil {
			return nil, e
		}
		return s.WithPreferences(v.Preferences, root.PersistentFlags().Changed("core"), root.PersistentFlags().Changed("timeout")), nil
	}
	write := func(v any) error {
		if o.json {
			if e := json.NewEncoder(out).Encode(envelope{domain.OutputSchemaVersion, true, v, nil}); e != nil {
				return domain.Unexpected()
			}
			return nil
		}
		switch x := v.(type) {
		case domain.PublicProfile:
			if _, e := fmt.Fprintf(out, "%s  %s  %s:%d  %s  credentials=%s  selected=%t\n", x.ID, x.Name, x.Host, x.Port, x.Method, x.Credentials, x.Selected); e != nil {
				return domain.Unexpected()
			}
		case []domain.PublicProfile:
			for _, p := range x {
				if _, e := fmt.Fprintf(out, "%s  %s  %s  selected=%t\n", p.ID, p.Name, p.Method, p.Selected); e != nil {
					return domain.Unexpected()
				}
			}
		case domain.Session:
			if _, e := fmt.Fprintln(out, x.String()); e != nil {
				return domain.Unexpected()
			}
		default:
			b, e := json.MarshalIndent(v, "", "  ")
			if e != nil {
				return domain.Unexpected()
			}
			_, e = fmt.Fprintln(out, string(b))
			if e != nil {
				return domain.Unexpected()
			}
			return nil
		}
		return nil
	}
	root.RunE = func(cmd *cobra.Command, _ []string) error {
		if interactive(o, in, out) {
			return enterTUI(cmd, o, service, in, out)
		}
		if o.json {
			return write(map[string]string{"usage": "tunnelctl --help", "menu": "Run tunnelctl tui with terminal input and output."})
		}
		return cmd.Help()
	}
	root.AddCommand(&cobra.Command{Use: "version", Short: "Print application and pinned core versions", Args: cobra.NoArgs, RunE: func(*cobra.Command, []string) error {
		v, e := buildinfo.Current()
		if e != nil {
			return e
		}
		return write(v)
	}})
	root.AddCommand(&cobra.Command{Use: "init", Short: "Initialize private versioned storage", Args: cobra.NoArgs, RunE: func(*cobra.Command, []string) error {
		s, e := service()
		if e != nil {
			return e
		}
		if e = s.Init(); e != nil {
			return e
		}
		return write(map[string]any{"initialized": true, "schema_version": domain.SchemaVersion})
	}})
	profile := &cobra.Command{Use: "profile", Short: "Manage private connection profiles", Args: cobra.NoArgs}
	root.AddCommand(profile)
	root.AddCommand(subscriptionCommand(service, write, in))
	root.AddCommand(coreCommand(service, write))
	root.AddCommand(settingsCommand(service, write))
	root.AddCommand(&cobra.Command{Use: "paths", Short: "Show per-user config, state, cache and runtime paths without secrets", Args: cobra.NoArgs, RunE: func(*cobra.Command, []string) error {
		s, e := app.New(o.root, o.core, o.timeout)
		if e != nil {
			return e
		}
		return write(s.Paths())
	}})
	var tags []string
	var favorite bool
	metadata := &cobra.Command{Use: "metadata PROFILE", Short: "Persist tags and favorite together (replaces both fields)", Args: cobra.ExactArgs(1), RunE: func(_ *cobra.Command, args []string) error {
		s, e := service()
		if e != nil {
			return e
		}
		v, e := s.ProfileMetadata(args[0], tags, favorite, nil)
		if e != nil {
			return e
		}
		return write(v)
	}}
	metadata.Flags().StringSliceVar(&tags, "tags", nil, "Comma-separated tags; replaces all tags")
	metadata.Flags().BoolVar(&favorite, "favorite", false, "Mark favorite")
	profile.AddCommand(metadata)
	root.AddCommand(probeCommand(service, write))
	profile.AddCommand(profileCheckCommand(service, write))
	root.AddCommand(tuiCommand(o, service, in, out))
	profile.AddCommand(&cobra.Command{Use: "rename PROFILE NAME", Short: "Rename a profile and preserve its label during refresh", Args: cobra.ExactArgs(2), RunE: func(_ *cobra.Command, args []string) error {
		s, e := service()
		if e != nil {
			return e
		}
		v, e := s.RenameProfile(args[0], args[1])
		if e != nil {
			return e
		}
		return write(v)
	}})
	profile.AddCommand(&cobra.Command{Use: "remove PROFILE", Short: "Remove an inactive profile", Args: cobra.ExactArgs(1), RunE: func(_ *cobra.Command, args []string) error {
		s, e := service()
		if e != nil {
			return e
		}
		if e = s.RemoveProfile(args[0]); e != nil {
			return e
		}
		return write(map[string]bool{"removed": true})
	}})
	var file string
	var stdin bool
	importCmd := &cobra.Command{Use: "import", Short: "Import one Shadowsocks SIP002 link from private file or stdin", Args: cobra.NoArgs, RunE: func(*cobra.Command, []string) error {
		if (file == "") == !stdin {
			return domain.Fail(2, "Choose exactly one of --file or --stdin.", "Provide a private file path or pipe one share link into stdin.")
		}
		var r io.Reader = in
		if file != "" {
			f, e := app.ImportFile(file)
			if e != nil {
				return e
			}
			defer f.Close()
			r = f
		}
		s, e := service()
		if e != nil {
			return e
		}
		p, e := s.Import(r)
		if e != nil {
			return e
		}
		return write(p)
	}}
	importCmd.Flags().StringVar(&file, "file", "", "Private file containing one share link")
	importCmd.Flags().BoolVar(&stdin, "stdin", false, "Read one share link from stdin")
	profile.AddCommand(importCmd)
	profile.AddCommand(&cobra.Command{Use: "list", Short: "List redacted profiles", Args: cobra.NoArgs, RunE: func(*cobra.Command, []string) error {
		s, e := service()
		if e != nil {
			return e
		}
		p, e := s.List()
		if e != nil {
			return e
		}
		return write(p)
	}})
	show := &cobra.Command{Use: "show PROFILE", Short: "Show a profile with credentials always redacted", Args: cobra.ExactArgs(1), RunE: func(_ *cobra.Command, args []string) error {
		s, e := service()
		if e != nil {
			return e
		}
		p, e := s.Show(args[0])
		if e != nil {
			return e
		}
		return write(p)
	}}
	show.Flags().Bool("redacted", true, "Credentials are always redacted")
	profile.AddCommand(show)
	profile.AddCommand(&cobra.Command{Use: "select PROFILE", Short: "Select a profile ID or unique name", Args: cobra.ExactArgs(1), RunE: func(_ *cobra.Command, args []string) error {
		s, e := service()
		if e != nil {
			return e
		}
		p, e := s.Select(args[0])
		if e != nil {
			return e
		}
		return write(p)
	}})
	var key, mode, authFile string
	var duration time.Duration
	var port uint
	run := &cobra.Command{Use: "run", Short: "Run a loopback proxy in the foreground; Ctrl+C stops it", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		s, e := service()
		if e != nil {
			return e
		}
		port, duration, authFile, mode, e = sessionDefaults(cmd, s, port, duration, authFile, mode)
		if e != nil {
			return e
		}
		if port < 1024 || port > 65535 {
			return domain.Fail(2, "Listener port must be between 1024 and 65535.", "Choose an unprivileged --listen-port.")
		}
		l, e := app.AuthFile(authFile, core.Listener{Port: uint16(port)})
		if e != nil {
			return e
		}
		ctx, cancel := signal.NotifyContext(cmd.Context(), platform.StopSignals()...)
		defer cancel()
		e = s.Run(ctx, key, mode, l, duration, func(v domain.Session) error {
			if o.json {
				return nil
			}
			return write(v)
		})
		if e != nil {
			return e
		}
		return write(map[string]any{"state": "idle", "cleanup": "complete"})
	}}
	run.Flags().StringVar(&key, "profile", "", "Profile ID or unique name; defaults to selected profile")
	run.Flags().StringVar(&mode, "mode", "proxy", "Operating mode (only proxy is supported)")
	run.Flags().DurationVar(&duration, "duration", 0, "Session lifetime after readiness, e.g. 10m")
	run.Flags().UintVar(&port, "listen-port", 2080, "Loopback mixed HTTP/SOCKS listener port")
	run.Flags().StringVar(&authFile, "proxy-auth-file", "", "Private JSON file containing local listener username and password")
	root.AddCommand(run)
	root.AddCommand(lifecycleCommands(o, service, write, in, out)...)
	root.AddCommand(&cobra.Command{Use: "status", Short: "Query the live supervisor; readiness does not prove upstream connectivity", Args: cobra.NoArgs, RunE: func(*cobra.Command, []string) error {
		s, e := service()
		if e != nil {
			return e
		}
		v, e := s.Status()
		if e != nil {
			return e
		}
		return write(v)
	}})
	root.AddCommand(&cobra.Command{Use: "stop", Short: "Request safe stop; repeated calls are safe", Args: cobra.NoArgs, RunE: func(*cobra.Command, []string) error {
		s, e := service()
		if e != nil {
			return e
		}
		if e = s.Stop(); e != nil {
			return e
		}
		return write(map[string]string{"state": "idle", "cleanup": "complete"})
	}})
	root.AddCommand(&cobra.Command{Use: "doctor", Short: "Check private storage and verified core compatibility without connecting", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		s, e := service()
		if e != nil {
			return e
		}
		v, e := s.Doctor(cmd.Context())
		if e != nil {
			return e
		}
		return write(v)
	}})
	root.SetContext(context.Background())
	completion := &cobra.Command{Use: "completion SHELL", Short: "Generate shell completion without modifying shell files", ValidArgs: []string{"bash", "zsh", "fish", "powershell"}, Args: cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs), RunE: func(_ *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return root.GenBashCompletionV2(out, true)
		case "zsh":
			return root.GenZshCompletion(out)
		case "fish":
			return root.GenFishCompletion(out, true)
		case "powershell":
			return root.GenPowerShellCompletionWithDesc(out)
		}
		return nil
	}}
	root.AddCommand(completion)
	return root
}
func commandContext(cmd *cobra.Command) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(cmd.Context(), platform.StopSignals()...)
}
