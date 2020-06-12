package python

type Option = func(*Python)

func WithPython(path string) Option {
	return func(py *Python) {
		py.executable = path
	}
}
