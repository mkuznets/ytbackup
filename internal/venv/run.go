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
	Traceback string `json:"traceback"`
}

func (se *ScriptError) Error() string {
	return se.ErrorText
}

func (v *VirtualEnv) Run(ctx context.Context, result interface{}, args ...string) error {
	if len(args) < 1 {
		panic("expected at least one argument")
	}

	f, err := v.fs.Open(args[0])
	if err != nil {
		return fmt.Errorf("%s: %v", args[0], err)
	}

	cargs := append([]string{"-"}, args[1:]...)
	out, err := v.run(ctx, f, v.python, cargs...)

	if err != nil {
		e, ok := err.(*exec.ExitError)
		if ok && e.ExitCode() == knownErrorCode {
			var serr ScriptError
			if e := json.Unmarshal(e.Stderr, &serr); e != nil {
				return fmt.Errorf("could not decode script error: %v", e)
			}
			return &serr
		}
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("timeout exceeded")
		}
		return err
	}

	if err := json.Unmarshal(out, &result); err != nil {
		return fmt.Errorf("could not decode script result: %v", err)
	}

	return nil
}
