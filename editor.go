package main

import (
	"bufio"
	"fmt"
	"log"

	"golang.org/x/sys/unix"
)

type buffer struct {
	text string
	length int
}

type position struct {
	x int
	y int
}

type editorState struct {
	content []string
	cursorPos position
}

// TODO: replace x & y with position struct
type editorConfig struct {
	rows int
	cols int
	x int
	y int
	stateIdx int
}

func (buf *buffer) appendText(text string) {
	newBuf := buf.text + text

	if len(newBuf) == 0 {
		return
	}

	buf.text = newBuf
	buf.length += len(newBuf)
}


func getEditorConfig(fd int, req uint) *editorConfig {
	winConfig, err := unix.IoctlGetWinsize(fd, req)

	if err != nil {
		panic(err)
	}

	return &editorConfig{rows: int(winConfig.Row), cols: int(winConfig.Col), x: 1, y: 0, stateIdx: 1}
}


func drawLeftBorder(rows int, buf *buffer) {
	for i := range rows - 1 {
		buf.appendText("~")
		buf.appendText("\x1b[K")

		if i < rows - 1 {
			buf.appendText("\r\n")
		} 
	}
}


func refreshScreen(config *editorConfig, buf *buffer, editorContent []string) {
	buf.appendText("\x1b[6 q")
	buf.appendText("\x1b[?25l")
	buf.appendText("\x1b[H")

	drawLeftBorder(config.rows, buf)

	for i, s := range editorContent {
		buf.appendText(fmt.Sprintf("\x1b[%d;%dH", i + 1, 2))
		buf.appendText(s)
	}

	cursorPos := fmt.Sprintf("\x1b[%d;%dH", config.y + 1, config.x + 1)
	buf.appendText(cursorPos)

	buf.appendText("\x1b[?25h")
	fmt.Print(buf.text)
}


// TODO: make all keymappings either octal or hex
func handleControlKeys(keypress int, config *editorConfig, editorContent []string, prevStates []editorState) ([]string, bool) {
	goBackToPrevState := false

	switch keypress {
		// Ctrl-z
		case 26:
			if config.stateIdx > 0 {
				if config.stateIdx == len(prevStates) {
					config.stateIdx -= 2
				} else {
					config.stateIdx -= 1
				}
				goBackToPrevState = true

				config.x = prevStates[config.stateIdx].cursorPos.x
				config.y = prevStates[config.stateIdx].cursorPos.y
				editorContent = prevStates[config.stateIdx].content

				if len(editorContent) == 0 {
					log.Println("editor content is nil. Resetting...")
					editorContent = make([]string, config.rows)
				}
			}

		// Backspace
		case 127:
			if config.x > 1 {
				config.x -= 1
				runes := []rune(editorContent[config.y])

				if config.x >= len(runes) {
					editorContent[config.y] = string(runes[:config.x - 1])
				} else {
					editorContent[config.y] = string(runes[:config.x]) + string(runes[config.x + 1:])
				}
			}
	}

	return editorContent, goBackToPrevState
}

func handleKeyPress(keypress string, reader *bufio.Reader, config *editorConfig, editorContent []string) {
	switch keypress {
		// TODO: fix bug where if [ key pressed it requires second [
		case "[":
			nextVal, _ := reader.ReadByte()

			switch string(nextVal) {
				case "A": // up
					if config.y - 1 >= 0 {
						config.y -= 1
                        row_len := len(editorContent[config.y])

                        if row_len > 0 {
                            config.x = row_len + 1
                        } else {
                            config.x = 1
                        }
					}

				case "B": // down
					config.y += 1
                    row_len := len(editorContent[config.y])

                    if row_len > 0 {
                        config.x = row_len + 1
                    } else {
                        config.x = 1
                    }

				case "C": // right
                    row_len := len(editorContent[config.y])
                    if config.x + 1 <= row_len + 1 {
                        config.x += 1
                    }

				case "D": // left
					if config.x - 1 > 0 {
						config.x -= 1
					}

				default:
					editorContent[config.y] += string(nextVal)
                    config.x += 1
			}

		default:
			editorContent[config.y] += keypress
            config.x += 1
	}

}
