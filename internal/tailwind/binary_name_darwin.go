//go:build darwin

package tailwind

import "runtime"

func binaryName() string {
	if runtime.GOARCH == "arm64" {
		return "tailwindcss-macos-arm64"
	}
	return "tailwindcss-macos-x64"
}

