package start

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/hpcloud/tail"
	"github.com/rs/zerolog/log"
	"golang.org/x/time/rate"
	"mkuznets.com/go/ytbackup/internal/utils"
)

const (
	marker           = "__progress__"
	progressInterval = 5 * time.Second
	idleTimeout      = 3 * time.Minute

	leftWarning = time.Minute
)

type Progress struct {
	Total      uint64
	Downloaded uint64
	Done       string
	Finished   bool
}

func trackProgress(ctx context.Context, cancel context.CancelFunc, path string) {
	logCtx := log.With()
	if videoID := ctx.Value(videoIDKey{}); videoID != nil {
		logCtx = logCtx.Str("id", videoID.(string))
	}
	logger := logCtx.Logger()

	cfg := tail.Config{Follow: true, Logger: tail.DiscardingLogger}

	mainLog, err := tail.TailFile(path, cfg)
	if err != nil {
		logger.Warn().Err(err).Msg("Could not track progress")
		return
	}

	ffmpegPath := strings.Replace(path, ".log", "-ffmpeg.log", 1)
	ffmpegLog, err := tail.TailFile(ffmpegPath, cfg)
	if err != nil {
		logger.Warn().Err(err).Msg("Could not track progress")
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
				logger.Error().Msg("Stopping idle download")
				cancel()
				return
			}
			if left <= leftWarning {
				logger.Warn().Msgf("Download is idle, will be stopped in %s", left.Truncate(time.Second))
			}
			sleep := 10 * time.Second
			if left < sleep {
				sleep = left
			}
			utils.SleepContext(ctx, sleep)
		}
	}(&lastEvent)

Loop:
	for {
		select {
		case <-ffmpegLog.Lines:
			lastEvent = time.Now()

		case line := <-mainLog.Lines:
			if ctx.Err() != nil {
				break Loop
			}
			if line.Err != nil {
				logger.Warn().Err(err).Msg("Tail error")
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
				ev := logger.Info().Str("pc", progress.Done)
				if progress.Downloaded > 0 {
					ev = ev.Str("dl", utils.IBytes(progress.Downloaded))
				}
				if progress.Total > 0 {
					ev = ev.Str("tot", utils.IBytes(progress.Total))
				}
				ev.Msg("Progress")
			}

		case <-ctx.Done():
			break Loop
		}
	}

	for _, t := range [2]*tail.Tail{mainLog, ffmpegLog} {
		if err := t.Stop(); err != nil {
			logger.Debug().Err(err).Msg("Could not close tail")
		}
		t.Cleanup()
	}
}
