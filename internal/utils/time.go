package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/senseyeio/duration"
)

func SleepContext(ctx context.Context, d time.Duration) {
	timer := time.NewTimer(d)
	select {
	case <-ctx.Done():
		if !timer.Stop() {
			<-timer.C
		}
		return
	case <-timer.C:
		timer.Stop()
		return
	}
}

func FormatDuration(seconds int) string {
	d := time.Duration(seconds) * time.Second
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

func ParseISO8601(s string) (time.Duration, error) {
	d, err := duration.ParseISO8601(s)
	if err != nil {
		return 0, err
	}

	var dur time.Duration
	dur += time.Duration(d.TH) * time.Hour
	dur += time.Duration(d.TM) * time.Minute
	dur += time.Duration(d.TS) * time.Second

	return dur, nil
}
