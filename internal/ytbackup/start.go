package ytbackup

import (
	"context"
	"fmt"
	"log"
	"time"

	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
	"mkuznets.com/go/ytbackup/internal/database"
	"mkuznets.com/go/ytbackup/internal/downloader"
	"mkuznets.com/go/ytbackup/internal/history"
	yt "mkuznets.com/go/ytbackup/internal/youtube"
)

type StartCommand struct {
	Command
}

func (cmd *StartCommand) Execute([]string) error {
	ctx := context.Background()

	vs, err := cmd.Config.Volumes.New()
	if err != nil {
		return err
	}

	dl, err := downloader.New(vs)
	if err != nil {
		return err
	}

	if cmd.Config.Targets.History {
		go func() {
			log.Printf("[INFO] Starting watch history crawler")
			if err := cmd.crawlHistory(ctx); err != nil {
				log.Printf("[ERR] Watch history crawler stopped: %v", err)
			}
		}()
	}

	if len(cmd.Config.Targets.Playlists) > 0 {
		go func() {
			log.Printf("[INFO] Starting playlists crawler")
			if err := cmd.crawlAPI(ctx); err != nil {
				log.Printf("[ERR] Playlists crawler stopped: %v", err)
			}
		}()
	}

	if err := dl.Serve(ctx, cmd.DB); err != nil {
		return err
	}

	return nil
}

func (cmd *StartCommand) crawlAPI(ctx context.Context) error {
	token := cmd.Config.Youtube.Credentials.Token()
	tokenSource := yt.NewConfig().TokenSource(ctx, token)

	service, err := youtube.NewService(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		return fmt.Errorf("could not create youtube client: %v", err)
	}

	videos := make([]string, 0, 50)

	runEveryInterval(ctx, cmd.Config.UpdateInterval, func() {
		for _, playlistID := range cmd.Config.Targets.Playlists {
			total := 0

			call := service.PlaylistItems.List("contentDetails")
			call = call.PlaylistId(playlistID)
			call = call.MaxResults(50)

			for {
				response, err := call.Do()
				if err != nil {
					log.Printf("youtube api error: %v", err)
					break
				}

				videos = videos[:0]
				for _, x := range response.Items {
					videos = append(videos, x.ContentDetails.VideoId)
				}

				n, err := database.InsertMany(ctx, cmd.DB, videos)
				if err != nil {
					log.Printf("[ERR] Playlist %s: %v", playlistID, err)
					break
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
			log.Printf("[INFO] Scheduled %d new videos from playlist %s", total, playlistID)
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

			n, err := database.InsertMany(ctx, cmd.DB, videos)
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
