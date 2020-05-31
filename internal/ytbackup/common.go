package ytbackup

import (
	"errors"
	"fmt"

	"github.com/mitchellh/go-homedir"
	"mkuznets.com/go/ytbackup/internal/config"
)

const configDefaults = `
browser:
  executable: chromium
  debug_port: 9222
`

type Config struct {
	History struct {
		Enable bool
	}
	Browser struct {
		Executable string            `yaml:"executable"`
		DebugPort  int               `yaml:"debug_port"`
		DataDir    string            `yaml:"data_dir"`
		ExtraArgs  map[string]string `yaml:"extra_args"`
	}
}

func (cfg *Config) validateBrowser() error {
	bcfg := &cfg.Browser
	if bcfg.Executable == "" {
		return errors.New("`browser.executable` is required")
	}
	if bcfg.DebugPort == 0 {
		return errors.New("`browser.debug_port` is required")
	}
	if bcfg.DataDir == "" {
		return errors.New("`browser.data_dir` is required")
	}

	absDataDir, err := homedir.Expand(bcfg.DataDir)
	if err != nil {
		return fmt.Errorf("could not expand `browser.data_dir`: %v", err)
	}
	bcfg.DataDir = absDataDir

	return nil
}

func (cfg *Config) Validate() error {
	if cfg.History.Enable {
		if err := cfg.validateBrowser(); err != nil {
			return fmt.Errorf("watch history is enabled, but browser is misconfigured:\n%v", err)
		}
	}
	return nil
}

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

	reader := config.New(
		"ytbackup.yaml",
		config.WithExplicitPath(options.ConfigPath),
		config.WithDefaults(configDefaults),
	)
	if err := reader.Read(&cfg); err != nil {
		return err
	}

	cmd.Config = &cfg

	return nil
}
