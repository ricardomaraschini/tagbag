package main

import (
	"fmt"
	"os"
	"path"
	"tagbag/storage"
	"tagbag/tgz"

	"github.com/urfave/cli/v2"
)

var diffCommand = &cli.Command{
	Name:  "diff",
	Usage: "Generates a diff between two tarballs",
	Flags: []cli.Flag{
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
			Value:   "./overlay.tgz",
		},
		&cli.StringFlag{
			Name:     "source",
			Required: true,
			Usage:    "Source tarball",
		},
		&cli.StringFlag{
			Name:     "target",
			Required: true,
			Usage:    "Target tarball",
		},
	},
	Action: func(c *cli.Context) error {
		tempdir := c.String("temp")
		defer os.RemoveAll(tempdir)
		srcdir := path.Join(tempdir, "source")
		if err := os.MkdirAll(srcdir, 0700); err != nil {
			return err
		}
		if err := tgz.Uncompress(c.String("source"), srcdir); err != nil {
			return fmt.Errorf("failed to uncompress tarball: %w", err)
		}
		tgtdir := path.Join(tempdir, "target")
		if err := os.MkdirAll(tgtdir, 0700); err != nil {
			return err
		}
		if err := tgz.Uncompress(c.String("target"), tgtdir); err != nil {
			return fmt.Errorf("failed to uncompress tarball: %w", err)
		}
		source := storage.New(srcdir)
		srcfiles, err := source.Files()
		if err != nil {
			return fmt.Errorf("failed to get files from source: %w", err)
		}
		fmt.Println("Calculating diff")
		target := storage.New(tgtdir)
		for file := range srcfiles {
			fname := path.Base(file)
			if err := target.DeleteBlob(fname); err != nil {
				return fmt.Errorf("failed to delete blob: %w", err)
			}
		}
		fmt.Println("Writing file", c.String("destination"))
		if err := tgz.Compress(tgtdir, c.String("destination")); err != nil {
			return fmt.Errorf("failed to compress tarball: %w", err)
		}
		return nil
	},
}
