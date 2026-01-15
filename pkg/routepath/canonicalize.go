package routepath

import (
	"errors"
	"net/url"
	"strings"
)

// CanonicalizeResult contains the result of path canonicalization.
type CanonicalizeResult struct {
	// Path is the canonicalized path (without query string).
	Path string

	// Query is the query string (without leading "?").
	Query string

	// Changed indicates if the path was modified during canonicalization.
	Changed bool
}

// Path canonicalization errors.
var (
	ErrInvalidPath           = errors.New("invalid path")
	ErrBackslashInPath       = errors.New("path contains backslash")
	ErrNullByteInPath        = errors.New("path contains null byte")
	ErrInvalidPercentEscape  = errors.New("invalid percent escape sequence")
	ErrPathEscapesRoot       = errors.New("path escapes root via ..")
	ErrEncodedSlashInSegment = errors.New("encoded slash (%2F) in non-catch-all segment")
)

// CanonicalizePath normalizes a URL path according to the routing contract.
//
// Per Section 1.2 (Path Canonicalization), the following transformations are applied:
//   - Remove trailing slash (except for root "/")
//   - Collapse multiple slashes (/blog//post → /blog/post)
//   - Remove "." segments (/blog/./post → /blog/post)
//   - Resolve ".." segments (/blog/../other → /other)
//
// The following inputs are rejected with an error:
//   - Paths containing backslash (\)
//   - Paths containing NUL byte (%00)
//   - Invalid percent-escapes (e.g., %GG, %2)
//   - ".." that would escape root (e.g., /../secret)
//
// The input may include a query string, which is preserved but not canonicalized.
func CanonicalizePath(input string) (CanonicalizeResult, error) {
	if input == "" {
		return CanonicalizeResult{Path: "/", Changed: true}, nil
	}

	// Split path and query.
	path, query, _ := strings.Cut(input, "?")

	// SECURITY: Reject backslash.
	if strings.Contains(path, "\\") {
		return CanonicalizeResult{}, ErrBackslashInPath
	}

	// SECURITY: Reject NUL byte (both literal and encoded).
	if strings.Contains(path, "\x00") || strings.Contains(strings.ToUpper(path), "%00") {
		return CanonicalizeResult{}, ErrNullByteInPath
	}

	// Validate percent-escapes if present.
	// Per contract Section 1.2.4: Only validate if input contains "%".
	if strings.Contains(path, "%") {
		if err := validatePercentEscapes(path); err != nil {
			return CanonicalizeResult{}, err
		}
	}

	// Track original before any modifications.
	original := path

	// Ensure path starts with "/".
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	// Collapse multiple slashes.
	for strings.Contains(path, "//") {
		path = strings.ReplaceAll(path, "//", "/")
	}

	// Split into segments and normalize.
	segments := strings.Split(path, "/")
	var result []string

	for _, seg := range segments {
		switch seg {
		case "", ".":
			// Skip empty segments and ".".
			continue
		case "..":
			// Pop the last segment, but don't go above root.
			if len(result) > 0 {
				result = result[:len(result)-1]
			} else {
				// SECURITY: ".." escapes root.
				return CanonicalizeResult{}, ErrPathEscapesRoot
			}
		default:
			result = append(result, seg)
		}
	}

	// Rebuild path.
	path = "/" + strings.Join(result, "/")

	// Remove trailing slash (except for root).
	if len(path) > 1 && strings.HasSuffix(path, "/") {
		path = strings.TrimSuffix(path, "/")
	}

	return CanonicalizeResult{
		Path:    path,
		Query:   query,
		Changed: path != original,
	}, nil
}

// validatePercentEscapes checks that all percent-escapes are valid.
// Valid escapes are %XX where X is a hex digit (0-9, a-f, A-F).
func validatePercentEscapes(path string) error {
	i := 0
	for i < len(path) {
		if path[i] == '%' {
			// Need at least 2 more characters.
			if i+2 >= len(path) {
				return ErrInvalidPercentEscape
			}
			// Check both hex digits.
			if !isHexDigit(path[i+1]) || !isHexDigit(path[i+2]) {
				return ErrInvalidPercentEscape
			}
			i += 3
		} else {
			i++
		}
	}
	return nil
}

// isHexDigit returns true if c is a valid hex digit.
func isHexDigit(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

// DecodeSegment decodes a single path segment.
// For non-catch-all params, if decoding produces "/" (i.e., %2F was present),
// this returns an error as it indicates a path smuggling attempt.
func DecodeSegment(segment string, isCatchAll bool) (string, error) {
	decoded, err := url.PathUnescape(segment)
	if err != nil {
		return "", ErrInvalidPercentEscape
	}

	// SECURITY: For non-catch-all params, reject %2F (encoded slash).
	// This prevents path smuggling attacks.
	if !isCatchAll && strings.Contains(decoded, "/") {
		return "", ErrEncodedSlashInSegment
	}

	return decoded, nil
}

// DecodePathSegments decodes all segments of a path.
// This splits the path by "/" and decodes each segment individually.
// For catch-all segments, the full remainder is decoded but "/" is preserved.
func DecodePathSegments(path string) ([]string, error) {
	// Remove leading slash for splitting.
	path = strings.TrimPrefix(path, "/")
	if path == "" {
		return nil, nil
	}

	segments := strings.Split(path, "/")
	result := make([]string, 0, len(segments))

	for _, seg := range segments {
		decoded, err := url.PathUnescape(seg)
		if err != nil {
			return nil, ErrInvalidPercentEscape
		}
		result = append(result, decoded)
	}

	return result, nil
}

// CanonicalizeAndValidateNavPath canonicalizes and validates a navigation path.
// This is used for NAV_* patches and ctx.Navigate() to ensure security.
//
// Per Section 4.2 (Full Navigation), NAV_* payloads MUST be relative paths only:
//   - MUST start with "/"
//   - MUST NOT be a full URL (no "http://", "https://", "//")
//
// Returns the canonicalized path with query string, or an error if invalid.
func CanonicalizeAndValidateNavPath(path string) (string, error) {
	// SECURITY: Reject absolute URLs to prevent open-redirect attacks.
	if strings.HasPrefix(path, "http://") ||
		strings.HasPrefix(path, "https://") ||
		strings.HasPrefix(path, "//") {
		return "", ErrInvalidPath
	}
	if !strings.HasPrefix(path, "/") {
		return "", ErrInvalidPath
	}

	// Canonicalize the path.
	result, err := CanonicalizePath(path)
	if err != nil {
		return "", err
	}

	// Rebuild with query string if present.
	if result.Query != "" {
		return result.Path + "?" + result.Query, nil
	}

	return result.Path, nil
}

// SplitPathAndQuery splits a path into path and query components.
// The query is returned without the leading "?".
func SplitPathAndQuery(input string) (path, query string) {
	path, query, _ = strings.Cut(input, "?")
	return path, query
}
