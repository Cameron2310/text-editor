package main

import (
	"bufio"
	"errors"
	"log"
	"os"
	"runtime"
	"text-editor/editor"
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


func readData(filePath string, config *editor.EditorConfig) {
	content, err := os.Open(filePath)
	var returnVal []string

	if err != nil {
        log.Println("Err -->", err)
	}

	defer content.Close()
	fileScanner := bufio.NewScanner(content)

	// TODO: change the way data is read
	const maxCapacity = 1024 * 1024
	fileScanner.Buffer(make([]byte, 0, maxCapacity), maxCapacity)

	fileScanner.Split(bufio.ScanLines)

	for fileScanner.Scan() {
		returnVal = append(returnVal, fileScanner.Text())
	}
	
    if len(returnVal) < config.Rows {
        for len(returnVal) < config.Rows {
            returnVal = append(returnVal, "")
        }
    }

	log.Printf("Reading from %v\n", filePath)

    config.Content = returnVal
}


func writeData(filePath string, data []string) {
	f, err := os.OpenFile(filePath, os.O_CREATE | os.O_WRONLY, 0644)

	if err != nil {
		panic(err)
	}

	defer f.Close()

	for _, str := range data {
		if len(str) > 0 {
            if str[len(str) - 1] != '\n' {
                f.WriteString(str + "\n")
            }

		} else {
            f.WriteString("\n")
        }
	}

	f.Sync()
}

