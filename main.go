package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"slices"

	"golang.org/x/sys/unix"
)


func main() {
	if len(os.Args) < 2 {
		fmt.Println("File path not found...")
		return
	}

	err := os.MkdirAll("./logs", os.ModePerm)

	if err != nil {
		errMsg := fmt.Sprintf("Could not create directory due to %v", err)
		panic(errMsg)
	}

	logFile, err := os.OpenFile("logs/text-editor.log", os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)

	if err != nil {
		log.Panic(err)
	}

	defer logFile.Close()

	log.SetOutput(logFile)
    log.SetFlags(log.Lshortfile | log.LstdFlags)

	filePath := os.Args[1]
	fd := unix.Stdin

	editorConfig := getEditorConfig(fd, unix.TIOCGWINSZ)
	buf := &buffer{text: "", length: 0}

	ioctlGet, ioctlSet, err := determineReadWriteOptions()

	if err != nil {
		panic(err)
	}

	data := readData(filePath)

	editorContent := make([]string, editorConfig.rows)
	var prevStates []editorState
	prevStates = append(prevStates, editorState{content: []string{}, cursorPos: position{x: editorConfig.x, y: editorConfig.y}})

	lenData := len(data)
	if lenData > 0 {
		editorContent = data
		// copy(editorContent, data)
	}

	term, err := unix.IoctlGetTermios(fd, ioctlGet)
	oldState := *term

	if err != nil {
		panic(err)
	}

	enableRawMode(term, fd, ioctlSet)

	reader := bufio.NewReader(os.Stdin)
	text := byte(0)
	quitCmd := 17

	for {
		if text == byte(quitCmd) {
			break
		}

		refreshScreen(editorConfig, buf, editorContent)
		text, _ = reader.ReadByte()
		var goBackToPrevState bool

		if (int(text) > 0 && int(text) <= 31) || int(text) == 127 {
			editorContent, goBackToPrevState = handleControlKeys(int(text), editorConfig, editorContent, prevStates)
		} else {
			handleKeyPress(string(text), reader, editorConfig, editorContent)
		}

		if slices.Equal(editorContent, prevStates[len(prevStates) - 1].content) == false {
			if !goBackToPrevState {
				tmp := make([]string, len(editorContent))
				copy(tmp, editorContent)

				newState := editorState{content: tmp, cursorPos: position{x: editorConfig.x, y: editorConfig.y}}

				if editorConfig.stateIdx < len(prevStates) {
					prevStates[editorConfig.stateIdx] = newState
				} else {
					prevStates = append(prevStates, newState)
				}
				
				editorConfig.stateIdx += 1
			}
		}
		
		buf.text = ""
	}

	defer disableRawMode(&oldState, fd, ioctlSet)
	defer writeData(filePath, editorContent)
	defer fmt.Println("\x1b[2J")
}
