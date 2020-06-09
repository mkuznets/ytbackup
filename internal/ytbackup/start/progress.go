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
	idleTimeout      = 3 * time.Minute
	leftWarning      = time.Minute
)

type Progress struct {
	Total      uint64
	Downloaded uint64
	Done       string
	Finished   bool
}

func trackProgress(ctx context.Context, cancel context.CancelFunc, path string) {
	nopLogger := stdlog.New(ioutil.Discard, "", 0)

	t, err := tail.TailFile(path, tail.Config{Follow: true, Logger: nopLogger})
	if err != nil {
		log.Warn().Err(err).Msg("Could not track progress")
		return
	}
	limiter := rate.NewLimiter(rate.Every(progressInterval), 1)

	lastEvent := time.Now()

	go func(last *time.Time) {
		for {
			if ctx.Err() != nil {
				return
			}
			idle := time.Since(*last)
			left := idleTimeout - idle
			if idle >= idleTimeout {
				log.Error().Msg("Stopping idle download")
				cancel()
				return
			}
			if left <= leftWarning {
				log.Warn().Msgf("Download is idle, will be stopped in %s", left.Truncate(time.Second))
			}
			sleep := 10 * time.Second
			if left < sleep {
				sleep = left
			}
			time.Sleep(sleep)
		}
	}(&lastEvent)

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

			lastEvent = time.Now()

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
