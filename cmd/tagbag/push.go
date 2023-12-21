package main

import (
	"fmt"
	"os"
	"path"

	"github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
	"github.com/urfave/cli/v2"

	"tagbag/storage"
	"tagbag/tgz"
)

var pushCommand = &cli.Command{
	Name:  "push",
	Usage: "Pushes multiple images from a deduplicated tarball",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "temp",
			Aliases: []string{"t"},
			Usage:   "Temporary directory to use",
			Value:   "/tmp/tagbag",
		},
		&cli.StringFlag{
			Name:     "source",
			Required: true,
			Aliases:  []string{"s"},
			Usage:    "Source tarball path",
		},
		&cli.StringFlag{
			Name:     "destination",
			Required: true,
			Aliases:  []string{"d"},
			Usage:    "Destination registry address",
		},
	},
	Action: func(c *cli.Context) error {
		tempdir := c.String("temp")
		pol := &signature.Policy{
			Default: signature.PolicyRequirements{
				signature.NewPRInsecureAcceptAnything(),
			},
		}
		polctx, err := signature.NewPolicyContext(pol)
		if err != nil {
			return fmt.Errorf("failed to create policy: %w", err)
		}
		if err := os.MkdirAll(tempdir, 0700); err != nil {
			return fmt.Errorf("failed to create tempdir: %w", err)
		}
		defer os.RemoveAll(tempdir)
		if err := tgz.Uncompress(c.String("source"), tempdir); err != nil {
			return fmt.Errorf("failed to uncompress tarball: %w", err)
		}
		storage := storage.New(tempdir)
		images, err := storage.Images()
		for _, src := range images {
			if err := storage.Image(src); err != nil {
				return fmt.Errorf("failed to load image: %w", err)
			}
			_, repo := path.Split(src)
			withproto := fmt.Sprintf("docker://%s/%s", c.String("d"), repo)
			dstref, err := alltransports.ParseImageName(withproto)
			if err != nil {
				return fmt.Errorf("failed parse %s transport: %w", src, err)
			}
			fmt.Println("Pushing", src, "to", withproto)
			if _, err := copy.Image(
				c.Context,
				polctx,
				dstref,
				storage,
				&copy.Options{
					SourceCtx:          &types.SystemContext{},
					DestinationCtx:     &types.SystemContext{},
					ReportWriter:       os.Stdout,
					ImageListSelection: copy.CopyAllImages,
				},
			); err != nil {
				return fmt.Errorf("failed copy %s: %w", src, err)
			}
		}
		return nil
	},
}
