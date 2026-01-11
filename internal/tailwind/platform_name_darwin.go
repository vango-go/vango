//go:build darwin

package tailwind

func PlatformName() string {
	return "macOS " + archName()
}

