package python

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	semver "github.com/hashicorp/go-version"
	"github.com/rs/zerolog/log"
)

const (
	versionFilename = "youtube-dl-version"
	zipMaxFileSize  = 2 * 1024 * 1024
)

type Release struct {
	TagName    string `json:"tag_name"`
	ZipballURL string `json:"zipball_url"`
}

func (py *Python) ensureYDL(ctx context.Context) (err error) {
	py.runLock.Lock()
	defer py.runLock.Unlock()

	log.Info().Msg("Checking for youtube-dl updates")

	currentVersion := py.readYDLVersion()
	if currentVersion != "" {
		log.Info().Str("version", currentVersion).Msg("Current youtube-dl")
	}

	url := "https://api.github.com/repos/ytdl-org/youtube-dl/releases/latest"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	client := http.Client{Timeout: 30 * time.Second}
	r, err := client.Do(req)
	if err != nil {
		return err
	}
	// noinspection GoUnhandledErrorResult
	defer r.Body.Close()

	var release Release
	if err := json.NewDecoder(r.Body).Decode(&release); err != nil {
		return err
	}
	if release.TagName == "" || release.ZipballURL == "" {
		return fmt.Errorf("release info missing tag_name and/or zipball_url")
	}

	log.Info().Str("version", release.TagName).Msg("Latest youtube-dl")

	if !shallUpgrade(currentVersion, release.TagName) {
		log.Info().Msg("youtube-dl is up to date")
		return nil
	}

	if err := os.RemoveAll(filepath.Join(py.root, "youtube_dl")); err != nil {
		return err
	}
	log.Info().Str("tag", release.TagName).Msg("Downloading youtube-dl")

	tmpfile, err := ioutil.TempFile("", "youtube-dl-")
	if err != nil {
		return err
	}
	defer func() {
		if e := os.Remove(tmpfile.Name()); e != nil {
			err = e
		}
	}()

	rz, err := client.Get(release.ZipballURL)
	if err != nil {
		return err
	}
	// noinspection GoUnhandledErrorResult
	defer rz.Body.Close()

	if _, err := io.Copy(tmpfile, rz.Body); err != nil {
		return err
	}

	if err := tmpfile.Sync(); err != nil {
		return err
	}
	if err := tmpfile.Close(); err != nil {
		return err
	}

	log.Info().Str("archive", tmpfile.Name()).Msg("Extracting youtube-dl")

	reader, err := zip.OpenReader(tmpfile.Name())
	if err != nil {
		return err
	}

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
		if !strings.HasPrefix(rel, "youtube_dl") {
			continue
		}
		target := filepath.Join(py.root, rel)
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

	if err := py.writeYDLVersion(release.TagName); err != nil {
		return err
	}
	log.Debug().Str("version", release.TagName).Msg("youtube-dl upgraded")

	return nil
}

func (py *Python) readYDLVersion() string {
	versionPath := filepath.Join(py.root, versionFilename)
	version, err := ioutil.ReadFile(versionPath)
	if err != nil {
		return ""
	}
	return string(version)
}

func (py *Python) writeYDLVersion(version string) error {
	versionPath := filepath.Join(py.root, versionFilename)
	return ioutil.WriteFile(versionPath, []byte(version), os.FileMode(0644))
}

func shallUpgrade(current, latest string) bool {
	cv, err := semver.NewVersion(current)
	if err != nil {
		cv = semver.Must(semver.NewVersion("0.0.0"))
	}
	lv, err := semver.NewVersion(latest)
	if err != nil {
		lv = semver.Must(semver.NewVersion("0.0.1"))
	}
	return lv.GreaterThan(cv)
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
