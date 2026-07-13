package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/dtonair/browctl/daemon"
	"github.com/dtonair/browctl/paths"
)

func main() {
	var socketPath string
	flag.StringVar(&socketPath, "socket", "", "Unix socket path")
	flag.Parse()

	layout, err := paths.Ensure()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if socketPath == "" {
		socketPath = layout.DaemonSocket
	}

	d := daemon.New(socketPath)
	if err := d.Serve(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
