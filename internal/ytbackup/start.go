package ytbackup

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"

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
			log.Debug().
				Stringer("interval", cmd.Config.UpdateInterval).
				Msg("Watch history crawler: starting")

			if err := cmd.crawlHistory(ctx); err != nil {
				log.Err(err).Msg("Watch history crawler: stopped")
			}
		}()
	}

	if len(cmd.Config.Sources.Playlists) > 0 {
		go func() {
			log.Debug().
				Stringer("interval", cmd.Config.UpdateInterval).
				Msg("Playlists crawler: starting")

			if err := cmd.crawlAPI(ctx); err != nil {
				log.Err(err).Msg("Playlists crawler: stopped")
			}
		}()
	}

	if !cmd.DisableDownload {
		log.Debug().Msg("Downloader: starting")
		if err := dl.Serve(ctx, cmd.Index); err != nil {
			return err
		}
		return nil
	}

	log.Warn().Msg("Downloader is disabled")
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
		log.Debug().Msg("Playlists: checking for new videos")

		for title, playlistID := range cmd.Config.Sources.Playlists {
			total := 0

			call := service.PlaylistItems.List("contentDetails")
			call = call.PlaylistId(playlistID)
			call = call.MaxResults(50)

			for {
				response, err := call.Do()
				if err != nil {
					log.Err(err).Msg("Youtube API error")
					break
				}

				videos = videos[:0]
				for _, x := range response.Items {
					videos = append(videos, x.ContentDetails.VideoId)
				}

				n, err := cmd.Index.Push(videos)
				if err != nil {
					log.Err(err).Msgf("Playlist `%s` error", title)
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

			if total > 0 {
				log.Info().
					Str("playlist", title).
					Int("count", total).
					Msg("New videos from playlist")
			}
		}

		log.Debug().Msg("Playlists: done")
	})

	return nil
}

func (cmd *StartCommand) crawlHistory(ctx context.Context) error {
	bro, err := cmd.Config.Browser.New()
	if err != nil {
		return err
	}

	runEveryInterval(ctx, cmd.Config.UpdateInterval, func() {
		log.Debug().Msg("Watch history: checking for new videos")

		err := bro.Do(ctx, func(ctx context.Context, url string) error {
			videos, err := history.Videos(ctx, url)
			if err != nil {
				return err
			}

			n, err := cmd.Index.Push(videos)
			if err != nil {
				return err
			}

			if n > 0 {
				log.Info().Int("count", n).Msg("New videos from watch history")
			}

			return nil
		})

		if err != nil {
			log.Err(err).Msg("Watch history error")
		}

		log.Debug().Msg("Watch history: done")
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
