package main

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

// Version is the version of the application. This is set at build time (see
// Makefile).
var Version = "v0.0.0"

var versionCommand = &cli.Command{
	Name:  "version",
	Usage: "Show version information",
	Action: func(c *cli.Context) error {
		fmt.Println(Version)
		return nil
	},
}
