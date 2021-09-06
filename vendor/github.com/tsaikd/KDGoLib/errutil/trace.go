package errutil

// Trace error stack, output to default ErrorFormatter, panic if output error
func Trace(errin error) {
	TraceSkip(errin, 1)
}

// TraceWrap trace errin and wrap with wraperr only if errin != nil
func TraceWrap(errin error, wraperr error) {
	if errin != nil {
		errs := NewErrorsSkip(1, wraperr, errin)
		TraceSkip(errs, 1)
	}
}

// TraceSkip error stack, output to default ErrorFormatter, skip n function calls, panic if output error
func TraceSkip(errin error, skip int) {
	Logger().TraceSkip(errin, 1)
}
