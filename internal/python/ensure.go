package python

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/rakyll/statik/fs"
	"github.com/rs/zerolog/log"
	"mkuznets.com/go/ytbackup/internal/pyfs"
)

func (py *Python) ensurePython(ctx context.Context) error {
	out, err := py.run(ctx, nil, "-V")
	if err != nil {
		return fmt.Errorf("could not run python: %v", err)
	}
	fields := bytes.Fields(out)
	if len(fields) < 2 {
		return fmt.Errorf("could not parse Python version: %s", out)
	}
	version := fields[1]

	log.Debug().Bytes("version", version).Msg("Python")

	var major, minor int
	n, err := fmt.Sscanf(string(version), "%d.%d", &major, &minor)
	if err != nil || n != 2 {
		return errors.New("could not get major and minor version of python")
	}
	if major < 3 || (major == 3 && minor < 5) {
		return fmt.Errorf("expected Python>=3.5, got %d.%d", major, minor)
	}

	return nil
}

func (py *Python) ensureScriptFS() error {
	scriptFS, err := fs.NewWithNamespace(pyfs.Python)
	if err != nil {
		return fmt.Errorf("could not open pyfs: %v", err)
	}
	log.Debug().Msg("Extracting Python scripts")

	return fs.Walk(scriptFS, "/", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Err(err).Msg("pyfs error")
			return nil
		}

		rel, err := filepath.Rel("/", path)
		if err != nil {
			return fmt.Errorf("relative path: %v", err)
		}
		target := filepath.Join(py.root, rel)

		if info.IsDir() {
			if err := os.MkdirAll(target, os.FileMode(0755)); err != nil {
				return fmt.Errorf("could not create directory: %v", err)
			}
			return nil
		}

		content, err := fs.ReadFile(scriptFS, path)
		if err != nil {
			return err
		}
		if err := ioutil.WriteFile(target, content, os.FileMode(0755)); err != nil {
			return err
		}

		return nil
	})
}

func (py *Python) ensurePyCache(ctx context.Context) error {
	_ = os.RemoveAll(filepath.Join(py.root, "__pycache__"))

	log.Debug().Msg("Pre-compiling Python scripts")

	_, err := py.run(ctx, nil, "-c", `import compileall; compileall.compile_dir(".")`)
	if err != nil {
		return fmt.Errorf("could not create pycache: %v", err)
	}

	return nil
}
