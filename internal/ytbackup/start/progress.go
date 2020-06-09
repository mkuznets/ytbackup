package start

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"strings"
	"time"

	stdlog "log"

	"github.com/hpcloud/tail"
	"github.com/rs/zerolog/log"
	"golang.org/x/time/rate"
	"mkuznets.com/go/ytbackup/internal/utils"
)

const (
	marker           = "__progress__"
	progressInterval = 5 * time.Second
)

type Progress struct {
	Total      uint64
	Downloaded uint64
	Done       string
	Finished   bool
}

func trackProgress(ctx context.Context, path string) {
	nopLogger := stdlog.New(ioutil.Discard, "", 0)

	t, err := tail.TailFile(path, tail.Config{Follow: true, Logger: nopLogger})
	if err != nil {
		log.Warn().Err(err).Msg("Could not track progress")
		return
	}
	limiter := rate.NewLimiter(rate.Every(progressInterval), 1)

Loop:
	for {
		select {
		case line := <-t.Lines:
			if ctx.Err() != nil {
				break Loop
			}

			if line.Err != nil {
				log.Warn().Err(err).Msg("Tail error")
				break Loop
			}
			if !strings.Contains(line.Text, marker) {
				continue
			}

			raw := strings.Split(line.Text, marker)[1]
			var progress Progress
			if err := json.NewDecoder(strings.NewReader(raw)).Decode(&progress); err != nil {
				continue
			}

			if limiter.Allow() || progress.Finished {
				ev := log.Info().Str("done", progress.Done)
				if progress.Downloaded > 0 {
					ev = ev.Str("downloaded", utils.IBytes(progress.Downloaded))
				}
				if progress.Total > 0 {
					ev = ev.Str("total", utils.IBytes(progress.Total))
				}
				ev.Msg("Progress")
			}

		case <-ctx.Done():
			break Loop
		}
	}

	if err := t.Stop(); err != nil {
		log.Debug().Err(err).Msg("Could not close tail")
	}
	t.Cleanup()
}
