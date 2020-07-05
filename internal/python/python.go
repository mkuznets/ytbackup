package python

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"mkuznets.com/go/ytbackup/internal/utils/ticker"
)

type Python struct {
	ydlOpts           map[string]interface{}
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

	if err := py.ensureFFMPEG(ctx); err != nil {
		return err
	}
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
		defer py.wg.Done()

		log.Debug().
			Stringer("interval", py.ydlUpdateInterval).
			Msg("Starting youtube-dl updater")

		ticker.New(py.ydlUpdateInterval, ticker.SkipFirst).MustDo(ctx, func() error {
			py.runLock.Lock()
			defer py.runLock.Unlock()

			upgraded, err := py.ensureYDL(ctx)
			if err != nil {
				log.Warn().Err(err).Msg("youtube-dl upgrade error")
				return nil
			}

			if upgraded {
				if err := py.ensurePyCache(ctx); err != nil {
					log.Warn().Err(err).Msg("python cache error")
				}
			}
			return nil
		})
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
