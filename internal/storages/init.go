package storages

import (
	"os"
	"path/filepath"

	"github.com/gofrs/uuid"
	"gopkg.in/yaml.v2"
)

const volFileHeader = `
# Created by ytbackup to identify the storage volume.
# Do not delete or edit this file!
`

func initStorageID(path string) (string, error) {
	volFile := filepath.Join(path, "storage")

	f, err := os.OpenFile(volFile, os.O_CREATE|os.O_RDWR, 0664)
	if err != nil {
		return "", err
	}
	defer func() {
		if e := f.Close(); e != nil {
			err = e
		}
	}()

	var vol struct{ ID string }

	if err := yaml.NewDecoder(f).Decode(&vol); err == nil && vol.ID != "" {
		// Return existing storage id
		return vol.ID, nil
	}

	// Write new storage id

	if err := f.Truncate(0); err != nil {
		return "", err
	}

	uid, err := uuid.NewV4()
	if err != nil {
		return "", err
	}
	vol.ID = uid.String()

	if err := yaml.NewEncoder(f).Encode(vol); err != nil {
		return "", err
	}

	if _, err := f.WriteString(volFileHeader); err != nil {
		return "", err
	}

	return vol.ID, nil
}
