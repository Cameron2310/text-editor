package main

import (
	"errors"
	"runtime"
)

const armGet = 0x40487413 // TIOCGETA
const armSet = 0x80487414 // TIOCSETA

const amdGet = 0x5401 // TCGETS
const amdSet = 0x5402 // TCSETS


func determineReadWriteOptions() (uint, uint, error) {
	sysArch := runtime.GOARCH

	switch sysArch {
		case "arm64":
			return armGet, armSet, nil

		case "amd64":
			return amdGet, amdSet, nil

		default:
			return 0, 0, errors.New("Architecture not found")
	}
}
