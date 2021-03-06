package python

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/rs/zerolog/log"
)

const knownErrorCode = 0xE7

func (py *Python) run(ctx context.Context, args ...string) ([]byte, error) {
	log.Debug().Str("executable", py.executable).Strs("args", args).Msg("Running python")

	c := exec.CommandContext(ctx, py.executable, args...) // nolint
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pgid: 0}
	c.Env = os.Environ()

	if py.ydlOpts != nil {
		envBuf := bytes.NewBuffer(nil)
		envBuf.WriteString("YDL_OPTS=")
		if err := json.NewEncoder(envBuf).Encode(py.ydlOpts); err != nil {
			return nil, fmt.Errorf("could not encode youtube-dl options: %v", err)
		}
		c.Env = append(c.Env, envBuf.String())
	}

	c.Dir = py.root
	c.Env = append(c.Env, fmt.Sprintf("PYTHONPATH=%s", py.root))
	return c.CombinedOutput()
}

type ScriptError struct {
	ErrorText string `json:"error"`
	Reason    string `json:"reason"`
}

func (se *ScriptError) Error() string {
	return se.ErrorText
}

func (py *Python) RunScript(ctx context.Context, result interface{}, args ...string) error {
	py.runLock.Lock()
	defer py.runLock.Unlock()

	if len(args) < 1 {
		panic("expected at least one argument")
	}

	out, err := py.run(ctx, args...)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("timeout exceeded")
		}
		if e, ok := err.(*exec.ExitError); ok {
			if e.ExitCode() == knownErrorCode {
				var serr ScriptError
				if e := json.Unmarshal(out, &serr); e != nil {
					return fmt.Errorf("could not decode script error: %v", e)
				}
				return &serr
			}
			return fmt.Errorf("%s\n%s", e.Error(), out)
		}
		return err
	}

	if err := json.Unmarshal(out, &result); err != nil {
		return fmt.Errorf("could not decode script result: %v", err)
	}

	return nil
}
