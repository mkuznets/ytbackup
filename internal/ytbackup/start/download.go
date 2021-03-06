package start

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"mkuznets.com/go/ytbackup/internal/index"
	"mkuznets.com/go/ytbackup/internal/python"
	"mkuznets.com/go/ytbackup/internal/utils"
	"mkuznets.com/go/ytbackup/internal/utils/ticker"
)

const (
	systemErrorDowntime = 2 * time.Minute
	ytVideoURLFormat    = "https://www.youtube.com/watch?v=%s"
)

type videoIDKey struct{}

type Result struct {
	ID    string
	Files []index.File
}

func (cmd *Command) RunDownloader(ctx context.Context) error {
	return ticker.New(5*time.Second).Do(ctx, func() error {
		videos, err := cmd.Index.Pop(1)
		if err != nil {
			log.Err(err).Msg("index: Pop error")
			return nil
		}

		if len(videos) > 0 {
			cmd.download(ctx, videos)
		}
		return nil
	})
}

func (cmd *Command) download(ctx context.Context, videos []*index.Video) {
	storage, err := cmd.Storages.Get()
	if err != nil {
		log.Err(err).Msgf("no suitable storage, sleeping for %s", systemErrorDowntime)
		utils.SleepContext(ctx, systemErrorDowntime)
		return
	}

	for _, video := range videos {
		log.Info().Str("id", video.ID).Msg("Downloading")

		results, err := cmd.downloadByID(video, storage.Path)
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
			if res.ID != video.ID {
				continue
			}

			video.Storages = []index.Storage{{ID: storage.ID}}
			video.Files = res.Files
			video.Status = index.StatusDone

			if err := cmd.Index.Put(video); err != nil {
				log.Err(err).Str("id", video.ID).Msg("Index error")
				continue
			}

			log.Info().
				Str("id", res.ID).
				Msg("Download complete")
		}
	}
}

func (cmd *Command) downloadByID(video *index.Video, rootDir string) ([]*Result, error) {
	ctx, cancel := context.WithCancel(cmd.CriticalCtx)
	defer cancel()

	ctx = context.WithValue(ctx, videoIDKey{}, video.ID)

	published := video.Meta.PublishedAt
	destDir := filepath.Join(
		rootDir,
		published.Format("2006"),
		published.Format("01"),
		fmt.Sprintf("%s_%s", published.Format("20060102"), video.ID),
	)

	logPath := filepath.Join(
		cmd.Config.Dirs.Logs(),
		fmt.Sprintf("%s_%s.log", time.Now().Format("2006-01-02T15-04-05"), video.ID),
	)

	cargs := []string{
		"dl.py",
		"--log=" + logPath,
		"--root=" + rootDir,
		"--cache=" + cmd.Config.Dirs.Cache,
		"--dst=" + destDir,
		fmt.Sprintf(ytVideoURLFormat, video.ID),
	}

	go func() {
		ticker.New(30*time.Second).MustDo(ctx, func() error {
			cmd.Index.Beat(video.ID)
			return nil
		})
	}()
	go trackProgress(ctx, cancel, logPath)

	var result []*Result
	if err := cmd.Python.RunScript(ctx, &result, cargs...); err != nil {
		return nil, err
	}

	return result, nil
}

func isSystemError(err error) bool {
	e, ok := err.(*python.ScriptError)
	return ok && e.Reason == "system"
}

func isRetriable(err error) bool {
	text := err.Error()
	if strings.Contains(text, "video is private") ||
		strings.Contains(text, "no longer available") ||
		strings.Contains(text, "not available") ||
		strings.Contains(text, "video has been removed") ||
		strings.Contains(text, "copyright grounds") ||
		strings.Contains(text, "in your country") ||
		strings.Contains(text, "confirm your age") ||
		strings.Contains(text, "recording is not available") ||
		strings.Contains(text, "video is unavailable") {
		return false
	}
	return true
}
