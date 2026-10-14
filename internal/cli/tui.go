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
		return enterTUI(cmd, o, service, in, out)
	}}
}
func interactive(o *options, in io.Reader, out io.Writer) bool {
	f, ok := in.(*os.File)
	output, outputOK := out.(*os.File)
	return ok && outputOK && !o.json && !o.nonInteractive && os.Getenv("TERM") != "dumb" && term.IsTerminal(f.Fd()) && term.IsTerminal(output.Fd())
}
func enterTUI(cmd *cobra.Command, o *options, service func() (*app.Service, error), in io.Reader, out io.Writer) error {
	if !interactive(o, in, out) {
		return domain.Fail(3, "TUI requires an interactive terminal.", "Use CLI commands with --json or --plain in scripts.")
	}
	s, e := service()
	if e != nil {
		return e
	}
	return tui.Run(cmd.Context(), s, in, out, !o.noColor && !o.plain)
}
