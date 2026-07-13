package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/dtonair/browctl/protocol"
	"github.com/spf13/cobra"
)

func newTabCommand(out io.Writer, jsonOutput *bool) *cobra.Command {
	cmd := &cobra.Command{Use: "tab", Short: "Manage tabs for a running profile"}
	cmd.AddCommand(newTabOpenCommand(out, jsonOutput))
	cmd.AddCommand(newTabListCommand(out, jsonOutput))
	cmd.AddCommand(newTabFocusCommand(out, jsonOutput))
	cmd.AddCommand(newTabCloseCommand(out, jsonOutput))
	return cmd
}

func newTabOpenCommand(out io.Writer, jsonOutput *bool) *cobra.Command {
	var profileName string
	cmd := &cobra.Command{
		Use:   "open --profile NAME URL",
		Short: "Open a new tab",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			raw, err := json.Marshal(map[string]string{"url": args[0]})
			if err != nil {
				return err
			}
			resp, err := requestDaemon(cmd.Context(), protocol.Request{Cmd: "tab.open", Profile: profileName, Args: raw}, true)
			if err != nil {
				return writeProtocolError(out, *jsonOutput, protocol.DaemonUnavailable, err.Error())
			}
			return writeTabResponse(out, *jsonOutput, resp, "")
		},
	}
	cmd.Flags().StringVar(&profileName, "profile", "", "profile name")
	_ = cmd.MarkFlagRequired("profile")
	return cmd
}

func newTabListCommand(out io.Writer, jsonOutput *bool) *cobra.Command {
	var profileName string
	cmd := &cobra.Command{
		Use:   "list --profile NAME",
		Short: "List tabs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := requestDaemon(cmd.Context(), protocol.Request{Cmd: "tab.list", Profile: profileName}, true)
			if err != nil {
				return writeProtocolError(out, *jsonOutput, protocol.DaemonUnavailable, err.Error())
			}
			return writeTabResponse(out, *jsonOutput, resp, "")
		},
	}
	cmd.Flags().StringVar(&profileName, "profile", "", "profile name")
	_ = cmd.MarkFlagRequired("profile")
	return cmd
}

func newTabFocusCommand(out io.Writer, jsonOutput *bool) *cobra.Command {
	var profileName string
	cmd := &cobra.Command{
		Use:   "focus --profile NAME TAB_ID",
		Short: "Focus a tab",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := requestDaemon(cmd.Context(), protocol.Request{Cmd: "tab.focus", Profile: profileName, Tab: args[0]}, true)
			if err != nil {
				return writeProtocolError(out, *jsonOutput, protocol.DaemonUnavailable, err.Error())
			}
			return writeTabResponse(out, *jsonOutput, resp, "")
		},
	}
	cmd.Flags().StringVar(&profileName, "profile", "", "profile name")
	_ = cmd.MarkFlagRequired("profile")
	return cmd
}

func newTabCloseCommand(out io.Writer, jsonOutput *bool) *cobra.Command {
	var profileName string
	cmd := &cobra.Command{
		Use:   "close --profile NAME TAB_ID",
		Short: "Close a tab",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := requestDaemon(cmd.Context(), protocol.Request{Cmd: "tab.close", Profile: profileName, Tab: args[0]}, true)
			if err != nil {
				return writeProtocolError(out, *jsonOutput, protocol.DaemonUnavailable, err.Error())
			}
			return writeTabResponse(out, *jsonOutput, resp, "")
		},
	}
	cmd.Flags().StringVar(&profileName, "profile", "", "profile name")
	_ = cmd.MarkFlagRequired("profile")
	return cmd
}

func writeTabResponse(out io.Writer, jsonOutput bool, resp protocol.Response, human string) error {
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
		if _, err := fmt.Fprintln(out, string(encoded)); err != nil {
			return err
		}
	}
	if !resp.OK {
		return resp.Error
	}
	return nil
}
