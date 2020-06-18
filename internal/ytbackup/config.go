package ytbackup

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"
	"mkuznets.com/go/ytbackup/internal/appdirs"
	"mkuznets.com/go/ytbackup/internal/browser"
	"mkuznets.com/go/ytbackup/internal/utils"
	"mkuznets.com/go/ytbackup/pkg/obscure"
)

const ConfigDefaults = `
sources:
  update_interval: 5m
  max_duration: 9h

browser:
  executable: chromium
  debug_port: 9222

python:
  executable: python3
  youtube-dl:
    update_interval: 3h
    version: latest
    lite: false
`

type Config struct {
	Sources struct {
		History struct {
			Enable bool
		}
		Playlists      map[string]string
		UpdateInterval time.Duration `yaml:"update_interval"`
		MaxDuration    time.Duration `yaml:"max_duration"`
	}
	Dirs     Dirs
	Storages []struct {
		Path string
	}
	Youtube struct {
		OAuth OAuth `yaml:"oauth"`
	}
	Python struct {
		Executable string `yaml:"executable"`
		YoutubeDL  struct {
			Lite           bool
			Version        string
			UpdateInterval time.Duration `yaml:"update_interval"`
		} `yaml:"youtube-dl"`
	}
	Browser Browser
}

type Dirs struct {
	Cache string
	Data  string
}

func (dirs *Dirs) Logs() string {
	return filepath.Join(dirs.Data, "logs")
}

func (dirs *Dirs) Metadata() string {
	return filepath.Join(dirs.Data, "metadata")
}

func (dirs *Dirs) Python() string {
	return filepath.Join(dirs.Data, "python")
}

func (dirs *Dirs) validate() error {
	paths := make([]string, 0, 5)

	// ---

	if dirs.Cache == "" {
		dirs.Cache = filepath.Join(appdirs.Cache, "ytbackup")
	}
	dirs.Cache = utils.MustExpand(dirs.Cache)
	paths = append(paths, dirs.Cache)

	log.Debug().Str("path", dirs.Cache).Msgf("Cache dir")

	// ---

	if dirs.Data == "" {
		dirs.Data = filepath.Join(appdirs.Data, "ytbackup")
	}
	dirs.Data = utils.MustExpand(dirs.Data)
	paths = append(paths, dirs.Data, dirs.Python(), dirs.Metadata(), dirs.Logs())

	log.Debug().Str("path", dirs.Data).Msgf("Data dir")

	// ---

	for _, path := range paths {
		if err := os.MkdirAll(path, os.FileMode(0755)); err != nil {
			return fmt.Errorf("could not create directory: %v", err)
		}
		if err := utils.IsWritableDir(path); err != nil {
			return fmt.Errorf("directory error: %v", err)
		}
	}

	return nil
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
	if err := cfg.Dirs.validate(); err != nil {
		return err
	}

	if err := cfg.validateYoutube(); err != nil {
		return fmt.Errorf("youtube config error:\n%v", err)
	}

	if cfg.Sources.History.Enable {
		if err := cfg.validateBrowser(); err != nil {
			return fmt.Errorf("watch history is enabled, but browser is misconfigured:\n%v", err)
		}
	}

	if err := cfg.validateStorages(); err != nil {
		return err
	}
	return nil
}

func (cfg *Config) validateYoutube() error {
	oauth := cfg.Youtube.OAuth
	if oauth.AccessToken == "" || oauth.RefreshToken == "" || oauth.TokenType == "" {
		return errors.New("oauth.{access_token, token_type, refresh_token} are required")
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
	bcfg.DataDir = utils.MustExpand(bcfg.DataDir)

	return nil
}

func (cfg *Config) validateStorages() (err error) {
	if len(cfg.Storages) == 0 {
		return errors.New("at least one storage must be configured")
	}

	for i, st := range cfg.Storages {
		cfg.Storages[i].Path = utils.MustExpand(st.Path)
	}

	return nil
}
