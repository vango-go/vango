package server

import (
	"fmt"
	"net/url"
)

// AppliedNavigateOptions is the concrete, inspected result of applying NavigateOption closures.
// This is useful in contexts outside server.ctx (e.g. SSR adapters) where NavigateOption is opaque.
type AppliedNavigateOptions struct {
	Replace bool
	Params  map[string]any
	Scroll  bool
}

// ApplyNavigateOptions applies NavigateOption closures to the default option set.
func ApplyNavigateOptions(opts ...NavigateOption) AppliedNavigateOptions {
	options := navigateOptions{
		Scroll: true,
	}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(&options)
	}
	return AppliedNavigateOptions{
		Replace: options.Replace,
		Params:  options.Params,
		Scroll:  options.Scroll,
	}
}

// BuildNavigateURL builds the final navigation URL by applying NavigateOption params
// and validating the resulting path against the navigation security rules.
//
// Returns an empty string if the path is invalid (e.g. absolute URL).
func BuildNavigateURL(path string, opts ...NavigateOption) (fullPath string, applied AppliedNavigateOptions) {
	applied = ApplyNavigateOptions(opts...)

	fullPath = path
	if len(applied.Params) > 0 {
		q := url.Values{}
		for k, v := range applied.Params {
			q.Set(k, fmt.Sprintf("%v", v))
		}
		if len(q) > 0 {
			fullPath = path + "?" + q.Encode()
		}
	}

	if !isRelativePath(fullPath) {
		return "", applied
	}
	return fullPath, applied
}

