package downloader

import (
	"encoding/json"
	"os"

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
	OutputDir  string `json:"output_dir"`
	Info       json.RawMessage
}

func (dr *Result) Cleanup() error {
	if dr.OutputDir == "" {
		return nil
	}
	log.Debug().Str("id", dr.ID).Str("path", dr.OutputDir).Msg("Cleaning up temporary files")
	return os.RemoveAll(dr.OutputDir)
}
