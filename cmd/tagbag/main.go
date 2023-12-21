package main

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

func main() {
	var app = &cli.App{
		Name:     "tagbag",
		Usage:    "Deduplicated container image tar balls",
		Commands: []*cli.Command{pullCommand, pushCommand},
	}
	if err := app.Run(os.Args); err != nil {
		logrus.Fatal(err)
	}
}
