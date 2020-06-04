package ytbackup

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/mitchellh/go-homedir"
	"golang.org/x/oauth2"
	"mkuznets.com/go/ytbackup/internal/browser"
	"mkuznets.com/go/ytbackup/internal/volumes"
	"mkuznets.com/go/ytbackup/pkg/obscure"
)

const ConfigDefaults = `
python:
  executable: python
browser:
  executable: chromium
  debug_port: 9222
update_interval: 5m
`

type Config struct {
	Sources struct {
		History   bool
		Playlists map[string]string
	}
	Destinations Volumes
	Youtube      struct {
		OAuth OAuth `yaml:"oauth"`
	}
	UpdateInterval time.Duration `yaml:"update_interval"`
	Python         struct {
		Executable string `yaml:"executable"`
	}
	Browser Browser
}

type Volumes []string

func (vs *Volumes) New() (*volumes.Volumes, error) {
	return volumes.New(*vs)
}

type Browser struct {
	Executable string            `yaml:"executable"`
	DebugPort  int               `yaml:"debug_port"`
	DataDir    string            `yaml:"data_dir"`
	ExtraArgs  map[string]string `yaml:"extra_args"`
}

func (bcfg *Browser) New() (*browser.Browser, error) {
	return browser.New(bcfg.Executable, bcfg.DataDir, bcfg.DebugPort, bcfg.ExtraArgs)
}

type OAuth struct {
	AccessToken  string    `json:"access_token" yaml:"access_token"`
	TokenType    string    `json:"token_type,omitempty" yaml:"token_type,omitempty"`
	RefreshToken string    `json:"refresh_token,omitempty" yaml:"refresh_token,omitempty"`
	Expiry       time.Time `json:"expiry,omitempty" yaml:"expiry,omitempty"`
}

func (cr *OAuth) Token() *oauth2.Token {
	return &oauth2.Token{
		AccessToken:  obscure.MustReveal(cr.AccessToken),
		TokenType:    cr.TokenType,
		RefreshToken: obscure.MustReveal(cr.RefreshToken),
		Expiry:       cr.Expiry,
	}
}

func NewCredentials(token *oauth2.Token) *OAuth {
	return &OAuth{
		AccessToken:  obscure.MustObscure(token.AccessToken),
		TokenType:    token.TokenType,
		RefreshToken: obscure.MustObscure(token.RefreshToken),
		Expiry:       token.Expiry,
	}
}

func (cfg *Config) Validate() error {
	if cfg.Sources.History {
		if err := cfg.validateBrowser(); err != nil {
			return fmt.Errorf("watch history is enabled, but browser is misconfigured:\n%v", err)
		}
	}
	if len(cfg.Sources.Playlists) > 0 {
		if err := cfg.validateYoutube(); err != nil {
			return fmt.Errorf("playlists are enabled, but youtube config is invalid:\n%v", err)
		}
	}
	if err := cfg.validateDestinations(); err != nil {
		return err
	}
	return nil
}

func (cfg *Config) validateYoutube() error {
	oauth := cfg.Youtube.OAuth
	if oauth.AccessToken == "" || oauth.RefreshToken == "" || oauth.TokenType == "" {
		return errors.New("oauth.{access_token, token_type, refresh_token} are required with `method: oauth`")
	}
	return nil
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

func (cfg *Config) validateDestinations() (err error) {
	if len(cfg.Destinations) == 0 {
		return errors.New("at least one destination directory must be configured")
	}

	for i := range cfg.Destinations {
		path, err := homedir.Expand(cfg.Destinations[i])
		if err != nil {
			return fmt.Errorf("could not expand destination path: %v", err)
		}
		cfg.Destinations[i] = path

		fi, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("destination path does not exist: %s", path)
			}
			return fmt.Errorf("could not open destination path %s: %v", path, err)
		}
		if !fi.IsDir() {
			return fmt.Errorf("provided destination path is not a directory: %s", path)
		}

		f, err := ioutil.TempFile(path, ".tmp*")
		if err != nil {
			return fmt.Errorf("destination path is not writable: %s", path)
		}
		_ = f
		if err := os.Remove(f.Name()); err != nil {
			return err
		}
		if err := f.Close(); err != nil {
			return err
		}
	}
	return nil
}
