package runtimecaller

import "strings"

// Filter use to filter runtime.Caller result
type Filter func(callinfo CallInfo) (valid bool, stop bool)

// FilterCommons contains all common filters
var FilterCommons = []Filter{
	FilterOnlyGoSource,
	FilterStopRuntimeCallerPackage,
}

// FilterOnlyGoSource filter CallInfo FileName end with ".go"
func FilterOnlyGoSource(callinfo CallInfo) (valid bool, stop bool) {
	filename := strings.ToLower(callinfo.FileName())
	return strings.HasSuffix(filename, ".go"), false
}

// FilterStopRuntimeCallerPackage filter CallInfo to stop after reach KDGoLib/runtimecaller package
func FilterStopRuntimeCallerPackage(callinfo CallInfo) (valid bool, stop bool) {
	if callinfo.PackageName() == "github.com/tsaikd/KDGoLib/runtimecaller" {
		return false, true
	}
	return true, false
}
