package venv

import "net/http"

type Option = func(*VirtualEnv)

func WithPython(path string) Option {
	return func(venv *VirtualEnv) {
		venv.systemPython = path
	}
}

func WithFS(fs http.FileSystem) Option {
	return func(venv *VirtualEnv) {
		venv.fs = fs
	}
}
