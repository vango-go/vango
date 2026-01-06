package router

import (
	"testing"
)

func TestCanonicalizePath(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantPath    string
		wantQuery   string
		wantChanged bool
		wantErr     error
	}{
		// Basic paths
		{
			name:        "root",
			input:       "/",
			wantPath:    "/",
			wantChanged: false,
		},
		{
			name:        "simple path",
			input:       "/about",
			wantPath:    "/about",
			wantChanged: false,
		},
		{
			name:        "nested path",
			input:       "/projects/123",
			wantPath:    "/projects/123",
			wantChanged: false,
		},

		// Trailing slash removal
		{
			name:        "trailing slash",
			input:       "/about/",
			wantPath:    "/about",
			wantChanged: true,
		},
		{
			name:        "nested trailing slash",
			input:       "/projects/123/",
			wantPath:    "/projects/123",
			wantChanged: true,
		},

		// Multiple slashes
		{
			name:        "double slash",
			input:       "/blog//post",
			wantPath:    "/blog/post",
			wantChanged: true,
		},
		{
			name:        "triple slash",
			input:       "/blog///post",
			wantPath:    "/blog/post",
			wantChanged: true,
		},

		// Dot segments
		{
			name:        "single dot",
			input:       "/blog/./post",
			wantPath:    "/blog/post",
			wantChanged: true,
		},
		{
			name:        "double dot up",
			input:       "/blog/posts/../other",
			wantPath:    "/blog/other",
			wantChanged: true,
		},
		{
			name:        "double dot to root",
			input:       "/blog/../",
			wantPath:    "/",
			wantChanged: true,
		},

		// Query string preservation
		{
			name:        "with query string",
			input:       "/projects/123?tab=details",
			wantPath:    "/projects/123",
			wantQuery:   "tab=details",
			wantChanged: false,
		},
		{
			name:        "normalized path with query",
			input:       "/projects/123/?tab=details",
			wantPath:    "/projects/123",
			wantQuery:   "tab=details",
			wantChanged: true,
		},

		// Empty input
		{
			name:        "empty string",
			input:       "",
			wantPath:    "/",
			wantChanged: true,
		},

		// No leading slash
		{
			name:        "no leading slash",
			input:       "about",
			wantPath:    "/about",
			wantChanged: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := CanonicalizePath(tc.input)
			if tc.wantErr != nil {
				if err != tc.wantErr {
					t.Errorf("CanonicalizePath(%q) error = %v, want %v", tc.input, err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Errorf("CanonicalizePath(%q) unexpected error = %v", tc.input, err)
				return
			}
			if result.Path != tc.wantPath {
				t.Errorf("CanonicalizePath(%q).Path = %q, want %q", tc.input, result.Path, tc.wantPath)
			}
			if result.Query != tc.wantQuery {
				t.Errorf("CanonicalizePath(%q).Query = %q, want %q", tc.input, result.Query, tc.wantQuery)
			}
			if result.Changed != tc.wantChanged {
				t.Errorf("CanonicalizePath(%q).Changed = %v, want %v", tc.input, result.Changed, tc.wantChanged)
			}
		})
	}
}

func TestCanonicalizePathErrors(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{
			name:    "backslash",
			input:   "/path\\with\\backslash",
			wantErr: ErrBackslashInPath,
		},
		{
			name:    "null byte literal",
			input:   "/path/\x00/null",
			wantErr: ErrNullByteInPath,
		},
		{
			name:    "null byte encoded",
			input:   "/path/%00/null",
			wantErr: ErrNullByteInPath,
		},
		{
			name:    "invalid percent escape incomplete",
			input:   "/path/%2",
			wantErr: ErrInvalidPercentEscape,
		},
		{
			name:    "invalid percent escape bad chars",
			input:   "/path/%GG",
			wantErr: ErrInvalidPercentEscape,
		},
		{
			name:    "escape root",
			input:   "/../secret",
			wantErr: ErrPathEscapesRoot,
		},
		{
			name:    "deep escape root",
			input:   "/a/../../secret",
			wantErr: ErrPathEscapesRoot,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := CanonicalizePath(tc.input)
			if err != tc.wantErr {
				t.Errorf("CanonicalizePath(%q) error = %v, want %v", tc.input, err, tc.wantErr)
			}
		})
	}
}

func TestCanonicalizeAndValidateNavPath(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		// Valid relative paths
		{
			name:  "simple path",
			input: "/about",
			want:  "/about",
		},
		{
			name:  "path with query",
			input: "/projects/123?tab=details",
			want:  "/projects/123?tab=details",
		},
		{
			name:  "root",
			input: "/",
			want:  "/",
		},
		{
			name:  "needs canonicalization",
			input: "/projects/123/",
			want:  "/projects/123",
		},

		// Invalid absolute URLs
		{
			name:    "http URL",
			input:   "http://evil.com/path",
			wantErr: true,
		},
		{
			name:    "https URL",
			input:   "https://evil.com/path",
			wantErr: true,
		},
		{
			name:    "protocol-relative URL",
			input:   "//evil.com/path",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := CanonicalizeAndValidateNavPath(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Errorf("CanonicalizeAndValidateNavPath(%q) expected error, got nil", tc.input)
				}
				return
			}
			if err != nil {
				t.Errorf("CanonicalizeAndValidateNavPath(%q) unexpected error = %v", tc.input, err)
				return
			}
			if got != tc.want {
				t.Errorf("CanonicalizeAndValidateNavPath(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestDecodeSegment(t *testing.T) {
	tests := []struct {
		name       string
		segment    string
		isCatchAll bool
		want       string
		wantErr    error
	}{
		{
			name:    "plain segment",
			segment: "hello",
			want:    "hello",
		},
		{
			name:    "encoded space",
			segment: "hello%20world",
			want:    "hello world",
		},
		{
			name:       "encoded slash in non-catch-all",
			segment:    "hello%2Fworld",
			isCatchAll: false,
			wantErr:    ErrEncodedSlashInSegment,
		},
		{
			name:       "encoded slash in catch-all",
			segment:    "hello%2Fworld",
			isCatchAll: true,
			want:       "hello/world",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := DecodeSegment(tc.segment, tc.isCatchAll)
			if tc.wantErr != nil {
				if err != tc.wantErr {
					t.Errorf("DecodeSegment(%q, %v) error = %v, want %v", tc.segment, tc.isCatchAll, err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Errorf("DecodeSegment(%q, %v) unexpected error = %v", tc.segment, tc.isCatchAll, err)
				return
			}
			if got != tc.want {
				t.Errorf("DecodeSegment(%q, %v) = %q, want %q", tc.segment, tc.isCatchAll, got, tc.want)
			}
		})
	}
}

func TestDecodePathSegments(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		want    []string
		wantErr bool
	}{
		{
			name: "simple path",
			path: "/a/b/c",
			want: []string{"a", "b", "c"},
		},
		{
			name: "root",
			path: "/",
			want: nil,
		},
		{
			name: "with encoded chars",
			path: "/hello%20world/test",
			want: []string{"hello world", "test"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := DecodePathSegments(tc.path)
			if tc.wantErr {
				if err == nil {
					t.Errorf("DecodePathSegments(%q) expected error", tc.path)
				}
				return
			}
			if err != nil {
				t.Errorf("DecodePathSegments(%q) unexpected error = %v", tc.path, err)
				return
			}
			if len(got) != len(tc.want) {
				t.Errorf("DecodePathSegments(%q) = %v, want %v", tc.path, got, tc.want)
				return
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("DecodePathSegments(%q)[%d] = %q, want %q", tc.path, i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestSplitPathAndQuery(t *testing.T) {
	tests := []struct {
		input     string
		wantPath  string
		wantQuery string
	}{
		{
			input:     "/path?query=value",
			wantPath:  "/path",
			wantQuery: "query=value",
		},
		{
			input:     "/path",
			wantPath:  "/path",
			wantQuery: "",
		},
		{
			input:     "/path?",
			wantPath:  "/path",
			wantQuery: "",
		},
		{
			input:     "/path?a=1&b=2",
			wantPath:  "/path",
			wantQuery: "a=1&b=2",
		},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			gotPath, gotQuery := SplitPathAndQuery(tc.input)
			if gotPath != tc.wantPath {
				t.Errorf("SplitPathAndQuery(%q) path = %q, want %q", tc.input, gotPath, tc.wantPath)
			}
			if gotQuery != tc.wantQuery {
				t.Errorf("SplitPathAndQuery(%q) query = %q, want %q", tc.input, gotQuery, tc.wantQuery)
			}
		})
	}
}
