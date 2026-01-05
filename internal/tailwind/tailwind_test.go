package tailwind

import (
	"runtime"
	"testing"
)

func TestBinaryName(t *testing.T) {
	name := binaryName()

	// Should contain "tailwindcss" prefix
	if len(name) < 11 || name[:11] != "tailwindcss" {
		t.Errorf("expected binary name to start with 'tailwindcss', got %s", name)
	}

	// Should have platform suffix
	switch runtime.GOOS {
	case "darwin":
		if runtime.GOARCH == "arm64" {
			if name != "tailwindcss-macos-arm64" {
				t.Errorf("expected tailwindcss-macos-arm64, got %s", name)
			}
		} else {
			if name != "tailwindcss-macos-x64" {
				t.Errorf("expected tailwindcss-macos-x64, got %s", name)
			}
		}
	case "linux":
		if runtime.GOARCH == "arm64" {
			if name != "tailwindcss-linux-arm64" {
				t.Errorf("expected tailwindcss-linux-arm64, got %s", name)
			}
		} else {
			if name != "tailwindcss-linux-x64" {
				t.Errorf("expected tailwindcss-linux-x64, got %s", name)
			}
		}
	case "windows":
		if name != "tailwindcss-windows-x64.exe" {
			t.Errorf("expected tailwindcss-windows-x64.exe, got %s", name)
		}
	}
}

func TestPlatformName(t *testing.T) {
	name := PlatformName()

	// Should not be empty
	if name == "" {
		t.Error("PlatformName() returned empty string")
	}

	// Should contain OS name
	switch runtime.GOOS {
	case "darwin":
		if name[:5] != "macOS" {
			t.Errorf("expected platform name to start with 'macOS', got %s", name)
		}
	case "linux":
		if name[:5] != "Linux" {
			t.Errorf("expected platform name to start with 'Linux', got %s", name)
		}
	case "windows":
		if name[:7] != "Windows" {
			t.Errorf("expected platform name to start with 'Windows', got %s", name)
		}
	}
}

func TestNewBinary(t *testing.T) {
	b := NewBinary()

	if b.Version != Version {
		t.Errorf("expected version %s, got %s", Version, b.Version)
	}

	if b.BinDir == "" {
		t.Error("BinDir should not be empty")
	}
}

func TestNewBinaryWithVersion(t *testing.T) {
	b := NewBinaryWithVersion("v3.4.0")

	if b.Version != "v3.4.0" {
		t.Errorf("expected version v3.4.0, got %s", b.Version)
	}
}

func TestDownloadURL(t *testing.T) {
	b := NewBinary()
	url := b.downloadURL()

	// Should contain GitHub releases URL
	if len(url) < 50 {
		t.Errorf("download URL seems too short: %s", url)
	}

	// Should contain version
	if !containsString(url, b.Version) {
		t.Errorf("download URL should contain version %s: %s", b.Version, url)
	}

	// Should contain binary name
	if !containsString(url, binaryName()) {
		t.Errorf("download URL should contain binary name: %s", url)
	}
}

func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
