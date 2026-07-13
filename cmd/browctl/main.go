package main

import (
	"fmt"
	"os"

	"github.com/dtonair/browctl/cli"
	"github.com/dtonair/browctl/protocol"
)

func main() {
	cmd := cli.NewBrowctlCommand(cli.Options{Out: os.Stdout, Err: os.Stderr})
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(protocol.ExitCode(err))
	}
}
