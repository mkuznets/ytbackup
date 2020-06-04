package history

import (
	"context"

	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/chromedp"
	"mkuznets.com/go/ytbackup/internal/utils"
)

const (
	ytHistoryURL = "https://www.youtube.com/feed/history"
	ytDataKey    = "ytInitialData"
)

func Videos(ctx context.Context, debugURL string) ([]string, error) {
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

	videos := make([]string, 0, 200)

	err := utils.ExtractByKey(res, "videoId", func(id string) {
		videos = append(videos, id)
	})
	if err != nil {
		return nil, err
	}

	return videos, nil
}
