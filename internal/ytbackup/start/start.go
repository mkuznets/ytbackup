package start

import (
	"github.com/rs/zerolog/log"
	"mkuznets.com/go/ytbackup/internal/python"
	"mkuznets.com/go/ytbackup/internal/ytbackup"
)

type Command struct {
	ytbackup.Command
	DisableDownload bool `long:"disable-download" description:"Do not download videos" env:"YTBACKUP_DISABLE_DOWNLOAD"`
	Python          *python.Python
}

func (cmd *Command) Execute([]string) error {
	pyConf := &cmd.Config.Python

	cmd.Python = python.New(
		cmd.Config.Dirs.Python(),
		python.WithPython(pyConf.Executable),
		python.WithYDLUpdateInterval(pyConf.YoutubeDL.UpdateInterval),
		python.WithYDLLite(pyConf.YoutubeDL.Lite),
		python.WithYDLVersion(pyConf.YoutubeDL.Version),
		python.WithYDLOptions(pyConf.YoutubeDL.Options),
	)

	if err := cmd.Python.Init(cmd.CriticalCtx); err != nil {
		return err
	}
	defer cmd.Python.Close()

	if cmd.Config.Sources.History.Enable {
		cmd.Wg.Add(1)
		go func() {
			defer cmd.Wg.Done()
			log.Info().
				Stringer("interval", cmd.Config.Sources.UpdateInterval).
				Msg("Watch history crawler: starting")

			if err := cmd.RunHistoryCrawler(cmd.Ctx); err != nil {
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
				Stringer("interval", cmd.Config.Sources.UpdateInterval).
				Msg("Playlists crawler: starting")

			if err := cmd.RunAPICrawler(cmd.Ctx); err != nil {
				log.Err(err).Msg("Playlists crawler")
				return
			}
			log.Info().Msg("Playlists crawler stopped")
		}()
	}

	if !cmd.DisableDownload {
		log.Info().Msg("Downloader: starting")

		cmd.Wg.Add(1)
		go func() {
			if err := cmd.RunEnqueuer(cmd.Ctx); err != nil {
				log.Err(err).Msg("Enqueuer error")
			}
			cmd.Wg.Done()
		}()

		if err := cmd.RunDownloader(cmd.Ctx); err != nil {
			return err
		}
		log.Info().Msg("Downloader: stopped")
		return nil
	}
	log.Warn().Msg("Downloader is disabled")

	cmd.Wg.Wait()

	return nil
}
