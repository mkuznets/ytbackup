package ydl

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

type Release struct {
	Tag        string `json:"tag_name"`
	ZipballURL string `json:"zipball_url"`
}

func GetRelease(ctx context.Context, version string) (*Release, error) {
	log.Info().Str("version", version).Msg("Fetching youtube-dl release")

	url := "https://api.github.com/repos/ytdl-org/youtube-dl/releases/latest"
	if version != "latest" {
		url = fmt.Sprintf("https://api.github.com/repos/ytdl-org/youtube-dl/releases/tags/%s", version)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	client := http.Client{Timeout: 30 * time.Second}

	r, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if err := checkResponse(r); err != nil {
		return nil, err
	}

	// noinspection GoUnhandledErrorResult
	defer r.Body.Close()

	var release Release
	if err := json.NewDecoder(r.Body).Decode(&release); err != nil {
		return nil, err
	}
	if release.Tag == "" || release.ZipballURL == "" {
		return nil, fmt.Errorf("release info missing tag_name and/or zipball_url")
	}

	if version == "latest" {
		log.Info().Str("version", release.Tag).Msg("Latest youtube-dl")
	}

	return &release, nil
}

func (release *Release) Download() (string, error) {
	log.Info().Str("tag", release.Tag).Msg("Downloading youtube-dl")

	tmpfile, err := ioutil.TempFile("", "youtube-dl-")
	if err != nil {
		return "", err
	}

	rz, err := http.Get(release.ZipballURL)
	if err != nil {
		return "", err
	}
	if err := checkResponse(rz); err != nil {
		return "", err
	}

	// noinspection GoUnhandledErrorResult
	defer rz.Body.Close()

	if _, err := io.Copy(tmpfile, rz.Body); err != nil {
		return "", err
	}

	if err := tmpfile.Sync(); err != nil {
		return "", err
	}
	if err := tmpfile.Close(); err != nil {
		return "", err
	}

	return tmpfile.Name(), nil
}

func checkResponse(r *http.Response) error {
	if c := r.StatusCode; c >= 200 && c <= 299 {
		return nil
	}

	if r.StatusCode == http.StatusNotFound {
		return fmt.Errorf("HTTP 404: %s", r.Request.URL)
	}

	var e struct{ Message string }

	data, err := ioutil.ReadAll(r.Body)
	if err == nil && data != nil {
		if err := json.Unmarshal(data, &e); err != nil {
			return fmt.Errorf("could not parse error HTTP %d error: %s", r.StatusCode, data)
		}
	}

	if e.Message != "" {
		return fmt.Errorf("github HTTP %d error: %s", r.StatusCode, e.Message)
	}

	return fmt.Errorf("github HTTP %d error: %s", r.StatusCode, data)
}
