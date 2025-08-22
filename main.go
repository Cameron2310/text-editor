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

	editorConfig := editor.GetEditorConfig(fd, unix.TIOCGWINSZ)
	buf := &editor.Buffer{Text: "", Length: 0}

	ioctlGet, ioctlSet, err := determineReadWriteOptions()

	if err != nil {
		panic(err)
	}

	data := readData(filePath, editorConfig)

	var prevStates []editor.EditorState
	prevStates = append(prevStates, editor.EditorState{Content: []string{}, CursorPos: editor.Position{X: editorConfig.Pos.X, Y: editorConfig.Pos.Y}})

	editorContent := make([]string, editorConfig.Rows)
	lenData := len(data)

	if lenData > 0 {
		editorContent = data
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

		editor.RefreshScreen(editorConfig, buf, editorContent)
		text, _ = reader.ReadByte()
		var goBackToPrevState bool

		if (int(text) > 0 && int(text) <= 31) || int(text) == 127 {
			editorContent, goBackToPrevState = editor.HandleControlKeys(text, editorConfig, editorContent, prevStates)
            goBackToPrevState = true

		} else {
			goBackToPrevState = editor.HandleKeyPress(string(text), reader, editorConfig, editorContent)
		}

        if !goBackToPrevState {
		    if slices.Equal(editorContent, prevStates[len(prevStates) - 1].Content) == false {
				tmp := make([]string, len(editorContent))
				copy(tmp, editorContent)

				newState := editor.EditorState{Content: tmp, CursorPos: editor.Position{X: editorConfig.Pos.X, Y: editorConfig.Pos.Y}}

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
	defer writeData(filePath, editorContent)
	defer fmt.Println("\x1b[2J")
}
