package cli

import (
	"github.com/spf13/cobra"
	"tunnelctl/internal/app"
	"tunnelctl/internal/core"
)

func profileCheckCommand(service func() (*app.Service, error), write func(any) error) *cobra.Command {
	return &cobra.Command{Use: "check PROFILE", Short: "Validate generated configuration with the approved core; do not connect", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		s, e := service()
		if e != nil {
			return e
		}
		ctx, cancel := commandContext(cmd)
		defer cancel()
		if e = s.CheckProfile(ctx, args[0], core.Listener{Port: 2080}); e != nil {
			return e
		}
		return write(map[string]bool{"config_validated": true, "connection_tested": false})
	}}
}
func probeCommand(service func() (*app.Service, error), write func(any) error) *cobra.Command {
	var target, authFile string
	cmd := &cobra.Command{Use: "probe", Short: "Explicit bounded TLS-verified HTTPS request via the current owned session", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		s, e := service()
		if e != nil {
			return e
		}
		ctx, cancel := commandContext(cmd)
		defer cancel()
		state, e := s.Status()
		if e != nil {
			return e
		}
		result, e := s.ProbeSession(ctx, state.ID, target, authFile)
		if e != nil {
			return e
		}
		return write(result)
	}}
	cmd.Flags().StringVar(&target, "target", "", "Explicit small HTTPS destination without credentials, query or fragment")
	cmd.Flags().StringVar(&authFile, "proxy-auth-file", "", "Private listener authentication file, if used by this session")
	cmd.MarkFlagRequired("target")
	return cmd
}
