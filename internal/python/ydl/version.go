package ydl

import (
	"io/ioutil"
	"os"
	"path/filepath"

	semver "github.com/hashicorp/go-version"
)

const (
	versionFilename = "youtube-dl-version"
)

func ReadVersion(path string) string {
	versionPath := filepath.Join(path, versionFilename)
	content, err := ioutil.ReadFile(versionPath)
	if err != nil {
		return "unknown"
	}
	return string(content)
}

func WriteVersion(path, version string) error {
	versionPath := filepath.Join(path, versionFilename)
	return ioutil.WriteFile(versionPath, []byte(version), os.FileMode(0644))
}

func ShallUpgrade(current, latest string) bool {
	cv, err := semver.NewVersion(current)
	if err != nil {
		cv = semver.Must(semver.NewVersion("0.0.0"))
	}
	lv, err := semver.NewVersion(latest)
	if err != nil {
		lv = semver.Must(semver.NewVersion("0.0.1"))
	}
	return !lv.Equal(cv)
}
