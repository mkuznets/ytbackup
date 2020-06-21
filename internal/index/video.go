package index

import (
	"fmt"
	"time"
)

type Status string

const (
	StatusNew        Status = "NEW"
	StatusEnqueued   Status = "ENQUEUED"
	StatusSkipped    Status = "SKIPPED"
	StatusInProgress Status = "INPROGRESS"
	StatusDone       Status = "DONE"
	StatusFailed     Status = "FAILED"
	StatusAny        Status = ""
)

type Video struct {
	ID         string     `json:"id"`
	Status     Status     `json:"status"`
	Storages   []Storage  `json:"storages,omitempty"`
	Files      []File     `json:"file,omitempty"`
	Deadline   *time.Time `json:"deadline,omitempty"`
	Attempt    int        `json:"attempt,omitempty"`
	RetryAfter *time.Time `json:"retry_after,omitempty"`
	Meta       *Meta      `json:"meta,omitempty"`
	Reason     string     `json:"reason,omitempty"`
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

type Storage struct {
	ID string `json:"id"`
}

type File struct {
	Path string `json:"path"`
	Hash string `json:"hash"`
	Size uint64 `json:"size"`
}

type Meta struct {
	Title        string    `json:"title,omitempty"`
	Description  string    `json:"description,omitempty"`
	ChannelID    string    `json:"channel_id,omitempty"`
	ChannelTitle string    `json:"channel_title,omitempty"`
	Tags         []string  `json:"tags,omitempty"`
	PublishedAt  time.Time `json:"published_at,omitempty"`
}
