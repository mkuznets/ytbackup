package python

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"mkuznets.com/go/ytbackup/internal/python/ydl"
	"mkuznets.com/go/ytbackup/internal/utils"
)

func (py *Python) ensureYDL(ctx context.Context) (bool, error) {
	log.Info().Msg("Checking for youtube-dl updates")

	currentVersion := ydl.ReadVersion(py.root)
	log.Info().Str("version", currentVersion).Msg("Current youtube-dl")

	release, err := ydl.GetRelease(ctx, py.ydlVersion)
	if err != nil {
		return false, err
	}

	if !ydl.ShallUpgrade(currentVersion, release.Tag) {
		log.Info().Msg("youtube-dl is up to date")
		return false, nil
	}

	if err := utils.RemoveDirs(py.root); err != nil {
		return false, err
	}

	archive, err := release.Download()
	if err != nil {
		return false, err
	}
	if err := ydl.ExtractZIP(archive, py.root); err != nil {
		return false, err
	}

	if py.ydlLite {
		log.Info().Msg("youtube-dl lite is enabled, removing redundant modules")
		if err := py.makeYDLLite(ctx); err != nil {
			return false, err
		}
	}

	if err := ydl.WriteVersion(py.root, release.Tag); err != nil {
		return false, err
	}
	log.Debug().Str("version", release.Tag).Msg("youtube-dl upgraded")

	return true, nil
}

func (py *Python) makeYDLLite(ctx context.Context) error {
	out, err := py.run(ctx, "ydl_lite.py")
	if err != nil {
		return fmt.Errorf("could not create lite version of youtube-dl: %s", out)
	}
	return nil
}
