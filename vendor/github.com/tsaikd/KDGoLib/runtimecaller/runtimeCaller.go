package runtimecaller

import (
	"path"
	"runtime"
	"strings"
)

// GetByFilters return CallInfo until all filters are valid
func GetByFilters(skip int, filters ...Filter) (callinfo CallInfo, ok bool) {
	filters = append(FilterCommons, filters...)
	for {
		skip++

		if callinfo, ok = retrieveCallInfo(skip); !ok {
			return
		}

		valid, stop := filterAll(callinfo, filters...)
		if valid {
			return callinfo, true
		}
		if stop {
			return callinfo, false
		}
	}
}

// ListByFilters return all CallInfo stack for all filters are valid
func ListByFilters(skip int, filters ...Filter) (callinfos []CallInfo) {
	filters = append(FilterCommons, filters...)
	for {
		var callinfo CallInfo
		var ok bool
		skip++

		if callinfo, ok = retrieveCallInfo(skip); !ok {
			return
		}

		valid, stop := filterAll(callinfo, filters...)
		if valid {
			callinfos = append(callinfos, callinfo)
		}
		if stop {
			return
		}
	}
}

// http://stackoverflow.com/questions/25262754/how-to-get-name-of-current-package-in-go
func retrieveCallInfo(skip int) (result CallInfo, ok bool) {
	callinfo := CallInfoImpl{}

	if callinfo.pc, callinfo.filePath, callinfo.line, ok = runtime.Caller(skip + 1); !ok {
		return
	}

	callinfo.fileDir, callinfo.fileName = path.Split(callinfo.filePath)
	callinfo.pcFunc = runtime.FuncForPC(callinfo.pc)

	parts := strings.Split(callinfo.pcFunc.Name(), ".")
	pl := len(parts)
	if pl < 1 {
		return result, false
	}
	callinfo.funcName = parts[pl-1]

	if pl >= 2 && parts[pl-2] != "" && parts[pl-2][0] == '(' {
		callinfo.funcName = parts[pl-2] + "." + callinfo.funcName
		callinfo.packageName = strings.Join(parts[0:pl-2], ".")
	} else {
		callinfo.packageName = strings.Join(parts[0:pl-1], ".")
	}

	return callinfo, true
}

func filterAll(callinfo CallInfo, filters ...Filter) (allvalid bool, onestop bool) {
	allvalid = true
	for _, filter := range filters {
		valid, stop := filter(callinfo)
		allvalid = allvalid && valid
		if stop {
			return allvalid, true
		}
		if !allvalid {
			return
		}
	}
	return
}
