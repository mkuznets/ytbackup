package ytbackup

import (
	"context"
	"fmt"
	"log"
	"time"

	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
	"mkuznets.com/go/ytbackup/internal/downloader"
	"mkuznets.com/go/ytbackup/internal/history"
	yt "mkuznets.com/go/ytbackup/internal/youtube"
)

type StartCommand struct {
	DisableDownload bool `long:"disable-download" description:"do not download new videos" env:"YTBACKUP_DISABLE_DOWNLOAD"`
	Command
}

func (cmd *StartCommand) Execute([]string) error {
	ctx := context.Background()

	vs, err := cmd.Config.Destinations.New()
	if err != nil {
		return err
	}

	dl, err := downloader.New(vs)
	if err != nil {
		return err
	}

	if cmd.Config.Sources.History {
		go func() {
			log.Printf("[INFO] Starting watch history crawler")
			if err := cmd.crawlHistory(ctx); err != nil {
				log.Printf("[ERR] Watch history crawler stopped: %v", err)
			}
		}()
	}

	if len(cmd.Config.Sources.Playlists) > 0 {
		go func() {
			log.Printf("[INFO] Starting playlists crawler")
			if err := cmd.crawlAPI(ctx); err != nil {
				log.Printf("[ERR] Playlists crawler stopped: %v", err)
			}
		}()
	}

	if !cmd.DisableDownload {
		log.Printf("[INFO] Starting downloader")
		if err := dl.Serve(ctx, cmd.Index); err != nil {
			return err
		}
		return nil
	}

	log.Printf("[INFO] Downloader is disabled")
	<-ctx.Done()
	return nil
}

func (cmd *StartCommand) createYoutubeService(ctx context.Context) (*youtube.Service, error) {
	token := cmd.Config.Youtube.OAuth.Token()
	tokenSource := yt.NewConfig().TokenSource(ctx, token)

	ytopts := []option.ClientOption{
		option.WithTokenSource(tokenSource),
	}

	service, err := youtube.NewService(ctx, ytopts...)
	if err != nil {
		return nil, fmt.Errorf("could not create youtube client: %v", err)
	}

	return service, nil
}

func (cmd *StartCommand) crawlAPI(ctx context.Context) error {
	videos := make([]string, 0, 50)

	service, err := cmd.createYoutubeService(ctx)
	if err != nil {
		return err
	}

	runEveryInterval(ctx, cmd.Config.UpdateInterval, func() {
		for title, playlistID := range cmd.Config.Sources.Playlists {
			total := 0

			call := service.PlaylistItems.List("contentDetails")
			call = call.PlaylistId(playlistID)
			call = call.MaxResults(50)

			for {
				response, err := call.Do()
				if err != nil {
					log.Printf("[ERR] Youtube API: %v", err)
					break
				}

				videos = videos[:0]
				for _, x := range response.Items {
					videos = append(videos, x.ContentDetails.VideoId)
				}

				n, err := cmd.Index.Push(videos)
				if err != nil {
					log.Printf("[ERR] Playlist `%s`: %v", title, err)
				}

				total += n
				if n == 0 {
					break
				}

				if response.NextPageToken == "" {
					break
				}
				call.PageToken(response.NextPageToken)
			}
			log.Printf("[INFO] Scheduled %d new videos from playlist `%s`", total, title)
		}
	})

	return nil
}

func (cmd *StartCommand) crawlHistory(ctx context.Context) error {
	bro, err := cmd.Config.Browser.New()
	if err != nil {
		return err
	}

	runEveryInterval(ctx, cmd.Config.UpdateInterval, func() {
		log.Printf("[INFO] Checking watch history")

		err := bro.Do(ctx, func(ctx context.Context, url string) error {
			videos, err := history.Videos(ctx, url)
			if err != nil {
				return err
			}

			n, err := cmd.Index.Push(videos)
			if err != nil {
				return err
			}

			log.Printf("[INFO] Scheduled %d new videos from watch history", n)

			return nil
		})

		if err != nil {
			log.Printf("[ERR] Watch history: %v", err)
		}
	})

	return nil
}

func runEveryInterval(ctx context.Context, interval time.Duration, fun func()) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	fun()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fun()
		}
	}
}
