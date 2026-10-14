package cli

import (
	"github.com/spf13/cobra"
	"strconv"
	"time"
	"tunnelctl/internal/app"
	"tunnelctl/internal/domain"
)

func settingsCommand(service func() (*app.Service, error), write func(any) error) *cobra.Command {
	root := &cobra.Command{Use: "settings", Short: "Manage durable defaults; active sessions are unchanged", Args: cobra.NoArgs}
	root.AddCommand(&cobra.Command{Use: "show", Short: "Show persisted defaults and snapshot revision", Args: cobra.NoArgs, RunE: func(*cobra.Command, []string) error {
		s, e := service()
		if e != nil {
			return e
		}
		v, e := s.Settings()
		if e != nil {
			return e
		}
		return write(v)
	}})
	root.AddCommand(&cobra.Command{Use: "set KEY VALUE", Short: "Atomically persist one default (never reconnects)", Long: "Supported keys: listener-port, duration, auth-file, core-path, timeout, color, probe-target, preferred-mode, core, subscription-policy, listener-address. File paths must be absolute. Empty values clear optional paths/target.", Args: cobra.ExactArgs(2), RunE: func(_ *cobra.Command, args []string) error {
		s, e := service()
		if e != nil {
			return e
		}
		v, e := s.EditSettings(nil, func(p *domain.Preferences) error { return setPreference(p, args[0], args[1]) })
		if e != nil {
			return e
		}
		return write(v)
	}})
	var confirm bool
	reset := &cobra.Command{Use: "reset", Short: "Reset preferences only; retain profiles, sources and selection", Args: cobra.NoArgs, RunE: func(*cobra.Command, []string) error {
		if !confirm {
			return domain.Fail(2, "Reset requires explicit confirmation.", "Use settings reset --yes; profiles and subscription secrets will remain.")
		}
		s, e := service()
		if e != nil {
			return e
		}
		v, e := s.EditSettings(nil, func(p *domain.Preferences) error { *p = domain.DefaultPreferences(); return nil })
		if e != nil {
			return e
		}
		return write(v)
	}}
	reset.Flags().BoolVar(&confirm, "yes", false, "Confirm resetting preferences only")
	root.AddCommand(reset)
	return root
}

func setPreference(p *domain.Preferences, key, value string) error {
	var e error
	switch key {
	case "listener-port":
		var n uint64
		n, e = strconv.ParseUint(value, 10, 16)
		p.ListenerPort = uint16(n)
	case "duration":
		p.Duration = value
	case "auth-file":
		p.AuthFile = value
	case "core-path":
		p.CorePath = value
	case "timeout":
		p.Timeout = value
	case "color":
		p.Color, e = strconv.ParseBool(value)
	case "probe-target":
		p.ProbeTarget = value
	case "preferred-mode":
		p.PreferredMode = value
	case "core":
		p.Core = value
	case "subscription-policy":
		p.SubscriptionPolicy = value
	case "listener-address":
		p.ListenerAddress = value
	default:
		return domain.Fail(2, "Unknown settings key.", "Run settings set --help for supported keys.")
	}
	if e != nil {
		return domain.Fail(2, "Invalid settings value.", "Use a valid port or boolean; run settings set --help.")
	}
	return p.Validate()
}

func sessionDefaults(cmd *cobra.Command, s *app.Service, port uint, duration time.Duration, auth, mode string) (uint, time.Duration, string, string, error) {
	v, e := s.Settings()
	if e != nil {
		return 0, 0, "", "", e
	}
	p := v.Preferences
	if !cmd.Flags().Changed("listen-port") {
		port = uint(p.ListenerPort)
	}
	if !cmd.Flags().Changed("duration") {
		duration, _ = time.ParseDuration(p.Duration)
	}
	if !cmd.Flags().Changed("proxy-auth-file") {
		auth = p.AuthFile
	}
	if !cmd.Flags().Changed("mode") {
		mode = p.PreferredMode
	}
	return port, duration, auth, mode, nil
}
