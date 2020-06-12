package start

import (
	"context"

	"github.com/rs/zerolog/log"
	"mkuznets.com/go/ytbackup/internal/history"
	"mkuznets.com/go/ytbackup/internal/utils"
)

func (cmd *Command) crawlHistory(ctx context.Context) error {
	bro, err := cmd.Config.Browser.New()
	if err != nil {
		return err
	}

	return utils.RunEveryInterval(ctx, cmd.Config.UpdateInterval, func() error {
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
		return nil
	})
}
