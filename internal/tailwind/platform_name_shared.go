package tailwind

import "runtime"

func archName() string {
	switch runtime.GOARCH {
	case "arm64":
		return "ARM64"
	case "amd64":
		return "x64"
	default:
		return runtime.GOARCH
	}
}

