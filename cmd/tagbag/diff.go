package main

import (
	_ "embed"
	"fmt"
	"os"
	"path"

	"github.com/urfave/cli/v2"

	"github.com/ricardomaraschini/tagbag/storage"
	"github.com/ricardomaraschini/tagbag/tgz"
)

//go:embed static/diff-usage.txt
var diffUsageText string

var diffCommand = &cli.Command{
	Name:      "diff",
	Usage:     "Generates a diff between two tarballs",
	UsageText: diffUsageText,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "temp",
			Usage: "Temporary directory to use",
			Value: "/tmp",
		},
		&cli.StringFlag{
			Name:    "output",
			Aliases: []string{"o"},
			Usage:   "Destination tarball",
			Value:   "./overlay.tgz",
		},
		&cli.StringFlag{
			Name:     "v1",
			Required: true,
			Usage:    "Version 1 of the tarball",
		},
		&cli.StringFlag{
			Name:     "v2",
			Required: true,
			Usage:    "Version 2 of the tarball",
		},
	},
	Action: func(c *cli.Context) error {
		basedir := c.String("temp")
		tempdir, err := os.MkdirTemp(basedir, "tagbag-*")
		if err != nil {
			return fmt.Errorf("failed to create %s directory: %w", tempdir, err)
		}
		defer os.RemoveAll(tempdir)

		srcdir := path.Join(tempdir, "v1")
		if err := os.MkdirAll(srcdir, 0700); err != nil {
			return err
		}
		if err := tgz.Uncompress(c.String("v1"), srcdir); err != nil {
			return fmt.Errorf("failed to uncompress tarball: %w", err)
		}
		tgtdir := path.Join(tempdir, "v2")
		if err := os.MkdirAll(tgtdir, 0700); err != nil {
			return err
		}
		if err := tgz.Uncompress(c.String("v2"), tgtdir); err != nil {
			return fmt.Errorf("failed to uncompress tarball: %w", err)
		}
		v1 := storage.New(srcdir)
		srcfiles, err := v1.Files()
		if err != nil {
			return fmt.Errorf("failed to get files from v1: %w", err)
		}
		fmt.Println("Calculating diff")
		v2 := storage.New(tgtdir)
		for file := range srcfiles {
			fname := path.Base(file)
			if err := v2.DeleteBlob(fname); err != nil {
				return fmt.Errorf("failed to delete blob: %w", err)
			}
		}
		fmt.Println("Writing file", c.String("output"))
		if err := tgz.Compress(tgtdir, c.String("output")); err != nil {
			return fmt.Errorf("failed to compress tarball: %w", err)
		}
		return nil
	},
}
