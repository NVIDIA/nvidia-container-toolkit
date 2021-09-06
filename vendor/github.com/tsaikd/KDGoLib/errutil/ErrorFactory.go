package errutil

import (
	"fmt"
	"sort"
)

// ErrorFactory is used for create or check ErrorObject
type ErrorFactory interface {
	Error() string
	Name() string

	New(err error, params ...interface{}) ErrorObject
	Match(err error) bool
	In(err error) bool
}

type errorFactory struct {
	errtext string
	name    string
}

var namedFactories = map[string]ErrorFactory{}

// AllNamedFactories return all named factories
func AllNamedFactories() map[string]ErrorFactory {
	return namedFactories
}

// AllSortedNamedFactories return all sorted named factories
// NOTE: this is slow for sorting in runtime
func AllSortedNamedFactories() []ErrorFactory {
	sorter := newSorter(namedFactories)
	sort.Sort(sorter)
	return sorter.data
}

// NewFactory return new ErrorFactory instance
func NewFactory(errtext string) ErrorFactory {
	callinfo, _ := RuntimeCaller(1)
	return NewNamedFactory(callinfo.PackageName()+"->"+errtext, errtext)
}

// NewNamedFactory return new ErrorFactory instance with factory name, panic if name duplicated
func NewNamedFactory(name string, errtext string) ErrorFactory {
	if _, ok := namedFactories[name]; ok {
		panic(fmt.Errorf("error factory name duplicated: %q", name))
	}
	factory := &errorFactory{
		errtext: errtext,
		name:    name,
	}
	namedFactories[name] = factory
	return factory
}

// FactoryOf return factory of error, return nil if not factory found
func FactoryOf(err error) ErrorFactory {
	errobj := castErrorObject(nil, 1, err)
	if errobj == nil {
		return nil
	}
	return errobj.Factory()
}

func (t errorFactory) Error() string {
	return t.errtext
}

func (t errorFactory) Name() string {
	return t.name
}

func (t *errorFactory) New(parent error, params ...interface{}) ErrorObject {
	errobj := castErrorObject(t, 1, fmt.Errorf(t.errtext, params...))
	errobj.SetParent(castErrorObject(nil, 1, parent))
	return errobj
}

func (t *errorFactory) Match(err error) bool {
	if t == nil || err == nil {
		return false
	}

	errcomp := castErrorObject(nil, 1, err)
	if errcomp == nil {
		return false
	}

	return errcomp.Factory() == t
}

func (t *errorFactory) In(err error) bool {
	if t == nil || err == nil {
		return false
	}

	exist := false

	if errtmp := WalkErrors(castErrorObject(nil, 1, err), func(errcomp ErrorObject) (stop bool, walkerr error) {
		if errcomp.Factory() == t {
			exist = true
			return true, nil
		}
		return false, nil
	}); errtmp != nil {
		panic(errtmp)
	}

	return exist
}

var _ ErrorFactory = (*errorFactory)(nil)
