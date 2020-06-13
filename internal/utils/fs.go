package utils

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/mitchellh/go-homedir"
)

func IsWritableDir(path string) (err error) {
	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("path does not exist: %s", path)
		}
		return err
	}

	if !fi.IsDir() {
		return fmt.Errorf("path is not a directory: %s", path)
	}

	f, err := ioutil.TempFile(path, ".tmp*")
	if err != nil {
		return fmt.Errorf("path is not writable: %s", path)
	}
	if err := os.Remove(f.Name()); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}

	return nil
}

func MustExpand(path string) string {
	expanded, err := homedir.Expand(path)
	if err != nil {
		panic(err)
	}
	absPath, err := filepath.Abs(expanded)
	if err != nil {
		panic(err)
	}
	return absPath
}

func RemoveDirs(root string) error {
	files, err := ioutil.ReadDir(root)
	if err != nil {
		return err
	}
	for _, fi := range files {
		if fi.IsDir() {
			if err := os.RemoveAll(filepath.Join(root, fi.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}
