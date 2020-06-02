package ytbackup

import (
	"database/sql"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/mitchellh/go-homedir"
	"mkuznets.com/go/ytbackup/internal/config"
	"mkuznets.com/go/ytbackup/internal/database"
)

const configDefaults = `
python:
  executable: python
browser:
  executable: chromium
  debug_port: 9222
`

type Config struct {
	History struct {
		Enable bool
	}
	Python struct {
		Executable string `yaml:"executable"`
	}
	Browser struct {
		Executable string            `yaml:"executable"`
		DebugPort  int               `yaml:"debug_port"`
		DataDir    string            `yaml:"data_dir"`
		ExtraArgs  map[string]string `yaml:"extra_args"`
	}
	Volumes []string
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

func (cfg *Config) validateVolumes() (err error) {
	if len(cfg.Volumes) == 0 {
		return errors.New("at least one volume must be configured")
	}
	for _, path := range cfg.Volumes {
		fi, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("volume path does not exist: %s", path)
			}
			return fmt.Errorf("could not open volume path %s: %v", path, err)
		}
		if !fi.IsDir() {
			return fmt.Errorf("provided volume path is not a directory: %s", path)
		}

		f, err := ioutil.TempFile(path, ".tmp*")
		if err != nil {
			return fmt.Errorf("volume path is not writable: %s", path)
		}
		_ = f
		if err := os.Remove(f.Name()); err != nil {
			return err
		}
		if err := f.Close(); err != nil {
			return err
		}
	}
	return
}

func (cfg *Config) Validate() error {
	if cfg.History.Enable {
		if err := cfg.validateBrowser(); err != nil {
			return fmt.Errorf("watch history is enabled, but browser is misconfigured:\n%v", err)
		}
	}

	if err := cfg.validateVolumes(); err != nil {
		return err
	}

	return nil
}

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
		config.WithDefaults(configDefaults),
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
