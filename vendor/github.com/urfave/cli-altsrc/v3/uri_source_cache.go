package altsrc

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
)

func readURI(uriString string) ([]byte, error) {
	u, err := url.Parse(uriString)
	if err != nil {
		return nil, err
	}

	if u.Host != "" { // i have a host, now do i support the scheme?
		switch u.Scheme {
		case "http", "https":
			res, err := http.Get(uriString)
			if err != nil {
				return nil, err
			}
			return io.ReadAll(res.Body)
		default:
			return nil, fmt.Errorf("%[1]w: scheme of %[2]q is unsupported", Err, uriString)
		}
	} else if u.Path != "" ||
		(runtime.GOOS == "windows" && strings.Contains(u.String(), "\\")) {
		if _, notFoundFileErr := os.Stat(uriString); notFoundFileErr != nil {
			return nil, fmt.Errorf("%[1]w: cannot read from %[2]q because it does not exist", Err, uriString)
		}
		return os.ReadFile(uriString)
	}

	return nil, fmt.Errorf("%[1]w: unable to determine how to load from %[2]q", Err, uriString)
}

type URISourceCache[T any] struct {
	uri          string
	m            *T
	unmarshaller func([]byte, any) error
}

func NewURISourceCache[T any](uri string, f func([]byte, any) error) *URISourceCache[T] {
	return &URISourceCache[T]{
		uri:          uri,
		unmarshaller: f,
	}
}

func (fsc *URISourceCache[T]) Get() T {
	if fsc.m == nil {
		res := new(T)
		if b, err := readURI(fsc.uri); err != nil {
			tracef("failed to read uri %[1]q: %[2]v", fsc.uri, err)
		} else if err := fsc.unmarshaller(b, res); err != nil {
			tracef("failed to unmarshal from file %[1]q: %[2]v", fsc.uri, err)
		} else {
			fsc.m = res
		}
	}

	if fsc.m == nil {
		tracef("returning empty")

		return *(new(T))
	}

	return *fsc.m
}

type MapAnyAnyURISourceCache = URISourceCache[map[any]any]

func NewMapAnyAnyURISourceCache(file string, f func([]byte, any) error) *MapAnyAnyURISourceCache {
	return NewURISourceCache[map[any]any](file, f)
}
