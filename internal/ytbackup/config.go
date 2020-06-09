package ytbackup

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"
	"mkuznets.com/go/ytbackup/internal/appdirs"
	"mkuznets.com/go/ytbackup/internal/browser"
	"mkuznets.com/go/ytbackup/internal/utils"
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
	Dirs struct {
		Cache string
		Data  string
		Logs  string
	}
	Storages []Storage
	Youtube  struct {
		OAuth OAuth `yaml:"oauth"`
	}
	UpdateInterval time.Duration `yaml:"update_interval"`
	Python         struct {
		Executable string `yaml:"executable"`
	}
	Browser Browser
}

type Storage struct {
	Path string
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
	if err := cfg.validateDirs(); err != nil {
		return err
	}

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
	if err := cfg.validateStorages(); err != nil {
		return err
	}
	return nil
}

func (cfg *Config) validateDirs() error {
	paths := make(map[string]string)

	// ---

	if cfg.Dirs.Cache == "" {
		cfg.Dirs.Cache = filepath.Join(appdirs.Cache, "ytbackup")
	}
	cfg.Dirs.Cache = utils.MustExpand(cfg.Dirs.Cache)
	paths["dirs.cache"] = cfg.Dirs.Cache

	// ---

	if cfg.Dirs.Data == "" {
		cfg.Dirs.Data = filepath.Join(appdirs.Data, "ytbackup")
	}
	cfg.Dirs.Data = utils.MustExpand(cfg.Dirs.Data)
	paths["dirs.data"] = cfg.Dirs.Data

	// ---

	if cfg.Dirs.Logs == "" {
		cfg.Dirs.Logs = filepath.Join(appdirs.State, "ytbackup")
	}
	cfg.Dirs.Logs = utils.MustExpand(cfg.Dirs.Logs)
	paths["dirs.logs"] = cfg.Dirs.Logs

	// ---

	for key, path := range paths {
		if err := os.MkdirAll(path, os.FileMode(0755)); err != nil {
			return fmt.Errorf("could not create `%s`: %v", key, err)
		}
		if err := utils.IsWritableDir(path); err != nil {
			return fmt.Errorf("`%s`: %v", key, err)
		}

		purpose := strings.Title(strings.Split(key, ".")[1])
		log.Debug().Str("path", path).Msgf("%s dir", purpose)
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
