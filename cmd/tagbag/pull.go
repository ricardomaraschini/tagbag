package main

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
	"go.podman.io/image/v5/copy"
	"go.podman.io/image/v5/signature"
	"go.podman.io/image/v5/transports/alltransports"
	"go.podman.io/image/v5/types"

	"github.com/ricardomaraschini/tagbag/storage"
	"github.com/ricardomaraschini/tagbag/tgz"
)

//go:embed static/pull-usage.txt
var pullUsageText string

var pullCommand = &cli.Command{
	Name:      "pull",
	Usage:     "Pull multiple images into a deduplicated tarball",
	UsageText: pullUsageText,
	Flags: []cli.Flag{
		&cli.StringSliceFlag{
			Name:     "image",
			Aliases:  []string{"i"},
			Usage:    "Images to pull to the tarball",
			Required: true,
		},
		&cli.StringFlag{
			Name:  "temp",
			Usage: "Temporary directory to use",
			Value: "/tmp",
		},
		&cli.StringFlag{
			Name:    "output",
			Aliases: []string{"o"},
			Usage:   "Destination tarball",
			Value:   "./tagbag.tgz",
		},
		&cli.StringFlag{
			Name:  "authfile",
			Usage: "Path of the authentication file",
		},
		&cli.BoolFlag{
			Name:  "insecure",
			Usage: "Ignore TLS certificate errors",
			Value: false,
		},
		&cli.BoolFlag{
			Name:  "all",
			Usage: "Pull all images (manifest lists)",
			Value: false,
		},
	},
	Action: func(c *cli.Context) error {
		basedir := c.String("temp")
		tempdir, err := os.MkdirTemp(basedir, "tagbag-*")
		if err != nil {
			return fmt.Errorf("failed to create %s directory: %w", tempdir, err)
		}
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

		insecure := types.OptionalBoolFalse
		if c.Bool("insecure") {
			insecure = types.OptionalBoolTrue
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
						AuthFilePath:                c.String("authfile"),
						DockerInsecureSkipTLSVerify: insecure,
					},
					DestinationCtx:     &types.SystemContext{},
					ReportWriter:       os.Stdout,
					ImageListSelection: imglist,
				},
			); err != nil {
				return fmt.Errorf("failed copy %s: %w", src, err)
			}
		}
		fmt.Println("Writing file", c.String("output"))
		if err = tgz.Compress(tempdir, c.String("output")); err != nil {
			return fmt.Errorf("failed compress: %w", err)
		}
		return nil
	},
}
