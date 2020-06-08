package start

import (
	"github.com/rs/zerolog/log"
	"mkuznets.com/go/ytbackup/internal/ytbackup"
)

type Command struct {
	DisableDownload bool `long:"disable-download" description:"do not download new videos" env:"YTBACKUP_DISABLE_DOWNLOAD"`
	ytbackup.Command
}

func (cmd *Command) Execute([]string) error {
	if cmd.Config.Sources.History {
		cmd.Wg.Add(1)
		go func() {
			defer cmd.Wg.Done()
			log.Info().
				Stringer("interval", cmd.Config.UpdateInterval).
				Msg("Watch history crawler: starting")

			if err := cmd.crawlHistory(cmd.Ctx); err != nil {
				log.Err(err).Msg("Watch history crawler")
				return
			}
			log.Info().Msg("Watch history crawler stopped")
		}()
	}

	if len(cmd.Config.Sources.Playlists) > 0 {
		cmd.Wg.Add(1)
		go func() {
			defer cmd.Wg.Done()
			log.Info().
				Stringer("interval", cmd.Config.UpdateInterval).
				Msg("Playlists crawler: starting")

			if err := cmd.crawlAPI(cmd.Ctx); err != nil {
				log.Err(err).Msg("Playlists crawler")
				return
			}
			log.Info().Msg("Playlists crawler stopped")
		}()
	}

	if !cmd.DisableDownload {
		log.Info().Msg("Downloader: starting")
		if err := cmd.Serve(cmd.Ctx); err != nil {
			return err
		}
		log.Info().Msg("Downloader: stopped")
		return nil
	}
	log.Warn().Msg("Downloader is disabled")
	return nil
}
