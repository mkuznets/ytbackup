package appdirs

import (
	"os"
	"path/filepath"

	"github.com/mitchellh/go-homedir"
)

var (
	Home   string
	Data   string
	Config string
	Cache  string
	State  string
)

func Reload() {
	home, err := homedir.Dir()
	if err != nil {
		panic("could not detect home directory")
	}
	Home = home
	initBaseDirs(home)
}

// nolint
func init() {
	Reload()
}

func SearchConfig(parts ...string) (string, bool) {
	var paths []string

	// XDG
	ps := append([]string{Config}, parts...)
	paths = append(paths, filepath.Join(ps...))

	// Default: $HOME/.config
	ps = append([]string{Home, ".config"}, parts...)
	paths = append(paths, filepath.Join(ps...))

	for _, path := range uniquePaths(paths) {
		if exists(path) {
			return path, true
		}
	}
	return "", false
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

func uniquePaths(paths []string) []string {
	var uniq []string
	registry := map[string]struct{}{}

	for _, p := range paths {
		dir, err := homedir.Expand(p)
		if err != nil || dir == "" || !filepath.IsAbs(dir) {
			continue
		}
		if _, ok := registry[p]; !ok {
			registry[p] = struct{}{}
			uniq = append(uniq, p)
		}
	}

	return uniq
}
