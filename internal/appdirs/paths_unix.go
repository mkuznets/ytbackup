package appdirs

import (
	"os"
	"path/filepath"

	"github.com/mitchellh/go-homedir"
)

func initBaseDirs(home string) {
	Data = xdgPath("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	Config = xdgPath("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	Cache = xdgPath("XDG_CACHE_HOME", filepath.Join(home, ".cache"))
}

func xdgPath(name, defaultPath string) string {
	dir, err := homedir.Expand(os.Getenv(name))
	if err == nil && dir != "" && filepath.IsAbs(dir) {
		return dir
	}
	return defaultPath
}
