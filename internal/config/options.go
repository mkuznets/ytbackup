package config

type Option = func(*Reader)

func WithExplicitPath(path string) Option {
	return func(r *Reader) {
		r.explicitPath = path
	}
}

func WithDefaults(defaults string) Option {
	return func(r *Reader) {
		r.defaultConfig = defaults
	}
}
