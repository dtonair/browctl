package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/dt/browctl/protocol"
	"github.com/dt/browctl/version"
	"github.com/spf13/cobra"
)

type Options struct {
	Out io.Writer
	Err io.Writer
}

func NewBrowctlCommand(opts Options) *cobra.Command {
	if opts.Out == nil {
		opts.Out = io.Discard
	}
	if opts.Err == nil {
		opts.Err = io.Discard
	}

	var jsonOutput bool

	cmd := &cobra.Command{
		Use:           "browctl",
		Short:         "Control persistent Chrome automation profiles",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.SetOut(opts.Out)
	cmd.SetErr(opts.Err)
	cmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "emit machine-readable JSON")

	cmd.AddCommand(newVersionCommand(opts.Out, &jsonOutput))
	cmd.AddCommand(newPingCommand(opts.Out, &jsonOutput))
	cmd.AddCommand(newDaemonCommand(opts.Out, &jsonOutput))
	cmd.AddCommand(newProfileCommand(opts.Out, &jsonOutput))
	cmd.AddCommand(newTabCommand(opts.Out, &jsonOutput))
	cmd.AddCommand(newActionCommands(opts.Out, &jsonOutput)...)

	return cmd
}

func newVersionCommand(out io.Writer, jsonOutput *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print browctl version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			info := version.Get()
			if *jsonOutput {
				return writeJSON(out, protocol.OK(info, protocol.Meta{}))
			}
			_, err := fmt.Fprintf(out, "browctl %s\ncommit: %s\ndate: %s\n", info.Version, info.Commit, info.Date)
			return err
		},
	}
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}
