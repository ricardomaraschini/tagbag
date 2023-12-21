package tgz

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Uncompress uncompresses source tgz into target directory.
func Uncompress(source, target string) error {
	fp, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer fp.Close()
	gzreader, err := gzip.NewReader(fp)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzreader.Close()
	treader := tar.NewReader(gzreader)
	for {
		header, err := treader.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to read tar header: %w", err)
		}
		path := filepath.Join(target, header.Name)
		info := header.FileInfo()
		if info.IsDir() {
			if err := os.MkdirAll(path, info.Mode()); err != nil {
				return fmt.Errorf("failed to create dir: %w", err)
			}
			continue
		}
		perm := os.O_CREATE | os.O_TRUNC | os.O_WRONLY
		file, err := os.OpenFile(path, perm, info.Mode())
		if err != nil {
			return fmt.Errorf("failed to open file: %w", err)
		}
		if _, err = io.Copy(file, treader); err != nil {
			file.Close()
			return err
		}
		file.Close()
	}
	return nil
}

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
	walker := func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(fi, fi.Name())
		if err != nil {
			return fmt.Errorf("failed to create header: %w", err)
		}
		header.Name, err = filepath.Rel(source, file)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}
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
	}
	return filepath.Walk(source, walker)
}
