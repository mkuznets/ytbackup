package history

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"

	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/chromedp"
)

const (
	ytHistoryURL     = "https://www.youtube.com/feed/history"
	ytVideoURLFormat = "https://www.youtube.com/watch?v=%s"
	ytDataKey        = "ytInitialData"
)

type Video struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

func Videos(ctx context.Context, debugURL string) ([]Video, error) {
	allocatorContext, cancel := chromedp.NewRemoteAllocator(ctx, debugURL)
	defer cancel()

	ctxt, cancel := chromedp.NewContext(allocatorContext)
	defer cancel()

	var res []byte

	if err := chromedp.Run(ctxt,
		chromedp.Navigate(ytHistoryURL),
		chromedp.Evaluate(ytDataKey, &res),
	); err != nil {
		return nil, err
	}

	if err := chromedp.Run(ctxt,
		browser.Close(),
	); err != nil {
		log.Fatal(err)
	}

	return decodeVideos(res), nil
}

func decodeVideos(data []byte) []Video {
	videos := make([]Video, 0)
	lastID := ""

	idKey := false
	dec := json.NewDecoder(bytes.NewBuffer(data))
	for {
		t, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		if v, ok := t.(string); ok {
			if v == "videoId" {
				idKey = true
				continue
			}
			if idKey && lastID != v {
				url := fmt.Sprintf(ytVideoURLFormat, v)
				videos = append(videos, Video{v, url})
				lastID = v
			}
		}
		idKey = false
	}

	return videos
}
