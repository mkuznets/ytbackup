package python

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type Python struct {
	executable  string
	root        string
	runLock     sync.Mutex
	initialised bool
	wg          *sync.WaitGroup
	cancel      context.CancelFunc
}

const (
	upgradeCheckInterval = 3 * time.Hour
)

func New(root string, opts ...Option) *Python {
	py := &Python{
		root:       root,
		executable: "python3",
		wg:         &sync.WaitGroup{},
	}
	for _, opt := range opts {
		opt(py)
	}
	return py
}

func (py *Python) Init(ctx context.Context) error {
	if py.initialised {
		return nil
	}
	log.Debug().Msg("Initialising Python environment")

	if err := py.ensurePython(ctx); err != nil {
		return err
	}
	if err := py.ensureYDL(ctx); err != nil {
		return err
	}
	if err := py.ensureScriptFS(); err != nil {
		return err
	}
	if err := py.ensurePyCache(ctx); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	py.cancel = cancel

	py.wg.Add(1)
	go func() {
		defer py.wg.Done()
		ticker := time.NewTicker(upgradeCheckInterval)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if ctx.Err() != nil {
					return
				}
				if err := py.ensureYDL(ctx); err != nil {
					log.Warn().Err(err).Msg("youtube-dl upgrade error")
				}
			}
		}
	}()

	py.initialised = true
	return nil
}

func (py *Python) Close() {
	if !py.initialised {
		return
	}
	py.cancel()
	py.wg.Wait()
	log.Debug().Msg("Python environment closed")
}
