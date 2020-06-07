package downloader

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
)

type Result struct {
	ID         string
	Title      string
	Uploader   string
	UploadDate ISODate `json:"upload_date"`
	File       string
	FileSize   int    `json:"filesize"`
	FileHash   string `json:"filehash"`
	Info       json.RawMessage
}

func (dr *Result) Cleanup() error {
	dir := filepath.Dir(dr.File)
	log.Debug().Str("id", dr.ID).Str("path", dir).Msg("Cleaning up temporary files")
	return os.RemoveAll(dir)
}

func (dr *Result) StoragePath() string {
	return filepath.Join(
		dr.UploadDate.Format("2006"),
		dr.UploadDate.Format("01"),
		filepath.Base(dr.File),
	)
}
