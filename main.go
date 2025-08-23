package main

import (
	"bufio"
	"text-editor/editor"
	"fmt"
	"log"
	"os"
	"slices"

	"golang.org/x/sys/unix"
)


func setupLogging() *os.File {
 	err := os.MkdirAll("./logs", os.ModePerm)

	if err != nil {
		errMsg := fmt.Sprintf("Could not create directory due to %v", err)
		panic(errMsg)
	}   

    logFile, err := os.OpenFile("logs/text-editor.log", os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)

    if err != nil {
        log.Panic(err)
    }

    log.SetOutput(logFile)
    log.SetFlags(log.Lshortfile | log.LstdFlags)

    return logFile
}


func main() {
	if len(os.Args) < 2 {
		fmt.Println("File path not found...")
		return
	}

    logFile := setupLogging()
	defer logFile.Close()

	filePath := os.Args[1]
	fd := unix.Stdin

	editorConfig := editor.NewEditorConfig(fd, unix.TIOCGWINSZ)
	buf := &editor.Buffer{Text: "", Length: 0}

	ioctlGet, ioctlSet, err := determineReadWriteOptions()

	if err != nil {
		panic(err)
	}

	var prevStates []*editor.Snapshot
	prevStates = append(prevStates, editorConfig.CreateSnapshot())

    // TODO: possibly move this elsewhere
    readData(filePath, editorConfig)
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

		editor.RefreshScreen(editorConfig, buf)
		text, _ = reader.ReadByte()
		var goBackToPrevState bool

		if (int(text) > 0 && int(text) <= 31) || int(text) == 127 {
			editorConfig.Content, goBackToPrevState = editor.HandleControlKeys(text, editorConfig, prevStates)
            goBackToPrevState = true

		} else {
			goBackToPrevState = editor.HandleKeyPress(string(text), reader, editorConfig)
		}

        if !goBackToPrevState {
		    if slices.Equal(editorConfig.Content, prevStates[len(prevStates) - 1].Content) == false {
				newState := editorConfig.CreateSnapshot()

				if editorConfig.StateIdx < len(prevStates) {
					prevStates[editorConfig.StateIdx] = newState

				} else {
					prevStates = append(prevStates, newState)
				}
			}

            editorConfig.StateIdx += 1
		}
		
		buf.Text = ""
	}

	defer disableRawMode(&oldState, fd, ioctlSet)
	defer writeData(filePath, editorConfig.Content)
	defer fmt.Println("\x1b[2J")
}
