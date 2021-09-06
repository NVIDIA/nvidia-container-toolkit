package errutil

import "encoding/json"

// WalkFunc is a callback for WalkErrors
type WalkFunc func(errcomp ErrorObject) (stop bool, err error)

// WalkErrors walk from base error through all parents
// return ErrorWalkLoop if detected loop
func WalkErrors(base ErrorObject, walkFunc WalkFunc) (err error) {
	if base == nil {
		return
	}

	loopCheckMap := map[ErrorObject]bool{}
	for base != nil {
		if _, exist := loopCheckMap[base]; exist {
			return ErrorWalkLoop.New(nil)
		}
		loopCheckMap[base] = true

		stop, walkerr := walkFunc(base)
		if walkerr != nil {
			return walkerr
		}
		if stop {
			return
		}

		base = base.Parent()
	}

	return
}

// Length count number of ErrorObject and all parents, return -1 if error
func Length(base ErrorObject) int {
	length := 0
	if err := WalkErrors(base, func(errcomp ErrorObject) (stop bool, walkerr error) {
		length++
		return false, nil
	}); err != nil {
		return -1
	}
	return length
}

// AddParent add parent to errobj
func AddParent(errobj ErrorObject, parent ErrorObject) error {
	if errobj == nil || parent == nil {
		return nil
	}

	// set parent if not exist
	if errobj.Parent() == nil {
		errobj.SetParent(parent)
		return nil
	}

	// find oldest parent to set
	base := errobj
	if err := WalkErrors(base.Parent(), func(errcomp ErrorObject) (stop bool, walkerr error) {
		// already in parent tree
		if errcomp == parent {
			base = nil
			return true, nil
		}
		base = errcomp
		return false, nil
	}); err != nil {
		return err
	}

	if base != nil {
		base.SetParent(parent)
	}

	return nil
}

// MarshalJSON marshal error to json
func MarshalJSON(errobj error) ([]byte, error) {
	errjson, err := newJSON(1, errobj)
	if errjson == nil || err != nil {
		return []byte(""), err
	}
	return json.Marshal(errjson)
}
