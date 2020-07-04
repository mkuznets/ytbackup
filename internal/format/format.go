package format

import (
	"encoding/json"
	"fmt"
	"io"

	"mkuznets.com/go/tabwriter"
	"mkuznets.com/go/ytbackup/internal/index"
)

type Formatter interface {
	Put(video *index.Video) error
	Flush() error
}

type Table struct {
	tw *tabwriter.Writer
}

func NewTable(w io.Writer) Formatter {
	return &Table{
		tw: tabwriter.NewWriter(w, 10, 1, 2, ' ', 0),
	}
}

func (t *Table) Put(video *index.Video) error {
	_, err := fmt.Fprintln(t.tw, video.Row())
	return err
}

func (t *Table) Flush() error {
	return t.tw.Flush()
}

type JSON struct {
	output io.Writer
	videos []*index.Video
}

func NewJSON(w io.Writer) Formatter {
	return &JSON{output: w}
}

func (j *JSON) Put(video *index.Video) error {
	j.videos = append(j.videos, video)
	return nil
}

func (j *JSON) Flush() error {
	enc := json.NewEncoder(j.output)
	enc.SetIndent("", "  ")
	return enc.Encode(j.videos)
}
