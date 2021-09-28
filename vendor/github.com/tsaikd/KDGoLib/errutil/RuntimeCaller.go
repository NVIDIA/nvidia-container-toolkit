package errutil

import (
	"bytes"
	"io"
	"strconv"
	"strings"

	"github.com/tsaikd/KDGoLib/runtimecaller"
)

// DefaultRuntimeCallerFilter use for filter error stack info
var DefaultRuntimeCallerFilter = []runtimecaller.Filter{}

func init() {
	DefaultRuntimeCallerFilter = append(
		runtimecaller.FilterCommons,
		RuntimeCallerFilterStopErrutilPackage,
		RuntimeCallerFilterStopSQLUtilPackage,
	)
}

// AddRuntimeCallerFilter add filters to DefaultRuntimeCallerFilter for RuntimeCaller()
func AddRuntimeCallerFilter(filters ...runtimecaller.Filter) {
	DefaultRuntimeCallerFilter = append(DefaultRuntimeCallerFilter, filters...)
}

// RuntimeCallerFilterStopErrutilPackage filter CallInfo to stop after reach KDGoLib/errutil package
func RuntimeCallerFilterStopErrutilPackage(callinfo runtimecaller.CallInfo) (valid bool, stop bool) {
	if callinfo.PackageName() == "github.com/tsaikd/KDGoLib/errutil" {
		return false, true
	}
	return true, false
}

// RuntimeCallerFilterStopSQLUtilPackage filter CallInfo to stop after reach KDGoLib/sqlutil package
func RuntimeCallerFilterStopSQLUtilPackage(callinfo runtimecaller.CallInfo) (valid bool, stop bool) {
	if callinfo.PackageName() == "github.com/tsaikd/KDGoLib/sqlutil" {
		return false, true
	}
	return true, false
}

// RuntimeCaller wrap runtimecaller.GetByFilters() with DefaultRuntimeCallerFilter
func RuntimeCaller(skip int, extraFilters ...runtimecaller.Filter) (callinfo runtimecaller.CallInfo, ok bool) {
	filters := append(DefaultRuntimeCallerFilter, extraFilters...)
	return runtimecaller.GetByFilters(skip+1, filters...)
}

// WriteCallInfo write readable callinfo with options
func WriteCallInfo(
	output io.Writer,
	callinfo runtimecaller.CallInfo,
	longFile bool,
	line bool,
	replacePackages map[string]string,
) (n int, err error) {
	buffer := &bytes.Buffer{}
	if longFile {
		pkgname := replacePackage(replacePackages, callinfo.PackageName())
		if _, err = buffer.WriteString(pkgname + "/" + callinfo.FileName()); err != nil {
			return
		}
	} else {
		if _, err = buffer.WriteString(callinfo.FileName()); err != nil {
			return
		}
	}
	if line {
		if _, err = buffer.WriteString(":" + strconv.Itoa(callinfo.Line())); err != nil {
			return
		}
	}
	return output.Write([]byte(buffer.String()))
}

func replacePackage(replacePackages map[string]string, pkgname string) (replaced string) {
	replaced = pkgname
	if replacePackages == nil {
		return
	}
	for src, tar := range replacePackages {
		replaced = strings.Replace(replaced, src, tar, -1)
	}
	return
}
