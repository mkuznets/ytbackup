package venv

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
)

type VirtualEnv struct {
	dir          string
	python       string
	pip          string
	systemPython string
	runLock      sync.Mutex
	fs           http.FileSystem
}

func New(rootDir string, opts ...Option) (*VirtualEnv, error) {
	venv := &VirtualEnv{
		dir:          rootDir,
		python:       filepath.Join(rootDir, "bin", "python"),
		pip:          filepath.Join(rootDir, "bin", "pip"),
		systemPython: "python3",
	}
	for _, opt := range opts {
		opt(venv)
	}

	if err := venv.init(); err != nil {
		return nil, err
	}
	return venv, nil
}

func (v *VirtualEnv) init() error {
	ctx := context.Background()

	if fi, err := os.Stat(v.dir); err == nil {
		if !fi.IsDir() {
			return fmt.Errorf("not a directory: %v", v.dir)
		}

		rctx, cancel := context.WithTimeout(ctx, 120*time.Second)
		defer cancel()

		if err := v.ensurePip(rctx); err != nil {
			return err
		}
		if err := v.ensureYdl(rctx); err != nil {
			return err
		}
		return nil
	}

	rctx, cancel := context.WithTimeout(ctx, 300*time.Second)
	defer cancel()

	if err := v.ensurePython(rctx); err != nil {
		return err
	}

	_, err := v.run(rctx, nil, v.systemPython, "-m", "venv", v.dir)
	if err != nil {
		return fmt.Errorf("could not create venv: %v", err)
	}

	if err := v.ensurePip(rctx); err != nil {
		return fmt.Errorf("venv error: %v", err)
	}
	if err := v.upgrade(rctx); err != nil {
		return fmt.Errorf("pip install error: %v", err)
	}
	if err := v.ensureYdl(rctx); err != nil {
		return err
	}

	return nil
}

func (v *VirtualEnv) run(ctx context.Context, input io.Reader, name string, args ...string) ([]byte, error) {
	v.runLock.Lock()
	defer v.runLock.Unlock()
	c := exec.CommandContext(ctx, name, args...)
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pgid: 0}

	if input != nil {
		c.Stdin = input
	}
	return c.CombinedOutput()
}

func (v *VirtualEnv) upgrade(ctx context.Context) error {
	if out, err := v.run(ctx, nil, v.pip, "install", "-U", "pip", "setuptools", "wheel"); err != nil {
		log.Err(err).Bytes("output", out).Msg("Could not upgrade setuptools/wheel")
		return err
	}
	if out, err := v.run(ctx, nil, v.pip, "install", "-U", "youtube-dl"); err != nil {
		log.Err(err).Bytes("output", out).Msg("Could not upgrade youtube")
		return err
	}
	return nil
}

func (v *VirtualEnv) ensurePython(ctx context.Context) error {
	out, err := v.run(ctx, nil, v.systemPython, "-V")
	if err != nil {
		return fmt.Errorf("could not check python version: %v", err)
	}
	var major, minor int
	n, err := fmt.Sscanf(string(out), "Python %d.%d", &major, &minor)
	if err != nil || n != 2 {
		return errors.New("could not get major and minor version of python")
	}
	if major < 3 || (major == 3 && minor < 7) {
		return fmt.Errorf("expected Python>=3.7, got %d.%d", major, minor)
	}
	return nil
}

func (v *VirtualEnv) ensurePip(ctx context.Context) error {
	if _, err := v.run(ctx, nil, v.python, "-V"); err != nil {
		return fmt.Errorf("could not run venv python: %v", err)
	}
	if _, err := v.run(ctx, nil, v.pip, "-V"); err != nil {
		return fmt.Errorf("could not run pip: %v", err)
	}
	return nil
}

func (v *VirtualEnv) ensureYdl(ctx context.Context) error {
	if _, err := v.run(ctx, nil, v.python, "-m", "youtube_dl", "--version"); err != nil {
		return fmt.Errorf("could not run youtube-dl: %v", err)
	}
	return nil
}
