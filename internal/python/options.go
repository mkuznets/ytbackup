package python

import "time"

type Option = func(*Python)

func WithPython(path string) Option {
	return func(py *Python) {
		py.executable = path
	}
}

func WithYDLOptions(env map[string]interface{}) Option {
	return func(py *Python) {
		py.ydlOpts = env
	}
}

func WithYDLUpdateInterval(interval time.Duration) Option {
	return func(py *Python) {
		py.ydlUpdateInterval = interval
	}
}

func WithYDLLite(v bool) Option {
	return func(py *Python) {
		py.ydlLite = v
	}
}

func WithYDLVersion(v string) Option {
	return func(py *Python) {
		py.ydlVersion = v
	}
}
