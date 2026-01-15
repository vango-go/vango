package router

import "github.com/vango-go/vango/pkg/routepath"

// CanonicalizeResult contains the result of path canonicalization.
type CanonicalizeResult = routepath.CanonicalizeResult

// Path canonicalization errors.
var (
	ErrInvalidPath           = routepath.ErrInvalidPath
	ErrBackslashInPath       = routepath.ErrBackslashInPath
	ErrNullByteInPath        = routepath.ErrNullByteInPath
	ErrInvalidPercentEscape  = routepath.ErrInvalidPercentEscape
	ErrPathEscapesRoot       = routepath.ErrPathEscapesRoot
	ErrEncodedSlashInSegment = routepath.ErrEncodedSlashInSegment
)

// CanonicalizePath normalizes a URL path according to the routing contract.
func CanonicalizePath(input string) (CanonicalizeResult, error) {
	return routepath.CanonicalizePath(input)
}

// DecodeSegment decodes a single path segment.
func DecodeSegment(segment string, isCatchAll bool) (string, error) {
	return routepath.DecodeSegment(segment, isCatchAll)
}

// DecodePathSegments decodes all segments of a path.
func DecodePathSegments(path string) ([]string, error) {
	return routepath.DecodePathSegments(path)
}

// CanonicalizeAndValidateNavPath canonicalizes and validates a navigation path.
func CanonicalizeAndValidateNavPath(path string) (string, error) {
	return routepath.CanonicalizeAndValidateNavPath(path)
}

// SplitPathAndQuery splits a path into path and query components.
// The query is returned without the leading "?".
func SplitPathAndQuery(input string) (path, query string) {
	return routepath.SplitPathAndQuery(input)
}
