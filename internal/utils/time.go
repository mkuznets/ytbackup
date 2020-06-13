package utils

import (
	"context"
	"fmt"
	"time"
)

func RunEveryInterval(ctx context.Context, interval time.Duration, fun func() error) error {
	if err := fun(); err != nil {
		return err
	}

	timer := time.NewTimer(interval)
	defer func() {
		if !timer.Stop() {
			<-timer.C
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return nil

		case <-timer.C:
			if ctx.Err() != nil {
				timer.Reset(interval)
				return nil
			}

			if err := fun(); err != nil {
				return err
			}

			timer.Reset(interval)
		}
	}
}

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
