package errutil

import (
	"errors"

	"github.com/tsaikd/KDGoLib/runtimecaller"
)

// errors
var (
	ErrorWalkLoop = NewFactory("detect error component parents loop when walking")
)

// New return a new ErrorObject object
func New(text string, errs ...error) ErrorObject {
	if text != "" {
		errs = append([]error{errors.New(text)}, errs...)
	}
	return NewErrorsSkip(1, errs...)
}

// NewErrors return ErrorObject that contains all input errors
func NewErrors(errs ...error) ErrorObject {
	return NewErrorsSkip(1, errs...)
}

// NewErrorsSkip return ErrorObject, skip function call
func NewErrorsSkip(skip int, errs ...error) ErrorObject {
	var errcomp ErrorObject
	var errtmp ErrorObject
	for i, size := 0, len(errs); i < size; i++ {
		errtmp = castErrorObject(nil, skip+1, errs[i])
		if errtmp == nil {
			continue
		}

		if errcomp == nil {
			errcomp = errtmp
			continue
		}

		if err := AddParent(errcomp, errtmp); err != nil {
			panic(err)
		}
	}
	return errcomp
}

// ErrorObject is a rich error interface
type ErrorObject interface {
	Error() string
	Factory() ErrorFactory
	Parent() ErrorObject
	SetParent(parent ErrorObject) ErrorObject
	runtimecaller.CallInfo
}

type errorObject struct {
	errtext string
	factory ErrorFactory
	parent  ErrorObject
	runtimecaller.CallInfo
}

func castErrorObject(factory ErrorFactory, skip int, err error) ErrorObject {
	if err == nil {
		return nil
	}
	switch err.(type) {
	case errorObject:
		res := err.(errorObject)
		return &res
	case *errorObject:
		return err.(*errorObject)
	case ErrorObject:
		return err.(ErrorObject)
	default:
		callinfo, _ := RuntimeCaller(skip + 1)
		return &errorObject{
			errtext:  err.Error(),
			factory:  factory,
			CallInfo: callinfo,
		}
	}
}

func getErrorText(errin error) string {
	errobj, ok := errin.(*errorObject)
	if ok {
		return errobj.errtext
	}
	return errin.Error()
}

func (t errorObject) Error() string {
	errtext, _ := errorObjectFormatter.Format(&t)
	return errtext
}

func (t *errorObject) Factory() ErrorFactory {
	if t == nil {
		return nil
	}
	return t.factory
}

func (t *errorObject) Parent() ErrorObject {
	if t == nil {
		return nil
	}
	return t.parent
}

func (t *errorObject) SetParent(parent ErrorObject) ErrorObject {
	if t == nil {
		return nil
	}
	if t == parent {
		return t
	}
	t.parent = parent
	return t
}

func (t *errorObject) MarshalJSON() ([]byte, error) {
	return MarshalJSON(t)
}

var _ ErrorObject = (*errorObject)(nil)

var errorObjectFormatter = ErrorFormatter(&ConsoleFormatter{
	Seperator: "; ",
})
