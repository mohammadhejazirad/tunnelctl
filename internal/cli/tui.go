package cli

import (
	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"
	"io"
	"os"
	"tunnelctl/internal/app"
	"tunnelctl/internal/domain"
	"tunnelctl/internal/tui"
)

func tuiCommand(o *options, service func() (*app.Service, error), in io.Reader, out io.Writer) *cobra.Command {
	return &cobra.Command{Use: "tui", Short: "Interactive keyboard interface using the same application services", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		f, ok := in.(*os.File)
		if !ok || o.json || o.nonInteractive || os.Getenv("TERM") == "dumb" {
			return domain.Fail(3, "TUI requires an interactive terminal.", "Use CLI commands with --json or --plain in scripts.")
		}
		output, outputOK := out.(*os.File)
		if !term.IsTerminal(f.Fd()) || !outputOK || !term.IsTerminal(output.Fd()) {
			return domain.Fail(3, "TUI input is not a terminal.", "Use the CLI for redirected input.")
		}
		s, e := service()
		if e != nil {
			return e
		}
		return tui.Run(cmd.Context(), s, in, out, !o.noColor && !o.plain)
	}}
}
