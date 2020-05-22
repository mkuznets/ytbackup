package ytbackup

import (
	"errors"
	"os"

	"gopkg.in/yaml.v2"
)

const (
	configBasename = "ytbackup.yaml"
	configDefaults = `
browser:
  executable: chromium
  debug_port: 9222
`
)

type Config struct {
	Browser Browser `yaml:"browser"`
}

func (cfg *Config) ValidateForHistory() error {
	bcfg := cfg.Browser
	if bcfg.Executable == "" {
		return errors.New("`browser.executable` config value is required")
	}
	if bcfg.DebugPort == 0 {
		return errors.New("`browser.debug_port` config value is required")
	}
	if bcfg.DataDir == "" {
		return errors.New("`browser.data_dir` config value is required")
	}
	return nil
}

type Browser struct {
	Executable string            `yaml:"executable"`
	DebugPort  int               `yaml:"debug_port"`
	DataDir    string            `yaml:"data_dir"`
	ExtraArgs  map[string]string `yaml:"extra_args"`
}

type ConfigCommand struct {
	Command
}

func (cmd *ConfigCommand) Execute(args []string) error {
	out, err := yaml.Marshal(cmd.Config)
	if err != nil {
		return err
	}
	if _, err := os.Stdout.Write(out); err != nil {
		return err
	}
	return nil
}
