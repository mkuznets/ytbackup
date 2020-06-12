package start

import (
	"context"

	"github.com/rs/zerolog/log"
	"mkuznets.com/go/ytbackup/internal/utils"
	"mkuznets.com/go/ytbackup/internal/youtube"
)

func (cmd *Command) crawlAPI(ctx context.Context) error {
	videos := make([]string, 0, 50)

	service, err := youtube.NewService(ctx, cmd.Config.Youtube.OAuth.Token())
	if err != nil {
		return err
	}

	return utils.RunEveryInterval(ctx, cmd.Config.UpdateInterval, func() error {
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
		return nil
	})
}
