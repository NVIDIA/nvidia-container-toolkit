package errutil

// ContainErrorFunc check error contain error by custom equalFunc
func ContainErrorFunc(err error, equalFunc func(error) bool) bool {
	errobj := castErrorObject(nil, 1, err)
	contain := false

	if walkerr := WalkErrors(errobj, func(errcomp ErrorObject) (stop bool, walkerr error) {
		if equalFunc(errcomp) {
			contain = true
			return true, nil
		}
		return false, nil
	}); walkerr != nil {
		return false
	}

	return contain
}
