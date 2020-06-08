package utils

import (
	"fmt"
	"io/ioutil"
	"os"
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
