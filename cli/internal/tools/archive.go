package tools

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func extractArchiveMember(assetPath, assetName, member, dir string) (string, error) {
	dst := filepath.Join(dir, ".archive-member")
	switch {
	case strings.HasSuffix(strings.ToLower(assetName), ".zip"):
		return dst, extractZipMember(assetPath, member, dst)
	case strings.HasSuffix(strings.ToLower(assetName), ".tar.gz"),
		strings.HasSuffix(strings.ToLower(assetName), ".tgz"):
		return dst, extractTarGzMember(assetPath, member, dst)
	default:
		return "", fmt.Errorf("unsupported archive %q (want .zip, .tar.gz or .tgz)", assetName)
	}
}

func extractZipMember(archivePath, member, dst string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	defer func() { _ = r.Close() }()

	found := 0
	for _, file := range r.File {
		if file.Name != member {
			continue
		}
		found++
		if found > 1 {
			return fmt.Errorf("archive member %q appears more than once", member)
		}
		if !file.Mode().IsRegular() {
			return fmt.Errorf("archive member %q is not a regular file", member)
		}
		src, err := file.Open()
		if err != nil {
			return err
		}
		err = writeMember(dst, src)
		_ = src.Close()
		if err != nil {
			return err
		}
	}
	return requireOneMember(member, found)
}

func extractTarGzMember(archivePath, member, dst string) error {
	f, err := os.Open(archivePath) //nolint:gosec // checksum-verified release asset
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("open gzip: %w", err)
	}
	defer func() { _ = gz.Close() }()

	found := 0
	tr := tar.NewReader(gz)
	for {
		header, nextErr := tr.Next()
		if nextErr == io.EOF {
			break
		}
		if nextErr != nil {
			return fmt.Errorf("read tar: %w", nextErr)
		}
		if header.Name != member {
			continue
		}
		found++
		if found > 1 {
			return fmt.Errorf("archive member %q appears more than once", member)
		}
		if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeRegA {
			return fmt.Errorf("archive member %q is not a regular file", member)
		}
		if err := writeMember(dst, tr); err != nil {
			return err
		}
	}
	return requireOneMember(member, found)
}

func writeMember(dst string, src io.Reader) error {
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(f, src)
	closeErr := f.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func requireOneMember(member string, found int) error {
	if found == 0 {
		return fmt.Errorf("archive member %q not found", member)
	}
	return nil
}
