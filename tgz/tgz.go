package tgz

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Compress creates a tar.gz file at target containing the contents of source.
func Compress(source, target string) error {
	tfile, err := os.Create(target)
	if err != nil {
		return fmt.Errorf("failed to create tar file: %w", err)
	}
	defer tfile.Close()
	gzwriter := gzip.NewWriter(tfile)
	defer gzwriter.Close()
	twriter := tar.NewWriter(gzwriter)
	defer twriter.Close()
	return filepath.Walk(
		source,
		func(file string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			header, err := tar.FileInfoHeader(fi, fi.Name())
			if err != nil {
				return fmt.Errorf("failed to create header: %w", err)
			}
			header.Name = filepath.ToSlash(file)
			if err := twriter.WriteHeader(header); err != nil {
				return fmt.Errorf("failed to write header: %w", err)
			}
			if fi.Mode().IsDir() {
				return nil
			}
			fp, err := os.Open(file)
			if err != nil {
				return fmt.Errorf("failed to open file: %w", err)
			}
			defer fp.Close()
			if _, err = io.Copy(twriter, fp); err != nil {
				return fmt.Errorf("failed to write file: %w", err)
			}
			return nil
		},
	)
}
