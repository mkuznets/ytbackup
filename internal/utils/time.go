package utils

import (
	"context"
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
