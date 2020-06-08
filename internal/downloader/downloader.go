package downloader

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/rakyll/statik/fs"
	"golang.org/x/time/rate"
	"mkuznets.com/go/ytbackup/internal/index"
	"mkuznets.com/go/ytbackup/internal/pyfs"
	"mkuznets.com/go/ytbackup/internal/venv"
	"mkuznets.com/go/ytbackup/internal/volumes"
)

const (
	networkDowntime  = time.Minute
	ytVideoURLFormat = "https://www.youtube.com/watch?v=%s"
)

type Downloader struct {
	venv    *venv.VirtualEnv
	volumes *volumes.Volumes
}

func New(vs *volumes.Volumes) (*Downloader, error) {
	root := filepath.Join(os.TempDir(), "ytbackup", "venv")

	scriptFS, err := fs.NewWithNamespace(pyfs.Python)
	if err != nil {
		return nil, fmt.Errorf("could not open pyfs: %v", err)
	}

	ve, err := venv.New(root,
		venv.WithFS(scriptFS),
	)
	if err != nil {
		return nil, err
	}

	dl := &Downloader{
		venv:    ve,
		volumes: vs,
	}

	return dl, nil
}

func (dl *Downloader) Download(ctx context.Context, videoID, root string) ([]*Result, error) {
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

	if err := dl.venv.RunScript(rctx, &result, cargs...); err != nil {
		return nil, err
	}
	return result, nil
}

func (dl *Downloader) Serve(ctx context.Context, db *index.Index) error {
	downloadLimiter := rate.NewLimiter(rate.Every(5*time.Second), 3)

	for {
		select {
		default:
			videos, err := db.Pop(1)
			if err != nil {
				return fmt.Errorf("pop: %v", err)
			}

			found := len(videos) > 0

			vol, root := dl.volumes.Root()

			for _, video := range videos {
				log.Info().Str("id", video.ID).Msg("Downloading")

				if err := downloadLimiter.Wait(ctx); err != nil {
					return fmt.Errorf("limiter wait: %v", err)
				}

				results, err := dl.Download(ctx, video.ID, root)
				if err != nil {
					if e, ok := err.(*venv.ScriptError); ok && e.Reason == "network" {
						_ = db.Retry(video.ID, index.RetryInfinite)

						log.Warn().Msgf("Network is down, sleeping for %s", networkDowntime)
						time.Sleep(networkDowntime)
						continue
					}

					log.Err(err).Str("id", video.ID).Msg("Could not download")
					_ = db.Retry(video.ID, index.RetryLimited)
					continue
				}

				for _, res := range results {
					err := func() (err error) {
						relPath, err := filepath.Rel(root, res.File)
						if err != nil {
							return err
						}

						v := &index.Video{
							ID:       res.ID,
							Storages: []index.Storage{{ID: vol, Key: relPath}},
							File:     &index.File{Hash: res.FileHash, Size: res.FileSize},
							Info:     res.Info,
						}
						if err := db.Done(v); err != nil {
							return fmt.Errorf("index error: %v", err)
						}

						if err := res.Cleanup(); err != nil {
							return fmt.Errorf("could not remove temporary files: %v", err)
						}

						return nil
					}()

					if err != nil {
						_ = db.Retry(video.ID, index.RetryLimited)
					}
				}
			}

			if !found {
				time.Sleep(5 * time.Second)
			}

		case <-ctx.Done():
			return nil
		}
	}
}
