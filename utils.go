package main

import (
	"errors"
	"fmt"
	"os"
	"runtime"
)


func determineReadWriteOptions() (uint, uint, error) {
	const armGet = 0x40487413 // TIOCGETA
	const armSet = 0x80487414 // TIOCSETA

	const amdGet = 0x5401 // TCGETS
	const amdSet = 0x5402 // TCSETS
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


func readData(filePath string) []string {
	content, err := os.ReadFile(filePath)
	var returnVal []string

	if err != nil {
		panic(err)
	}

	for _, val := range content {
		returnVal = append(returnVal, string(val))
	}

	return returnVal
}


func writeData(filePath string, data []string) {
	f, err := os.OpenFile(filePath, os.O_APPEND | os.O_CREATE | os.O_WRONLY, 0644)

	if err != nil {
		panic(err)
	}

	defer f.Close()

	for _, str := range data {
		fmt.Print(str)
		_, err := f.WriteString(str)

		if err != nil {
			panic(err)
		}
	}

	f.Sync()
}
