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

func New(path string, basename string, defaults string, cfg interface{}) error {
	copts, err := configOptions(path, basename, defaults)
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

	return nil
}

func configOptions(path string, basename string, defaults string) ([]config.YAMLOption, error) {
	options := make([]config.YAMLOption, 0)

	// Default config
	options = append(options, config.Source(strings.NewReader(defaults)))

	// Alternative config from one of default paths
	home, err := homedir.Dir()
	if err != nil {
		return nil, err
	}

	altPath := ""
	if configHome, ok := os.LookupEnv("XDG_CONFIG_HOME"); ok {
		altPath = filepath.Join(configHome, basename)
	} else {
		altPath = filepath.Join(home, ".config", basename)
	}

	content, err := ioutil.ReadFile(altPath)
	if err == nil {
		log.Printf("using config file: %v", altPath)
		options = append(options, config.Source(bytes.NewBuffer(content)))
	}

	// Primary config passed via CLI arguments
	if path != "" {
		absPath, err := homedir.Expand(path)
		if err != nil {
			return nil, err
		}
		log.Printf("using config file: %v", absPath)
		options = append(options, config.File(absPath))
	}

	return options, nil
}
