package start

import (
	"context"

	"github.com/rs/zerolog/log"
	"mkuznets.com/go/ytbackup/internal/ytbackup"
)

type Command struct {
	DisableDownload bool `long:"disable-download" description:"do not download new videos" env:"YTBACKUP_DISABLE_DOWNLOAD"`
	ytbackup.Command
}

func (cmd *Command) Execute([]string) error {
	ctx := context.Background()

	if cmd.Config.Sources.History {
		go func() {
			log.Info().
				Stringer("interval", cmd.Config.UpdateInterval).
				Msg("Watch history crawler: starting")

			if err := cmd.crawlHistory(ctx); err != nil {
				log.Err(err).Msg("Watch history crawler: stopped")
			}
		}()
	}

	if len(cmd.Config.Sources.Playlists) > 0 {
		go func() {
			log.Info().
				Stringer("interval", cmd.Config.UpdateInterval).
				Msg("Playlists crawler: starting")

			if err := cmd.crawlAPI(ctx); err != nil {
				log.Err(err).Msg("Playlists crawler: stopped")
			}
		}()
	}

	if !cmd.DisableDownload {
		log.Debug().Msg("Downloader: starting")
		if err := cmd.Serve(ctx); err != nil {
			return err
		}
		return nil
	}

	log.Warn().Msg("Downloader is disabled")
	<-ctx.Done()
	return nil
}
