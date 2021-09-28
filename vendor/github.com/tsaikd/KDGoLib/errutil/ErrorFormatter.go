package errutil

// ErrorFormatter to format error
type ErrorFormatter interface {
	Format(error) (string, error)
}

// TraceFormatter to trace error occur line formatter
type TraceFormatter interface {
	ErrorFormatter
	FormatSkip(errin error, skip int) (string, error)
}
