package config

import (
	"strings"

	"github.com/rs/zerolog/log"
	"mkuznets.com/go/ytbackup/internal/appdirs"

	"github.com/mitchellh/go-homedir"
	"go.uber.org/config"
)

type Config interface {
	Validate() error
}

type Reader struct {
	filename, explicitPath, defaultConfig string
}

func New(filename string, opts ...Option) *Reader {
	r := new(Reader)
	r.filename = filename

	for _, opt := range opts {
		opt(r)
	}

	return r
}

func (r *Reader) Read(cfg Config) error {
	copts, err := r.yamlOptions()
	if err != nil {
		return err
	}

	provider, err := config.NewYAML(copts...)
	if err != nil {
		return err
	}

	if err := provider.Get("").Populate(cfg); err != nil {
		return err
	}

	return cfg.Validate()
}

func (r *Reader) yamlOptions() ([]config.YAMLOption, error) {
	options := make([]config.YAMLOption, 0, 3)

	// Default config
	if r.defaultConfig != "" {
		options = append(options, config.Source(strings.NewReader(r.defaultConfig)))
	}

	// Alternative config from one of default paths
	if path, ok := appdirs.SearchConfig(r.filename); ok {
		log.Debug().Str("path", path).Msg("Config file")
		options = append(options, config.File(path))
	}

	// Explicit config passed via CLI arguments
	if r.explicitPath != "" {
		path, err := homedir.Expand(r.explicitPath)
		if err != nil {
			return nil, err
		}
		log.Debug().Str("path", path).Msg("Config file")
		options = append(options, config.File(path))
	}

	return options, nil
}
