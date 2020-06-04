package downloader

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
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
	log.Printf("[DEBUG] Cleaning up temporary files for %s: %s", dr.ID, dir)
	return os.RemoveAll(dir)
}

func (dr *Result) StoragePath() string {
	return filepath.Join(
		dr.UploadDate.Format("2006"),
		dr.UploadDate.Format("01"),
		filepath.Base(dr.File),
	)
}
