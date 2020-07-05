package index

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/mattn/go-runewidth"
)

var (
	reContentWarning = regexp.MustCompile(`.+Content Warning(.*)`)
	reYoutubeSaid    = regexp.MustCompile(`.+YouTube said:(.*)`)
	reSorry          = regexp.MustCompile(`(.+)Sorry.*`)
	reSpaces         = regexp.MustCompile(`\s+`)
)

func (v *Video) Row(full bool) string {
	trFunc := truncate
	if full {
		trFunc = func(s string, n int) string { return s }
	}

	meta := "\t\t\t"
	if v.Meta != nil {
		meta = fmt.Sprintf(
			"\t%s\t%s\t%s",
			v.Meta.PublishedAt.Format("2006-01-02"),
			trFunc(v.Meta.ChannelTitle, 20),
			trFunc(v.Meta.Title, 30),
		)
	}

	line := fmt.Sprintf("%s\t%s%s\t%s", v.ID, v.Status, meta, trFunc(v.shortReason(), 90))
	line = strings.ReplaceAll(line, "\n", " ")

	return line
}

func (v *Video) shortReason() string {
	if v.Reason == "" {
		return v.Reason
	}
	r := reContentWarning.ReplaceAllString(v.Reason, "$1")
	r = reYoutubeSaid.ReplaceAllString(r, "$1")
	r = reSpaces.ReplaceAllString(r, " ")
	r = reSorry.ReplaceAllString(r, "$1")
	r = strings.TrimSpace(r)
	return r
}

func truncate(s string, n int) string {
	return runewidth.Truncate(s, n, "...")
}
