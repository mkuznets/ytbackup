package ytbackup

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"mkuznets.com/go/ytbackup/internal/config"
	"mkuznets.com/go/ytbackup/internal/index"
	"mkuznets.com/go/ytbackup/internal/storages"
)

// Options is a group of common options for all subcommands.
type Options struct {
	ConfigPath string `short:"c" long:"config" description:"custom config path" env:"YTBACKUP_CONFIG"`
	Debug      bool   `long:"debug" description:"enable debug logging" env:"YTBACKUP_DEBUG"`
}

// Command is a common part of all subcommands.
type Command struct {
	DB          *sql.DB
	Index       *index.Index
	Storages    *storages.Storages
	Config      *Config
	Wg          *sync.WaitGroup
	Ctx         context.Context
	CriticalCtx context.Context
}

func (cmd *Command) Init(opts interface{}) error {
	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: "2006-01-02 15:04:05",
	})

	// -------------

	ctx, cancel := context.WithCancel(context.Background())
	cmd.Ctx = ctx

	critCtx, critCancel := context.WithCancel(context.Background())
	cmd.CriticalCtx = critCtx

	cmd.Wg = &sync.WaitGroup{}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	//

	go func() {
		cnt := 0
		for s := range signalChan {
			slog := log.Warn().Stringer("signal", s)
			switch cnt {
			case 0:
				slog.Msgf("Graceful termination")
				cancel()
			case 1:
				slog.Msgf("Send one more for hard termination")
				critCancel()
			case 2:
				slog.Msgf("Hard termination")
				os.Exit(1)
			}
			cnt++
		}
	}()

	// -------------

	options, ok := opts.(*Options)
	if !ok {
		panic("type mismatch")
	}

	// -------------

	lvl := zerolog.InfoLevel
	if options.Debug {
		lvl = zerolog.DebugLevel
	}
	zerolog.SetGlobalLevel(lvl)

	// -------------

	var cfg Config

	reader := config.New(
		"ytbackup.yaml",
		config.WithExplicitPath(options.ConfigPath),
		config.WithDefaults(ConfigDefaults),
	)
	if err := reader.Read(&cfg); err != nil {
		return fmt.Errorf("config error: %v", err)
	}

	cmd.Config = &cfg

	// -------------

	sts := storages.New()
	for _, s := range cmd.Config.Storages {
		sts.Add(s.Path)
	}
	cmd.Storages = sts

	// -------------

	idx := index.New(filepath.Join(cmd.Config.Dirs.Metadata(), "index.db"))
	if err := idx.Init(); err != nil {
		return err
	}
	cmd.Index = idx

	return nil
}

func (cmd *Command) Close() {
	cmd.Wg.Wait()
	if err := cmd.Index.Close(); err != nil {
		log.Err(err).Msg("Could not close index")
	}
}
