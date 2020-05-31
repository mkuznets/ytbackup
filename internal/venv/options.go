package venv

type Option = func(*VirtualEnv)

func WithPython(path string) Option {
	return func(venv *VirtualEnv) {
		venv.systemPython = path
	}
}
