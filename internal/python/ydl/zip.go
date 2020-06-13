package ydl

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
)

const (
	zipMaxFileSize = 2 * 1024 * 1024 // bytes
)

func ExtractZIP(archive, root string) error {
	log.Info().Str("archive", archive).Msg("Extracting youtube-dl")

	// noinspection GoUnhandledErrorResult
	defer os.Remove(archive)

	reader, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}

	// noinspection GoUnhandledErrorResult
	defer reader.Close()

	if len(reader.File) < 1 {
		return errors.New("release zipball is empty")
	}
	if !reader.File[0].FileInfo().IsDir() {
		return errors.New("expected a nested directory in release zipball")
	}
	zipBaseDir := reader.File[0].Name

	for _, f := range reader.File {
		rel, err := filepath.Rel(zipBaseDir, f.Name)
		if err != nil {
			return err
		}
		if !shallExtract(rel) {
			continue
		}
		target := filepath.Join(root, rel)
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, os.FileMode(0755)); err != nil {
				return fmt.Errorf("could not create directory: %v", err)
			}
			continue
		}
		if err := extractFile(f, target); err != nil {
			return err
		}
	}

	return nil
}

func shallExtract(path string) bool {
	return strings.HasPrefix(path, "youtube_dl") ||
		path == "devscripts" ||
		(strings.HasPrefix(path, "devscripts") && strings.Contains(path, "lazy"))
}

func extractFile(f *zip.File, dst string) error {
	outFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	rc, err := f.Open()
	if err != nil {
		return err
	}

	if _, err := io.CopyN(outFile, rc, zipMaxFileSize); err != nil && err != io.EOF {
		return err
	}

	if err := outFile.Close(); err != nil {
		return err
	}
	if err := rc.Close(); err != nil {
		return err
	}

	return nil
}
