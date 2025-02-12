package main

import (
	_ "embed"
	"fmt"
	"os"
	"path"

	"github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
	"github.com/urfave/cli/v2"

	"github.com/ricardomaraschini/tagbag/storage"
	"github.com/ricardomaraschini/tagbag/tgz"
)

//go:embed static/push-usage.txt
var pushUsageText string

var pushCommand = &cli.Command{
	Name:      "push",
	Usage:     "Pushes multiple images from a deduplicated tarball",
	UsageText: pushUsageText,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "temp",
			Usage: "Temporary directory to use",
			Value: "/tmp",
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
		&cli.StringFlag{
			Name:  "authfile",
			Usage: "Path of the authentication file",
		},
		&cli.StringSliceFlag{
			Name:    "overlay",
			Aliases: []string{"o"},
			Usage:   "Overlay tarball paths",
		},
		&cli.BoolFlag{
			Name:  "insecure",
			Usage: "Ignore TLS certificate errors",
			Value: false,
		},
	},
	Action: func(c *cli.Context) error {
		pol := &signature.Policy{
			Default: signature.PolicyRequirements{
				signature.NewPRInsecureAcceptAnything(),
			},
		}

		polctx, err := signature.NewPolicyContext(pol)
		if err != nil {
			return fmt.Errorf("failed to create policy: %w", err)
		}

		basedir := c.String("temp")
		tempdir, err := os.MkdirTemp(basedir, "tagbag-*")
		if err != nil {
			return fmt.Errorf("failed to create %s directory: %w", tempdir, err)
		}
		defer os.RemoveAll(tempdir)

		if err := tgz.Uncompress(c.String("source"), tempdir); err != nil {
			return fmt.Errorf("failed to uncompress tarball: %w", err)
		}
		for _, overlay := range c.StringSlice("overlay") {
			if err := tgz.Uncompress(overlay, tempdir); err != nil {
				return fmt.Errorf("failed to uncompress overlay: %w", err)
			}
		}
		storage := storage.New(tempdir)
		images, err := storage.Images()
		if err != nil {
			return fmt.Errorf("failed to list images: %w", err)
		}

		insecure := types.OptionalBoolFalse
		if c.Bool("insecure") {
			insecure = types.OptionalBoolTrue
		}

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
					DestinationCtx: &types.SystemContext{
						AuthFilePath:                c.String("authfile"),
						DockerInsecureSkipTLSVerify: insecure,
					},
					SourceCtx:          &types.SystemContext{},
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
