package main

import (
	"fmt"
	"os"

	"github.com/dt/browctl/cli"
	"github.com/dt/browctl/protocol"
)

func main() {
	cmd := cli.NewBrowctlCommand(cli.Options{Out: os.Stdout, Err: os.Stderr})
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(protocol.ExitCode(err))
	}
}
