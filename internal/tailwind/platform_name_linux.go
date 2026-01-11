//go:build linux

package tailwind

func PlatformName() string {
	return "Linux " + archName()
}

