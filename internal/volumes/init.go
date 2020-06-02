package volumes

import (
	"bytes"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/gofrs/uuid"
	"gopkg.in/yaml.v2"
)

func initVolume(path string, key *string) (err error) {
	volFile := filepath.Join(path, "volume")

	f, err := os.OpenFile(volFile, os.O_CREATE|os.O_RDWR, 0664)
	if err != nil {
		return err
	}
	defer func() {
		if e := f.Close(); e != nil {
			err = e
		}
	}()

	content, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}

	content = bytes.TrimSpace(content)
	var vol struct{ Key string }

	if len(content) == 0 {
		if err := f.Truncate(0); err != nil {
			return err
		}

		id, err := uuid.NewV4()
		if err != nil {
			return err
		}

		vol.Key = id.String()

		if _, err := f.WriteString(volFileHeader); err != nil {
			return err
		}
		if err := yaml.NewEncoder(f).Encode(vol); err != nil {
			return err
		}
		*key = id.String()
		return nil
	}

	if err := yaml.Unmarshal(content, &vol); err != nil {
		return err
	}
	if vol.Key == "" {
		return errors.New("could not read volume key")
	}
	*key = vol.Key

	return nil
}
