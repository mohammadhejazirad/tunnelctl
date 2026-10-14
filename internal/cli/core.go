package cli

import (
	"github.com/spf13/cobra"
	"tunnelctl/internal/app"
	"tunnelctl/internal/core/singbox"
)

func coreCommand(service func() (*app.Service, error), write func(any) error) *cobra.Command {
	root := &cobra.Command{Use: "core", Short: "Inspect approved external cores; no automatic download or install"}
	root.AddCommand(&cobra.Command{Use: "list", Short: "List approved artifact policy, not installed executables", Args: cobra.NoArgs, RunE: func(*cobra.Command, []string) error {
		return write([]map[string]any{{"name": "sing-box", "version": singbox.Version, "target": "linux/amd64", "sha256": singbox.LinuxAMD64SHA256, "distribution": "external user-installed core"}})
	}})
	root.AddCommand(&cobra.Command{Use: "verify", Short: "Check executable integrity and pinned version without connecting", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		s, e := service()
		if e != nil {
			return e
		}
		ctx, cancel := commandContext(cmd)
		defer cancel()
		v, e := s.VerifyCore(ctx)
		if e != nil {
			return e
		}
		return write(v)
	}})
	return root
}
