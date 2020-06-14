package index

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"mkuznets.com/go/ytbackup/internal/utils"
)

type Status string

const (
	StatusNew        Status = "NEW"
	StatusSkipped    Status = "SKIPPED"
	StatusInProgress Status = "INPROGRESS"
	StatusDone       Status = "DONE"
	StatusFailed     Status = "FAILED"
	StatusAny        Status = ""
)

type Video struct {
	ID         string          `json:"id"`
	Status     Status          `json:"status"`
	Storages   []Storage       `json:"storages,omitempty"`
	Files      []File          `json:"file,omitempty"`
	Deadline   *time.Time      `json:"deadline,omitempty"`
	Attempt    int             `json:"attempt,omitempty"`
	RetryAfter *time.Time      `json:"retry_after,omitempty"`
	Info       json.RawMessage `json:"info,omitempty"`
	Reason     string          `json:"reason,omitempty"`
}

func (v *Video) Key() []byte {
	return []byte(v.ID)
}

func (v *Video) StatusKey() []byte {
	return []byte(fmt.Sprintf("%s::%s", v.Status, v.ID))
}

func (v *Video) ClearSystem() {
	v.RetryAfter = nil
	v.Attempt = 0
	v.Deadline = nil
}

func (v *Video) Row() string {
	line := fmt.Sprintf("%s\t%s", v.Status, v.ID)

	switch v.Status {
	case StatusSkipped:
		line += fmt.Sprintf("\t%s", v.Reason)
	case StatusFailed:
		line += fmt.Sprintf("\t%s", v.Reason)
	case StatusDone:
		if v.Info != nil {
			var info InfoShort
			err := json.Unmarshal(v.Info, &info)
			if err == nil {
				line += info.Row()
			}
		}
	}
	line = strings.ReplaceAll(line, "\n", " ")

	return line
}

type Storage struct {
	ID string `json:"id"`
}

type File struct {
	Path string `json:"path"`
	Hash string `json:"hash"`
	Size uint64 `json:"size"`
}

type InfoShort struct {
	Uploader   string
	Title      string
	Duration   int
	UploadDate string `json:"upload_date"`
}

func (info *InfoShort) Row() string {
	uploadDateString := "0000-00-00"
	uploadDate, err := time.Parse("20060102", info.UploadDate)
	if err == nil {
		uploadDateString = uploadDate.Format("2006-01-02")
	}

	return fmt.Sprintf(
		"\t%s\t%s\t%s\t%s",
		uploadDateString,
		utils.FormatDuration(info.Duration),
		info.Uploader,
		info.Title,
	)
}
