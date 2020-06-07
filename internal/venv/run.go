package venv

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
)

const knownErrorCode = 0xE7

type ScriptError struct {
	ErrorText string `json:"error"`
	Reason    string `json:"reason"`
}

func (se *ScriptError) Error() string {
	return se.ErrorText
}

func (v *VirtualEnv) RunScript(ctx context.Context, result interface{}, args ...string) error {
	if len(args) < 1 {
		panic("expected at least one argument")
	}
	if v.fs == nil {
		return errors.New("scripts filesystem is not configured")
	}

	f, err := v.fs.Open(args[0])
	if err != nil {
		return fmt.Errorf("%s: %v", args[0], err)
	}

	cargs := append([]string{"-"}, args[1:]...)
	out, err := v.run(ctx, f, v.python, cargs...)

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
