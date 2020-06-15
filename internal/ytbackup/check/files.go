package check

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"mkuznets.com/go/ytbackup/internal/index"
	"mkuznets.com/go/ytbackup/internal/ytbackup"
)

type FilesCommand struct {
	Hashes bool `long:"hashes"`
	ytbackup.Command
}

func (cmd *FilesCommand) Execute([]string) error {
	sts := map[string]string{}
	for _, st := range cmd.Storages.List() {
		sts[st.ID] = st.Path
	}

	err := cmd.Index.Map(index.StatusDone, func(video *index.Video) error {
		for _, st := range video.Storages {
			path, ok := sts[st.ID]
			if !ok {
				continue
			}
			for _, f := range video.Files {
				filePath := filepath.Join(path, f.Path)

				fi, err := os.Stat(filePath)
				if err != nil {
					log.Err(err).Str("id", video.ID).Str("path", f.Path).Msg("could not stat file")
					continue
				}

				if uint64(fi.Size()) != f.Size {
					log.Err(err).
						Str("id", video.ID).
						Str("path", f.Path).
						Int64("fs_size", fi.Size()).
						Uint64("db_size", f.Size).
						Msg("size does not match")
					continue
				}

				if cmd.Hashes {
					fp, err := os.Open(filePath)
					if err != nil {
						log.Err(err).Str("id", video.ID).Str("path", f.Path).Msg("could not open file")
						continue
					}

					h := sha256.New()
					if _, err := io.Copy(h, fp); err != nil {
						log.Err(err).Str("id", video.ID).Str("path", f.Path).Msg("copy error")
						continue
					}

					digest := fmt.Sprintf("%x", h.Sum(nil))

					if digest != f.Hash {
						log.Err(err).Str("id", video.ID).Str("path", f.Path).Msg("hash does not match")
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}
