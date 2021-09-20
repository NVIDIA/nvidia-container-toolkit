errutil
=======

An error handling helper, providing more APIs than built-in package (errors, fmt), and compatible with go error interface

## Why use errutil instead of built-in errors and fmt

[![Cover](https://storage.googleapis.com/gcs.milkr.io/topic/255/cover/ea703e50df3a438da92e2f402358c96d37c5c39a)](https://milkr.io/tsaikd/Go-lang-error-handling)

* https://milkr.io/tsaikd/Go-lang-error-handling

## Usage

* Import package from master branch

```
import "github.com/tsaikd/KDGoLib/errutil"
```

* Declare error factory

```
var ErrorOccurWithReason = errutil.NewFactory("An error occur, reason: %q")
```

* Return error with factory

```
func doSomething() (err error) {
	// do something

	// return error with factory,
	// first argument is parent error,
	// the others are used for factory
	return ErrorOccurWithReason.New(nil, "some reason here")
}
```

* Handle errors

```
func errorHandlingForOneCase() {
	if err := doSomething(); err != nil {
		if ErrorOccurWithReason.In(err) {
			// handling expected error
			return
		}

		// handling unexpected error
		return
	}
}
```

```
func errorHandlingForMultipleCases() {
	if err := doSomething(); err != nil {
		switch errutil.FactoryOf(err) {
		case ErrorOccurWithReason:
			// handling expected error
			return
		default:
			// handling unexpected error
			return
		}
	}
}
```

## Optional usage

* Import from v1 branch

```
import "gopkg.in/tsaikd/KDGoLib.v1/errutil"
```

* Use like built-in errors package
  * bad case because all errors should be exported for catching by other package

```
func doSomething() (err error) {
	// do something

	// return error with factory
	return errutil.New("An error occur")
}
```
