package history

import (
	"context"
	"fmt"

	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/chromedp"
	"mkuznets.com/go/ytbackup/internal/utils"
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

func Videos(ctx context.Context, debugURL string) ([]*Video, error) {
	allocatorContext, cancel := chromedp.NewRemoteAllocator(ctx, debugURL)
	defer cancel()

	ctxt, cancel := chromedp.NewContext(allocatorContext)
	defer cancel()

	var res []byte

	actions := []chromedp.Action{
		chromedp.Navigate(ytHistoryURL),
		chromedp.Evaluate(ytDataKey, &res),
	}
	if err := chromedp.Run(ctxt, actions...); err != nil {
		return nil, err
	}

	if err := chromedp.Run(ctxt, browser.Close()); err != nil {
		return nil, err
	}

	videos := make([]*Video, 0, 200)

	err := utils.ExtractByKey(res, "videoId", func(id string) {
		url := fmt.Sprintf(ytVideoURLFormat, id)
		videos = append(videos, &Video{id, url})
	})
	if err != nil {
		return nil, err
	}

	return videos, nil
}
