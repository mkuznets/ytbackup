package index

import (
	"encoding/json"
	"time"
)

type Video struct {
	ID         string          `json:"id"`
	Storages   []Storage       `json:"storages,omitempty"`
	File       *File           `json:"file,omitempty"`
	Deadline   *time.Time      `json:"deadline,omitempty"`
	Attempt    int             `json:"attempt,omitempty"`
	RetryAfter *time.Time      `json:"retry_after,omitempty"`
	Info       json.RawMessage `json:"info,omitempty"`
}

func (v *Video) Key() []byte {
	return []byte(v.ID)
}

type Storage struct {
	ID  string `json:"id"`
	Key string `json:"key"`
}

type File struct {
	Hash string `json:"hash"`
	Size int    `json:"size"`
}
