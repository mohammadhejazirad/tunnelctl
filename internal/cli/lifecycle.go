package cli

import (
	"github.com/spf13/cobra"
	"io"
	"time"
	"tunnelctl/internal/app"
	"tunnelctl/internal/core"
	"tunnelctl/internal/domain"
)

func lifecycleCommands(o *options, service func() (*app.Service, error), write func(any) error, in io.Reader, out io.Writer) []*cobra.Command {
	var key, authFile, mode string
	var duration time.Duration
	var port uint
	start := &cobra.Command{Use: "start", Short: "Start an explicit detached loopback proxy supervisor", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		if mode != "proxy" {
			return domain.Fail(3, "Only proxy mode is implemented.", "TUN requires an independently verified privileged adapter.")
		}
		if port < 1024 || port > 65535 {
			return domain.Fail(2, "Invalid listener port.", "Use 1024–65535.")
		}
		l, e := app.AuthFile(authFile, core.Listener{Port: uint16(port)})
		if e != nil {
			return e
		}
		s, e := service()
		if e != nil {
			return e
		}
		ctx, cancel := commandContext(cmd)
		defer cancel()
		v, e := s.Start(ctx, key, l, duration)
		if e != nil {
			return e
		}
		return write(v)
	}}
	start.Flags().StringVar(&key, "profile", "", "Profile ID/name; defaults to selection")
	start.Flags().StringVar(&mode, "mode", "proxy", "Operating mode (only proxy is supported)")
	start.Flags().DurationVar(&duration, "duration", 0, "Lifetime after readiness")
	start.Flags().UintVar(&port, "listen-port", 2080, "Loopback listener port")
	start.Flags().StringVar(&authFile, "proxy-auth-file", "", "Private local listener credentials")
	supervisor := &cobra.Command{Use: "__supervise", Hidden: true, Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		ctx, cancel := commandContext(cmd)
		defer cancel()
		return app.RunDetached(ctx, o.core, o.timeout)
	}}
	var execKey string
	child := &cobra.Command{Use: "exec [--profile PROFILE] -- COMMAND [ARGS...]", Short: "Run a child with proxy environment variables in an isolated session", Args: cobra.MinimumNArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		s, e := service()
		if e != nil {
			return e
		}
		ctx, cancel := commandContext(cmd)
		defer cancel()
		childOut := out
		if o.json {
			childOut = cmd.ErrOrStderr()
		}
		if e = s.Exec(ctx, execKey, args, in, childOut, cmd.ErrOrStderr()); e != nil {
			return e
		}
		return write(map[string]any{"child_exit_code": 0, "cleanup": "complete"})
	}}
	child.Flags().StringVar(&execKey, "profile", "", "Profile ID/name; defaults to selection")
	logs := &cobra.Command{Use: "logs", Short: "Read bounded private session state events; raw core logs are unavailable", Args: cobra.NoArgs, RunE: func(*cobra.Command, []string) error {
		s, e := service()
		if e != nil {
			return e
		}
		v, e := s.Logs()
		if e != nil {
			return e
		}
		return write(v)
	}}
	return []*cobra.Command{start, supervisor, child, logs}
}
