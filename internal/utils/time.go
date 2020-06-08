package utils

import (
	"context"
	"time"
)

func RunEveryInterval(ctx context.Context, interval time.Duration, fun func()) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	fun()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if ctx.Err() != nil {
				return
			}
			fun()
		}
	}
}
