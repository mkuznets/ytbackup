package ytbackup

import (
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"mkuznets.com/go/ytbackup/internal/config"
	"mkuznets.com/go/ytbackup/internal/index"
)

// Options is a group of common options for all subcommands.
type Options struct {
	ConfigPath string `short:"c" long:"config" description:"custom config path" env:"YTBACKUP_CONFIG"`
}

// Command is a common part of all subcommands.
type Command struct {
	DB     *sql.DB
	Index  *index.Index
	Config *Config
}

func (cmd *Command) Init(opts interface{}) error {
	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: "2006-01-02 15:04:05",
	})
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	options, ok := opts.(*Options)
	if !ok {
		panic("type mismatch")
	}

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

	idx := index.New("index.db")
	if err := idx.Init(); err != nil {
		return fmt.Errorf("could not initialise store: %v", err)
	}
	cmd.Index = idx

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		s := <-signalChan
		log.Info().Stringer("signal", s).Msgf("Graceful shutdown")
		cmd.gracefulShutdown()
	}()

	return nil
}

func (cmd *Command) gracefulShutdown() {
	if err := cmd.Index.Close(); err != nil {
		log.Err(err).Msg("Could not close index")
	}
	os.Exit(1)
}
