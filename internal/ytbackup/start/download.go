package start

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"mkuznets.com/go/ytbackup/internal/index"
	"mkuznets.com/go/ytbackup/internal/python"
	"mkuznets.com/go/ytbackup/internal/storages"
	"mkuznets.com/go/ytbackup/internal/utils"
)

const (
	systemErrorDowntime = 2 * time.Minute
	ytVideoURLFormat    = "https://www.youtube.com/watch?v=%s"
)

type Result struct {
	ID        string
	Skipped   string
	Files     []index.File
	OutputDir string `json:"output_dir"`
	Info      json.RawMessage
}

func (cmd *Command) Serve(ctx context.Context) error {
	return utils.RunEveryInterval(ctx, 5*time.Second, func() error {
		videos, err := cmd.Index.Pop(1)
		if err != nil {
			log.Err(err).Msg("index: Pop error")
			return nil
		}

		if len(videos) > 0 {
			storage, err := cmd.Storages.Get()
			if err != nil {
				log.Err(err).Msgf("no suitable storage, sleeping for %s", systemErrorDowntime)
				utils.SleepContext(ctx, systemErrorDowntime)
				return nil
			}
			cmd.fetchNew(ctx, videos, storage)
		}
		return nil
	})
}

func (cmd *Command) fetchNew(ctx context.Context, videos []*index.Video, storage *storages.Ready) {
	for _, video := range videos {
		log.Info().Str("id", video.ID).Msg("Downloading")

		results, err := cmd.downloadByID(video.ID, storage.Path)
		if err != nil {
			log.Err(err).Str("id", video.ID).Msg("Download error")

			if isSystemError(err) {
				log.Warn().Msgf("System error, sleeping for %s", systemErrorDowntime)
				_ = cmd.Index.Retry(video.ID, index.RetryInfinite)
				utils.SleepContext(ctx, systemErrorDowntime)
				continue
			}

			if isRetriable(err) {
				_ = cmd.Index.Retry(video.ID, index.RetryLimited)
			} else {
				log.Info().Str("id", video.ID).Msg("Download failed with non-retriable error")
				video.Status = index.StatusFailed
				video.Reason = err.Error()
				_ = cmd.Index.Put(video)
			}
			continue
		}

		for _, res := range results {
			if res.Skipped != "" {
				log.Info().Str("id", res.ID).Str("reason", res.Skipped).Msg("Download skipped")

				v := &index.Video{ID: res.ID, Reason: res.Skipped}
				if err := cmd.Index.Skip(v); err != nil {
					log.Err(err).Str("id", video.ID).Msg("Index error")
				}
				continue
			}

			v := &index.Video{
				ID:       res.ID,
				Storages: []index.Storage{{ID: storage.ID}},
				Files:    res.Files,
				Info:     res.Info,
			}
			if err := cmd.Index.Done(v); err != nil {
				log.Err(err).Str("id", video.ID).Msg("Index error")
				continue
			}

			log.Info().
				Str("id", res.ID).
				Msg("Download complete")
		}
	}
}

func (cmd *Command) downloadByID(videoID, root string) ([]*Result, error) {
	rctx, cancel := context.WithCancel(cmd.CriticalCtx)
	defer cancel()

	url := fmt.Sprintf(ytVideoURLFormat, videoID)

	cacheDir := cmd.Config.Dirs.Cache

	logPath := filepath.Join(
		cmd.Config.Dirs.Logs(),
		fmt.Sprintf("%s_%s.log", time.Now().Format("2006-01-02T15-04-05"), videoID),
	)

	maxDuration := int(cmd.Config.Sources.MaxDuration.Seconds())

	cargs := []string{
		"dl.py",
		"--log=" + logPath,
		"download",
		"--root=" + root,
		"--cache=" + cacheDir,
		fmt.Sprintf("--max-duration=%d", maxDuration),
		url,
	}
	go trackProgress(rctx, cancel, logPath)

	var result []*Result
	if err := cmd.Python.RunScript(rctx, &result, cargs...); err != nil {
		return nil, err
	}

	return result, nil
}

func isSystemError(err error) bool {
	if e, ok := err.(*python.ScriptError); ok && e.Reason == "system" {
		return true
	}
	return false
}

func isRetriable(err error) bool {
	text := err.Error()
	if strings.Contains(text, "video is private") ||
		strings.Contains(text, "no longer available") ||
		strings.Contains(text, "video has been removed") ||
		strings.Contains(text, "copyright grounds") ||
		strings.Contains(text, "in your country") ||
		strings.Contains(text, "confirm your age") ||
		strings.Contains(text, "video is unavailable") {
		return false
	}
	return true
}
