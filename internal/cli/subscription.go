package cli

import (
	"github.com/spf13/cobra"
	"io"
	"tunnelctl/internal/app"
	"tunnelctl/internal/domain"
)

func subscriptionCommand(service func() (*app.Service, error), write func(any) error, in io.Reader) *cobra.Command {
	root := &cobra.Command{Use: "subscription", Short: "Manage protected HTTPS subscription sources"}
	var name, caFile string
	var stdin bool
	var cidrs []string
	add := &cobra.Command{Use: "add", Short: "Add a source without downloading; its URL is never displayed", Args: cobra.NoArgs, RunE: func(*cobra.Command, []string) error {
		if !stdin {
			return domain.Fail(2, "Subscription URL input requires --url-stdin.", "Pipe a private URL file into stdin; never put a secret URL in command arguments.")
		}
		s, e := service()
		if e != nil {
			return e
		}
		v, e := s.Subscriptions().Add(name, in, cidrs, caFile)
		if e != nil {
			return e
		}
		return write(v)
	}}
	add.Flags().StringVar(&name, "name", "", "Unique display name")
	add.MarkFlagRequired("name")
	add.Flags().BoolVar(&stdin, "url-stdin", false, "Read the secret HTTPS URL from stdin")
	add.Flags().StringSliceVar(&cidrs, "allow-private-cidr", nil, "Explicit private source IP/CIDR exception (repeatable)")
	add.Flags().StringVar(&caFile, "ca-file", "", "Private PEM CA file for a controlled HTTPS source")
	root.AddCommand(add)
	root.AddCommand(&cobra.Command{Use: "list", Short: "List redacted sources", Args: cobra.NoArgs, RunE: func(*cobra.Command, []string) error {
		s, e := service()
		if e != nil {
			return e
		}
		v, e := s.Subscriptions().List()
		if e != nil {
			return e
		}
		return write(v)
	}})
	root.AddCommand(&cobra.Command{Use: "show SUBSCRIPTION", Short: "Show metadata without its secret URL or validators", Args: cobra.ExactArgs(1), RunE: func(_ *cobra.Command, args []string) error {
		s, e := service()
		if e != nil {
			return e
		}
		v, e := s.Subscriptions().Show(args[0])
		if e != nil {
			return e
		}
		return write(v)
	}})
	root.AddCommand(&cobra.Command{Use: "rename SUBSCRIPTION NAME", Short: "Rename a source without changing its identity", Args: cobra.ExactArgs(2), RunE: func(_ *cobra.Command, args []string) error {
		s, e := service()
		if e != nil {
			return e
		}
		v, e := s.Subscriptions().Rename(args[0], args[1])
		if e != nil {
			return e
		}
		return write(v)
	}})
	root.AddCommand(&cobra.Command{Use: "remove SUBSCRIPTION", Short: "Remove the source and retain its profiles as standalone profiles", Args: cobra.ExactArgs(1), RunE: func(_ *cobra.Command, args []string) error {
		s, e := service()
		if e != nil {
			return e
		}
		if e = s.Subscriptions().Remove(args[0]); e != nil {
			return e
		}
		return write(map[string]any{"removed": true, "profiles": "retained as standalone"})
	}})
	var preview, removals bool
	var token string
	update := &cobra.Command{Use: "update SUBSCRIPTION", Short: "Fetch, validate, preview or atomically commit a snapshot", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		s, e := service()
		if e != nil {
			return e
		}
		ctx, cancel := commandContext(cmd)
		defer cancel()
		v, e := s.Subscriptions().Update(ctx, args[0], preview, removals, token)
		if e != nil {
			return e
		}
		return write(v)
	}}
	update.Flags().BoolVar(&preview, "preview", false, "Preview without changing profiles or HTTP validators")
	update.Flags().BoolVar(&removals, "accept-removals", false, "Explicitly accept removed unselected profiles")
	update.Flags().StringVar(&token, "if-preview", "", "Require the source to match a previous preview token")
	root.AddCommand(update)
	return root
}
