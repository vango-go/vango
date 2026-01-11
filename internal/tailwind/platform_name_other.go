//go:build !darwin && !linux && !windows

package tailwind

import "runtime"

func PlatformName() string {
	return runtime.GOOS + " " + archName()
}

