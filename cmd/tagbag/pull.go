package main

import (
	"fmt"
	"os"

	"github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
	"github.com/urfave/cli/v2"

	"tagbag/storage"
	"tagbag/tgz"
)

var pullCommand = &cli.Command{
	Name:  "pull",
	Usage: "Pull multiple images into a deduplicated tarball",
	Flags: []cli.Flag{
		&cli.StringSliceFlag{
			Name:     "image",
			Aliases:  []string{"i"},
			Usage:    "Images to pull to the tarball",
			Required: true,
		},
		&cli.StringFlag{
			Name:    "temp",
			Aliases: []string{"t"},
			Usage:   "Temporary directory to use",
			Value:   "/tmp/tagbag",
		},
		&cli.StringFlag{
			Name:    "destination",
			Aliases: []string{"d"},
			Usage:   "Destination tarball",
			Value:   "./tagbag.tgz",
		},
		&cli.StringFlag{
			Name:  "authfile",
			Usage: "Path of the authentication file",
		},
		&cli.BoolFlag{
			Name:  "all",
			Usage: "Pull all images (manifest lists)",
			Value: false,
		},
	},
	Action: func(c *cli.Context) error {
		tempdir := c.String("temp")
		defer os.RemoveAll(tempdir)
		pol := &signature.Policy{
			Default: signature.PolicyRequirements{
				signature.NewPRInsecureAcceptAnything(),
			},
		}
		polctx, err := signature.NewPolicyContext(pol)
		if err != nil {
			return fmt.Errorf("failed to create policy: %w", err)
		}
		imglist := copy.CopySystemImage
		if c.Bool("all") {
			imglist = copy.CopyAllImages
		}
		storage := storage.New(tempdir)
		for _, src := range c.StringSlice("image") {
			if err := storage.Image(src); err != nil {
				return fmt.Errorf("failed start %s write: %w", src, err)
			}
			withproto := fmt.Sprintf("docker://%s", src)
			srcref, err := alltransports.ParseImageName(withproto)
			if err != nil {
				return fmt.Errorf("failed parse %s transport: %w", src, err)
			}
			fmt.Println("Pulling", src)
			if _, err := copy.Image(
				c.Context,
				polctx,
				storage,
				srcref,
				&copy.Options{
					SourceCtx: &types.SystemContext{
						AuthFilePath: c.String("authfile"),
					},
					DestinationCtx:     &types.SystemContext{},
					ReportWriter:       os.Stdout,
					ImageListSelection: imglist,
				},
			); err != nil {
				return fmt.Errorf("failed copy %s: %w", src, err)
			}
		}
		fmt.Println("Writing file", c.String("destination"))
		if err = tgz.Compress(tempdir, c.String("destination")); err != nil {
			return fmt.Errorf("failed compress: %w", err)
		}
		return nil
	},
}
