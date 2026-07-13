package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/dtonair/browctl/protocol"
	"github.com/spf13/cobra"
)

func newProfileCommand(out io.Writer, jsonOutput *bool) *cobra.Command {
	cmd := &cobra.Command{Use: "profile", Short: "Manage persistent Chrome profiles"}
	cmd.AddCommand(newProfileCreateCommand(out, jsonOutput))
	cmd.AddCommand(newProfileListCommand(out, jsonOutput))
	cmd.AddCommand(newProfileInspectCommand(out, jsonOutput))
	cmd.AddCommand(newProfileDeleteCommand(out, jsonOutput))
	cmd.AddCommand(newProfileStartCommand(out, jsonOutput))
	cmd.AddCommand(newProfileStopCommand(out, jsonOutput))
	return cmd
}

func newProfileCreateCommand(out io.Writer, jsonOutput *bool) *cobra.Command {
	var chromePath string
	cmd := &cobra.Command{
		Use:   "create NAME",
		Short: "Create a persistent Chrome profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			raw, err := json.Marshal(map[string]string{"chrome_path": chromePath})
			if err != nil {
				return err
			}
			resp, err := requestDaemon(cmd.Context(), protocol.Request{Cmd: "profile.create", Profile: args[0], Args: raw}, true)
			if err != nil {
				return writeProtocolError(out, *jsonOutput, protocol.DaemonUnavailable, err.Error())
			}
			return writeProfileResponse(cmd.Context(), out, *jsonOutput, resp, fmt.Sprintf("created %s", args[0]))
		},
	}
	cmd.Flags().StringVar(&chromePath, "chrome-path", "", "Chrome executable path for this profile")
	return cmd
}

func newProfileListCommand(out io.Writer, jsonOutput *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := requestDaemon(cmd.Context(), protocol.Request{Cmd: "profile.list"}, true)
			if err != nil {
				return writeProtocolError(out, *jsonOutput, protocol.DaemonUnavailable, err.Error())
			}
			return writeProfileResponse(cmd.Context(), out, *jsonOutput, resp, "")
		},
	}
}

func newProfileInspectCommand(out io.Writer, jsonOutput *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "inspect NAME",
		Short: "Inspect a profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := requestDaemon(cmd.Context(), protocol.Request{Cmd: "profile.inspect", Profile: args[0]}, true)
			if err != nil {
				return writeProtocolError(out, *jsonOutput, protocol.DaemonUnavailable, err.Error())
			}
			return writeProfileResponse(cmd.Context(), out, *jsonOutput, resp, "")
		},
	}
}

func newProfileDeleteCommand(out io.Writer, jsonOutput *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "delete NAME",
		Short: "Delete a profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := requestDaemon(cmd.Context(), protocol.Request{Cmd: "profile.delete", Profile: args[0]}, true)
			if err != nil {
				return writeProtocolError(out, *jsonOutput, protocol.DaemonUnavailable, err.Error())
			}
			return writeProfileResponse(cmd.Context(), out, *jsonOutput, resp, fmt.Sprintf("deleted %s", args[0]))
		},
	}
}

func newProfileStartCommand(out io.Writer, jsonOutput *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "start NAME",
		Short: "Start Chrome for a profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := requestDaemon(cmd.Context(), protocol.Request{Cmd: "profile.start", Profile: args[0]}, true)
			if err != nil {
				return writeProtocolError(out, *jsonOutput, protocol.DaemonUnavailable, err.Error())
			}
			return writeProfileResponse(cmd.Context(), out, *jsonOutput, resp, fmt.Sprintf("started %s", args[0]))
		},
	}
}

func newProfileStopCommand(out io.Writer, jsonOutput *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "stop NAME",
		Short: "Stop Chrome for a profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := requestDaemon(cmd.Context(), protocol.Request{Cmd: "profile.stop", Profile: args[0]}, true)
			if err != nil {
				return writeProtocolError(out, *jsonOutput, protocol.DaemonUnavailable, err.Error())
			}
			return writeProfileResponse(cmd.Context(), out, *jsonOutput, resp, fmt.Sprintf("stopped %s", args[0]))
		},
	}
}

func writeProfileResponse(_ context.Context, out io.Writer, jsonOutput bool, resp protocol.Response, human string) error {
	if jsonOutput {
		if err := writeJSON(out, resp); err != nil {
			return err
		}
	} else if resp.OK {
		if human != "" {
			_, err := fmt.Fprintln(out, human)
			return err
		}
		encoded, err := json.MarshalIndent(resp.Data, "", "  ")
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(out, string(encoded))
		if err != nil {
			return err
		}
	}
	if !resp.OK {
		return resp.Error
	}
	return nil
}
