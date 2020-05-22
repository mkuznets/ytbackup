package ytbackup

import (
	"mkuznets.com/go/ytbackup/internal/config"
)

// Options is a group of common options for all subcommands.
type Options struct {
	ConfigPath string `short:"c" long:"config" description:"custom config path"`
}

// Command is a common part of all subcommands.
type Command struct {
	Config  *Config
	workdir string
}

func (cmd *Command) Init(opts interface{}) error {
	options, ok := opts.(*Options)
	if !ok {
		panic("type mismatch")
	}

	var cfg Config
	err := config.New(options.ConfigPath, configBasename, configDefaults, &cfg)
	if err != nil {
		return err
	}
	cmd.Config = &cfg

	return nil
}
