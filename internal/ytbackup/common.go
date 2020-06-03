package ytbackup

import (
	"database/sql"
	"fmt"

	"mkuznets.com/go/ytbackup/internal/config"
	"mkuznets.com/go/ytbackup/internal/database"
)

// Options is a group of common options for all subcommands.
type Options struct {
	ConfigPath string `short:"c" long:"config" description:"custom config path"`
}

// Command is a common part of all subcommands.
type Command struct {
	DB     *sql.DB
	Config *Config
}

func (cmd *Command) Init(opts interface{}) error {
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

	db, err := database.New()
	if err != nil {
		return fmt.Errorf("could not initialise database: %v", err)
	}
	cmd.DB = db

	return nil
}
