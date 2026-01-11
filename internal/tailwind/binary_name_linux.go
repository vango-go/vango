//go:build linux

package tailwind

import "runtime"

func binaryName() string {
	if runtime.GOARCH == "arm64" {
		return "tailwindcss-linux-arm64"
	}
	return "tailwindcss-linux-x64"
}

