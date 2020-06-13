package python

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type Python struct {
	executable        string
	root              string
	runLock           sync.Mutex
	initialised       bool
	wg                *sync.WaitGroup
	cancel            context.CancelFunc
	ydlUpdateInterval time.Duration
	ydlLite           bool
	ydlVersion        string
}

func New(root string, opts ...Option) *Python {
	py := &Python{
		root:              root,
		executable:        "python3",
		wg:                &sync.WaitGroup{},
		ydlUpdateInterval: 3 * time.Hour,
		ydlLite:           false,
		ydlVersion:        "latest",
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

	py.runLock.Lock()
	defer py.runLock.Unlock()

	if err := py.ensurePython(ctx); err != nil {
		return err
	}
	if err := py.ensureScriptFS(); err != nil {
		return err
	}
	if _, err := py.ensureYDL(ctx); err != nil {
		return err
	}
	if err := py.ensurePyCache(ctx); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	py.cancel = cancel

	py.wg.Add(1)
	go func() {
		log.Debug().
			Stringer("interval", py.ydlUpdateInterval).
			Msg("Starting youtube-dl updater")

		defer py.wg.Done()
		ticker := time.NewTicker(py.ydlUpdateInterval)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if ctx.Err() != nil {
					return
				}
				py.runLock.Lock()

				upgraded, err := py.ensureYDL(ctx)
				if err != nil {
					log.Warn().Err(err).Msg("youtube-dl upgrade error")
				}

				if upgraded {
					if err := py.ensurePyCache(ctx); err != nil {
						log.Warn().Err(err).Msg("python cache error")
					}
				}

				py.runLock.Unlock()
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
