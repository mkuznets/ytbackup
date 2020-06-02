package volumes

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"syscall"
)

type Volumes struct {
	vols map[string]string
}

const volFileHeader = `# Created by ytbackup to identify the storage volume.
# Do not delete or edit this file!
`

func New(paths []string) (*Volumes, error) {
	volumes := new(Volumes)
	volumes.vols = make(map[string]string)

	for _, path := range paths {
		var key string
		if err := initVolume(path, &key); err != nil {
			return nil, fmt.Errorf("could not initialise volume %s: %v", path, err)
		}
		if p, ok := volumes.vols[key]; ok {
			return nil, fmt.Errorf("duplicated volumes (key=%s): %s and %s", key, p, path)
		}
		volumes.vols[key] = path
	}

	return volumes, nil
}

func (vs *Volumes) Put(src, dst string) (string, error) {
	srcFile, err := os.Open(src)
	if err != nil {
		return "", err
	}
	fi, err := srcFile.Stat()
	if err != nil {
		return "", err
	}

	targetSize := uint64(fi.Size() * 2)

	for key, path := range vs.vols {
		var st syscall.Statfs_t
		if err := syscall.Statfs(path, &st); err != nil {
			return "", err
		}

		freeSpace := st.Bavail * uint64(st.Bsize)
		log.Printf("free: %v, required: %v", freeSpace, targetSize)

		if freeSpace < targetSize {
			continue
		}

		realDst := filepath.Join(path, dst)

		if err := os.MkdirAll(filepath.Dir(realDst), 0755); err != nil {
			return "", fmt.Errorf("could not create destination directory: %v", err)
		}

		if err := copyFile(src, realDst); err != nil {
			return "", err
		}

		return key, nil
	}

	return "", errors.New("could not write file")
}

func copyFile(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		if e := in.Close(); e != nil {
			err = e
		}
	}()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		if e := out.Close(); e != nil {
			err = e
		}
	}()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return
}
