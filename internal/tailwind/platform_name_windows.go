//go:build windows

package tailwind

func PlatformName() string {
	return "Windows " + archName()
}

