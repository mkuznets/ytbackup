package start

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/rs/zerolog/log"
	"mkuznets.com/go/ytbackup/internal/index"
	"mkuznets.com/go/ytbackup/internal/storages"
	"mkuznets.com/go/ytbackup/internal/venv"
)

const (
	networkDowntime  = time.Minute
	ytVideoURLFormat = "https://www.youtube.com/watch?v=%s"
)

type Result struct {
	ID        string
	Files     []index.File
	OutputDir string `json:"output_dir"`
	Info      json.RawMessage
}

func (cmd *Command) Serve(ctx context.Context) error {
	for {
		if ctx.Err() != nil {
			return nil
		}

		videos, err := cmd.Index.Pop(1)
		if err != nil {
			log.Err(err).Msg("index: Pop error")
			continue
		}

		if len(videos) > 0 {
			storage, err := cmd.Storages.Get()
			if err != nil {
				log.Err(err).Msg("could not find a suitable storage")
				time.Sleep(time.Minute)
				continue
			}
			cmd.fetchNew(videos, storage)
		}

		if ctx.Err() == nil {
			time.Sleep(5 * time.Second)
		}
	}
}

func (cmd *Command) fetchNew(videos []*index.Video, storage *storages.Ready) {
	for _, video := range videos {
		log.Info().Str("id", video.ID).Msg("Downloading")

		results, err := cmd.downloadByID(video.ID, storage.Path)
		if err != nil {
			if isNetworkError(err) {
				log.Warn().Msgf("Network is down, sleeping for %s", networkDowntime)
				_ = cmd.Index.Retry(video.ID, index.RetryInfinite)
				time.Sleep(networkDowntime)
				continue
			}

			log.Err(err).Str("id", video.ID).Msg("Download error")
			_ = cmd.Index.Retry(video.ID, index.RetryLimited)
			continue
		}

		for _, res := range results {
			v := &index.Video{
				ID:       res.ID,
				Storages: []index.Storage{{ID: storage.ID}},
				Files:    res.Files,
				Info:     res.Info,
			}

			log.Info().Str("id", res.ID).Msg("Updating index")
			if err := cmd.Index.Done(v); err != nil {
				log.Err(err).Str("id", video.ID).Msg("Index error")
				_ = cmd.Index.Retry(video.ID, index.RetryLimited)
				continue
			}

			log.Info().
				Str("id", res.ID).
				Msg("Download complete")
		}
	}
}

func (cmd *Command) downloadByID(videoID, root string) ([]*Result, error) {
	// cmd.Ctx is not used here because it is cancelled on graceful termination.
	// Instead, we want the last download to finish and only then terminate the downloader.
	rctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	url := fmt.Sprintf(ytVideoURLFormat, videoID)

	cacheDir := cmd.Config.Dirs.Cache

	logPath := filepath.Join(
		cmd.Config.Dirs.Logs,
		fmt.Sprintf("%s_%s.log", time.Now().Format("2006-01-02T15-04-05"), videoID),
	)

	cargs := []string{
		"/dl.py",
		"--log=" + logPath,
		"download",
		"--root=" + root,
		"--cache=" + cacheDir,
		url,
	}

	log.Debug().Strs("args", cargs).Msg("Running python")

	go trackProgress(rctx, cancel, logPath)

	var result []*Result
	if err := cmd.Venv.RunScript(rctx, &result, cargs...); err != nil {
		return nil, err
	}

	return result, nil
}

func isNetworkError(err error) bool {
	if e, ok := err.(*venv.ScriptError); ok && e.Reason == "network" {
		return true
	}
	return false
}
