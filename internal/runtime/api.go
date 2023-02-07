package runtime

type rt struct {
	logger       *Logger
	modeOverride string
}

// Interface is the interface for the runtime library.
type Interface interface {
	Run([]string) error
}

// Option is a function that configures the runtime.
type Option func(*rt)

// New creates a runtime with the specified options.
func New(opts ...Option) Interface {
	r := rt{}
	for _, opt := range opts {
		opt(&r)
	}
	if r.logger == nil {
		r.logger = NewLogger()
	}
	return &r
}

// WithModeOverride allows for overriding the mode specified in the config.
func WithModeOverride(mode string) Option {
	return func(r *rt) {
		r.modeOverride = mode
	}
}
