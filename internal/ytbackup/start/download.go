package start

import "C"
import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog/log"
	"mkuznets.com/go/ytbackup/internal/storages"
	"mkuznets.com/go/ytbackup/internal/utils"

	"mkuznets.com/go/ytbackup/internal/index"
	"mkuznets.com/go/ytbackup/internal/venv"
)

const (
	networkDowntime  = time.Minute
	ytVideoURLFormat = "https://www.youtube.com/watch?v=%s"
)

type Result struct {
	ID          string
	File        string
	StoragePath string `json:"storage_path"`
	FileSize    uint64 `json:"filesize"`
	FileHash    string `json:"filehash"`
	OutputDir   string `json:"output_dir"`
	Info        json.RawMessage
}

func (dr *Result) Cleanup() {
	if dr.OutputDir == "" {
		return
	}
	log.Debug().Str("id", dr.ID).Str("path", dr.OutputDir).Msg("Removing temporary files")

	err := os.RemoveAll(dr.OutputDir)
	if err != nil {
		log.Warn().Err(err).Str("path", dr.OutputDir).Msg("Could not remove temporary files")
	}
}

func (cmd *Command) Serve(ctx context.Context) error {
	ticker := time.NewTicker(5 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return nil

		case <-ticker.C:
			storage, err := cmd.Storages.Get()
			if err != nil {
				log.Err(err).Msg("could not find a suitable storage")
				time.Sleep(time.Minute)
				continue
			}
			videos, err := cmd.Index.Pop(1)
			if err != nil {
				log.Err(err).Msg("index: Pop error")
				continue
			}
			cmd.fetchNew(ctx, videos, storage)
		}
	}
}

func (cmd *Command) fetchNew(ctx context.Context, videos []*index.Video, storage *storages.Ready) {
	for _, video := range videos {
		log.Info().Str("id", video.ID).Msg("Downloading")

		results, err := cmd.downloadByID(ctx, video.ID, storage.Path)
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
			log.Info().
				Str("id", res.ID).
				Str("size", utils.IBytes(res.FileSize)).
				Msg("Download completed")

			v := &index.Video{
				ID:       res.ID,
				Storages: []index.Storage{{ID: storage.ID, Path: res.StoragePath}},
				File:     &index.File{Hash: res.FileHash, Size: res.FileSize},
				Info:     res.Info,
			}

			if err := cmd.Index.Done(v); err != nil {
				log.Err(err).Str("id", video.ID).Msg("Index error")
				_ = cmd.Index.Retry(video.ID, index.RetryLimited)
				continue
			}

			log.Info().Str("id", res.ID).Msg("Index updated")
			res.Cleanup()
		}
	}
}

func (cmd *Command) downloadByID(ctx context.Context, videoID, root string) ([]*Result, error) {
	rctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	url := fmt.Sprintf(ytVideoURLFormat, videoID)

	cargs := []string{
		"/dl.py",
		"--log=/tmp/ytbackup/dl.log",
		"download",
		fmt.Sprintf("--root=%s", root),
		"--cache=/tmp/ytbackup/ydl_cache/",
		url,
	}

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