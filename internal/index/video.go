package index

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/mattn/go-runewidth"
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

var (
	reContentWarning = regexp.MustCompile(`.+Content Warning(.*)`)
	reYoutubeSaid    = regexp.MustCompile(`.+YouTube said:(.*)`)
	reSorry          = regexp.MustCompile(`(.+)Sorry.*`)
	reSpaces         = regexp.MustCompile(`\s+`)
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

func (v *Video) Row() string {
	line := fmt.Sprintf("%s\t%s%s\t%s", v.ID, v.Status, v.Meta.Row(), v.ShortReason())
	line = strings.ReplaceAll(line, "\n", " ")
	return line
}

func (v *Video) ShortReason() string {
	r := reContentWarning.ReplaceAllString(v.Reason, "$1")
	r = reYoutubeSaid.ReplaceAllString(r, "$1")
	r = reSpaces.ReplaceAllString(r, " ")
	r = reSorry.ReplaceAllString(r, "$1")
	r = strings.TrimSpace(r)
	return Truncate(r, 90)
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

func (meta *Meta) Row() string {
	if meta == nil {
		return "\t\t\t"
	}

	return fmt.Sprintf(
		"\t%s\t%s\t%s",
		meta.PublishedAt.Format("2006-01-02"),
		Truncate(meta.ChannelTitle, 20),
		Truncate(meta.Title, 30),
	)
}

func Truncate(s string, n int) string {
	return runewidth.Truncate(s, n, "...")
}
