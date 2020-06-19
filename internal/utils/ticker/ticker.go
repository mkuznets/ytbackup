package ticker

import (
	"context"
	"time"
)

type Ticker struct {
	runFirst bool
	interval time.Duration
}

func New(interval time.Duration, opts ...Option) *Ticker {
	ticker := &Ticker{
		runFirst: true,
		interval: interval,
	}
	for _, opt := range opts {
		opt(ticker)
	}
	return ticker
}

func (t *Ticker) Do(ctx context.Context, fun func() error) error {
	if t.runFirst {
		if err := fun(); err != nil {
			return err
		}
	}

	timer := time.NewTimer(t.interval)
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
				timer.Reset(t.interval)
				return nil
			}

			if err := fun(); err != nil {
				return err
			}

			timer.Reset(t.interval)
		}
	}
}

func (t *Ticker) MustDo(ctx context.Context, fun func() error) {
	if err := t.Do(ctx, fun); err != nil {
		panic(err)
	}
}
