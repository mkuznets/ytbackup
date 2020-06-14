package python

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/rakyll/statik/fs"
	"github.com/rs/zerolog/log"
	"mkuznets.com/go/ytbackup/internal/pyfs"
)

func (py *Python) ensurePython(ctx context.Context) error {
	out, err := py.run(ctx, "-V")
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
	cacheDirs := make([]string, 0)

	err := filepath.Walk(py.root, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if fi.IsDir() && fi.Name() == "__pycache__" {
			cacheDirs = append(cacheDirs, path)
		}
		return nil
	})
	if err != nil {
		log.Warn().Err(err).Msg("python cache cleanup error")
		return nil
	}
	for _, path := range cacheDirs {
		if err := os.RemoveAll(path); err != nil {
			log.Warn().Err(err).Msg("python cache cleanup error")
		}
	}

	log.Debug().Msg("Pre-compiling Python scripts")

	_, err = py.run(ctx, "-c", `import compileall; compileall.compile_dir(".")`)
	if err != nil {
		return fmt.Errorf("could not create pycache: %v", err)
	}

	return nil
}

func (py *Python) ensureFFMPEG(ctx context.Context) error {
	name := "ffmpeg"
	args := []string{"-version"}

	log.Debug().Str("executable", name).Strs("args", args).Msg("Running")

	c := exec.CommandContext(ctx, name, args...)

	var b bytes.Buffer
	c.Stdout = &b

	if err := c.Run(); err != nil {
		return fmt.Errorf("ffmpeg is not available")
	}

	versionLine, err := bufio.NewReader(&b).ReadString('\n')
	if err != nil {
		log.Warn().Err(err).Msg("could not read ffmpeg version")
		return nil
	}

	var version string

	n, err := fmt.Sscanf(versionLine, "ffmpeg version %s", &version)
	if err != nil || n != 1 {
		log.Warn().Err(err).Msg("could not read ffmpeg version")
		return nil
	}

	log.Debug().Str("version", version).Msg("ffmpeg")
	return nil
}
