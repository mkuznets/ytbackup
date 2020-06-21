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

func (v *Video) Row() string {
	line := fmt.Sprintf("%s\t%s%s\t%s", v.ID, v.Status, v.Meta.row(), v.shortReason())
	line = strings.ReplaceAll(line, "\n", " ")
	return line
}

func (v *Video) shortReason() string {
	r := reContentWarning.ReplaceAllString(v.Reason, "$1")
	r = reYoutubeSaid.ReplaceAllString(r, "$1")
	r = reSpaces.ReplaceAllString(r, " ")
	r = reSorry.ReplaceAllString(r, "$1")
	r = strings.TrimSpace(r)
	return truncate(r, 90)
}

func (meta *Meta) row() string {
	if meta == nil {
		return "\t\t\t"
	}

	return fmt.Sprintf(
		"\t%s\t%s\t%s",
		meta.PublishedAt.Format("2006-01-02"),
		truncate(meta.ChannelTitle, 20),
		truncate(meta.Title, 30),
	)
}

func truncate(s string, n int) string {
	return runewidth.Truncate(s, n, "...")
}
