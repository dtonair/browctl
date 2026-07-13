package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/dt/browctl/protocol"
	"github.com/spf13/cobra"
)

func newActionCommands(out io.Writer, jsonOutput *bool) []*cobra.Command {
	return []*cobra.Command{
		newGotoCommand(out, jsonOutput),
		newClickCommand(out, jsonOutput),
		newFillCommand(out, jsonOutput),
		newTextCommand(out, jsonOutput),
		newWaitCommand(out, jsonOutput),
	}
}

func actionFlags(cmd *cobra.Command, profileName, tabID *string) {
	cmd.Flags().StringVar(profileName, "profile", "", "profile name")
	cmd.Flags().StringVar(tabID, "tab", "active", "tab id or active")
	_ = cmd.MarkFlagRequired("profile")
}

func newGotoCommand(out io.Writer, jsonOutput *bool) *cobra.Command {
	var profileName, tabID, wait string
	cmd := &cobra.Command{
		Use:   "goto --profile NAME URL",
		Short: "Navigate a tab",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			raw, _ := json.Marshal(map[string]string{"url": args[0], "wait": wait})
			resp, err := requestDaemon(cmd.Context(), protocol.Request{Cmd: "page.goto", Profile: profileName, Tab: tabID, Args: raw}, true)
			if err != nil {
				return writeProtocolError(out, *jsonOutput, protocol.DaemonUnavailable, err.Error())
			}
			return writeActionResponse(out, *jsonOutput, resp)
		},
	}
	actionFlags(cmd, &profileName, &tabID)
	cmd.Flags().StringVar(&wait, "wait", "none", "post-action wait policy: none|load|url:<glob>|selector:<css>")
	return cmd
}

func newClickCommand(out io.Writer, jsonOutput *bool) *cobra.Command {
	var profileName, tabID, wait string
	cmd := &cobra.Command{
		Use:   "click --profile NAME SELECTOR",
		Short: "Click an element",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			raw, _ := json.Marshal(map[string]string{"selector": args[0], "wait": wait})
			resp, err := requestDaemon(cmd.Context(), protocol.Request{Cmd: "element.click", Profile: profileName, Tab: tabID, Args: raw}, true)
			if err != nil {
				return writeProtocolError(out, *jsonOutput, protocol.DaemonUnavailable, err.Error())
			}
			return writeActionResponse(out, *jsonOutput, resp)
		},
	}
	actionFlags(cmd, &profileName, &tabID)
	cmd.Flags().StringVar(&wait, "wait", "none", "post-action wait policy")
	return cmd
}

func newFillCommand(out io.Writer, jsonOutput *bool) *cobra.Command {
	var profileName, tabID string
	cmd := &cobra.Command{
		Use:   "fill --profile NAME SELECTOR VALUE",
		Short: "Fill an element",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			raw, _ := json.Marshal(map[string]string{"selector": args[0], "value": args[1]})
			resp, err := requestDaemon(cmd.Context(), protocol.Request{Cmd: "element.fill", Profile: profileName, Tab: tabID, Args: raw}, true)
			if err != nil {
				return writeProtocolError(out, *jsonOutput, protocol.DaemonUnavailable, err.Error())
			}
			return writeActionResponse(out, *jsonOutput, resp)
		},
	}
	actionFlags(cmd, &profileName, &tabID)
	return cmd
}

func newTextCommand(out io.Writer, jsonOutput *bool) *cobra.Command {
	var profileName, tabID string
	cmd := &cobra.Command{
		Use:   "text --profile NAME SELECTOR",
		Short: "Read element text",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			raw, _ := json.Marshal(map[string]string{"selector": args[0]})
			resp, err := requestDaemon(cmd.Context(), protocol.Request{Cmd: "element.text", Profile: profileName, Tab: tabID, Args: raw}, true)
			if err != nil {
				return writeProtocolError(out, *jsonOutput, protocol.DaemonUnavailable, err.Error())
			}
			return writeActionResponse(out, *jsonOutput, resp)
		},
	}
	actionFlags(cmd, &profileName, &tabID)
	return cmd
}

func newWaitCommand(out io.Writer, jsonOutput *bool) *cobra.Command {
	cmd := &cobra.Command{Use: "wait", Short: "Wait for page conditions"}
	cmd.AddCommand(newWaitSelectorCommand(out, jsonOutput))
	cmd.AddCommand(newWaitURLCommand(out, jsonOutput))
	return cmd
}

func newWaitSelectorCommand(out io.Writer, jsonOutput *bool) *cobra.Command {
	var profileName, tabID string
	cmd := &cobra.Command{
		Use:   "selector --profile NAME SELECTOR",
		Short: "Wait for a selector to become visible",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			raw, _ := json.Marshal(map[string]string{"selector": args[0]})
			resp, err := requestDaemon(cmd.Context(), protocol.Request{Cmd: "wait.selector", Profile: profileName, Tab: tabID, Args: raw}, true)
			if err != nil {
				return writeProtocolError(out, *jsonOutput, protocol.DaemonUnavailable, err.Error())
			}
			return writeActionResponse(out, *jsonOutput, resp)
		},
	}
	actionFlags(cmd, &profileName, &tabID)
	return cmd
}

func newWaitURLCommand(out io.Writer, jsonOutput *bool) *cobra.Command {
	var profileName, tabID string
	cmd := &cobra.Command{
		Use:   "url --profile NAME PATTERN",
		Short: "Wait for URL to match a pattern",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			raw, _ := json.Marshal(map[string]string{"pattern": args[0]})
			resp, err := requestDaemon(cmd.Context(), protocol.Request{Cmd: "wait.url", Profile: profileName, Tab: tabID, Args: raw}, true)
			if err != nil {
				return writeProtocolError(out, *jsonOutput, protocol.DaemonUnavailable, err.Error())
			}
			return writeActionResponse(out, *jsonOutput, resp)
		},
	}
	actionFlags(cmd, &profileName, &tabID)
	return cmd
}

func writeActionResponse(out io.Writer, jsonOutput bool, resp protocol.Response) error {
	if jsonOutput {
		if err := writeJSON(out, resp); err != nil {
			return err
		}
	} else if resp.OK {
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
