package config

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

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

	home, err := homedir.Dir()
	if err != nil {
		return nil, err
	}

	// Alternative config from one of default paths
	altPath := ""
	if configHome, ok := os.LookupEnv("XDG_CONFIG_HOME"); ok {
		altPath = filepath.Join(configHome, r.filename)
	} else {
		altPath = filepath.Join(home, ".config", r.filename)
	}

	content, err := ioutil.ReadFile(altPath)
	if err == nil {
		log.Printf("using config file: %v", altPath)
		options = append(options, config.Source(bytes.NewBuffer(content)))
	}

	// Explicit config passed via CLI arguments
	if r.explicitPath != "" {
		absPath, err := homedir.Expand(r.explicitPath)
		if err != nil {
			return nil, err
		}
		log.Printf("using config file: %v", absPath)
		options = append(options, config.File(absPath))
	}

	return options, nil
}
