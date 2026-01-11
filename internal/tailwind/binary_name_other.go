//go:build !darwin && !linux && !windows

package tailwind

func binaryName() string {
	return "tailwindcss-linux-x64"
}

