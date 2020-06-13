package storages

import (
	"fmt"
	"syscall"

	"github.com/rs/zerolog/log"
	"mkuznets.com/go/ytbackup/internal/utils"
)

const freeRequired = 1 << 30 // 1 GiB

type Ready struct {
	ID, Path string
	Free     uint64
}

type Storages struct {
	paths []string
}

func New() *Storages {
	return &Storages{}
}

func (st *Storages) Add(path string) {
	st.paths = append(st.paths, path)
}

func (st *Storages) List() []*Ready {
	rs := make([]*Ready, 0, len(st.paths))

	for _, path := range st.paths {
		if err := utils.IsWritableDir(path); err != nil {
			log.Debug().Err(err).Msg("storage path is not writable")
			continue
		}

		id, err := initStorageID(path)
		if err != nil {
			log.Debug().Err(err).Msg("could not create or read storage id")
			continue
		}

		r := &Ready{ID: id, Path: path, Free: freeSpace(path)}
		log.Debug().
			Str("path", r.Path).
			Str("id", r.ID).
			Str("free", utils.IBytes(r.Free)).
			Msg("Storage found")

		rs = append(rs, r)
	}

	return rs
}

func (st *Storages) Get() (*Ready, error) {
	for _, r := range st.List() {
		if r.Free > freeRequired {
			log.Debug().
				Str("path", r.Path).
				Str("id", r.ID).
				Str("free", utils.IBytes(r.Free)).
				Msg("Storage selected")
			return r, nil
		}
	}
	return nil, fmt.Errorf("no storage with >= %s free space", utils.IBytes(freeRequired))
}

func freeSpace(path string) uint64 {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return 0
	}
	return st.Bavail * uint64(st.Bsize)
}
