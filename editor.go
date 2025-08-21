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

type editorConfig struct {
	rows int
	cols int
	pos position
	stateIdx int
    firstRowToView int
	firstColToView int
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

	return &editorConfig{rows: int(winConfig.Row), cols: int(winConfig.Col), pos: position{x: 1, y: 0}, stateIdx: 1, firstRowToView: 0, firstColToView: 0}
}


func drawLeftBorder(rows int, buf *buffer) {
	for i := range rows {
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

    start := config.firstRowToView
    end := start + config.rows
    text := editorContent[start : end]

	offsetCount := ((config.pos.x / config.cols) - 1) * config.cols
	lastCol := config.firstColToView + config.cols

	for i, s := range text {
		if offsetCount >= 0 && lastCol >= config.pos.x {
			if len(s) > config.firstColToView {
				config.firstColToView = offsetCount + (config.pos.x % config.cols) + 1
				s = s[config.firstColToView:]

			} else {
				s = " "
			} 
		}

		buf.appendText(fmt.Sprintf("\x1b[%d;%dH", i + 1, 0))
		buf.appendText(s)
	}

	var cursorPos string
	if len(editorContent[config.pos.y]) > 0 {
		cursorPos = fmt.Sprintf("\x1b[%d;%dH", (config.pos.y - start) + 1, config.pos.x)

	} else {
		cursorPos = fmt.Sprintf("\x1b[%d;%dH", (config.pos.y - start) + 1, config.pos.x + 1)
	}

	buf.appendText(cursorPos)
	buf.appendText("\x1b[?25h")

	fmt.Print(buf.text)
}


// TODO: fix bug where initial state gets overwritten
func handleControlKeys(keypress byte, config *editorConfig, editorContent []string, prevStates []editorState) ([]string, bool) {
	goBackToPrevState := false

	switch keypress {
        // Enter
		case '\x0d':
			firstHalf := editorContent[:config.pos.y + 1]
            secondHalf := make([]string, len(editorContent))

            copy(secondHalf, editorContent)
            secondHalf = secondHalf[config.pos.y + 1:]

			newEditorContent := append(firstHalf, "")
			newEditorContent = append(newEditorContent, secondHalf...)

			editorContent = newEditorContent

			config.pos.y += 1
			config.pos.x = 1

            if config.firstRowToView + config.rows == config.pos.y {
                config.firstRowToView += 1
            }

		// Ctrl-z undo
		case '\x1a':
			if config.stateIdx > 0 {
				if config.stateIdx == len(prevStates) && config.stateIdx > 1{
					config.stateIdx -= 2
				} else {
					config.stateIdx -= 1
				}
				goBackToPrevState = true

				config.pos.x = prevStates[config.stateIdx].cursorPos.x
				config.pos.y = prevStates[config.stateIdx].cursorPos.y
                copy(editorContent, prevStates[config.stateIdx].content)

				if len(editorContent) == 0  || len(prevStates[config.stateIdx].content) == 0{
					log.Println("editor content is nil. Resetting...")
					editorContent = make([]string, config.rows)
                    config.stateIdx = 1
				}
			}

		// Ctrl-r Redo
		case '\x12':
			if config.stateIdx + 1 < len(prevStates) {
				config.stateIdx += 1
				goBackToPrevState = true

				config.pos.x = prevStates[config.stateIdx].cursorPos.x
				config.pos.y = prevStates[config.stateIdx].cursorPos.y
                copy(editorContent, prevStates[config.stateIdx].content)
			}

		// Backspace
		case '\x7f':
			if config.pos.x > 1 {
				config.pos.x -= 1
				runes := []rune(editorContent[config.pos.y])

				if config.pos.x >= len(runes) {
					editorContent[config.pos.y] = string(runes[:config.pos.x - 1])
				} else {
					editorContent[config.pos.y] = string(runes[:config.pos.x]) + string(runes[config.pos.x + 1:])
				}
			}
	}

	return editorContent, goBackToPrevState
}


func handleKeyPress(keypress string, reader *bufio.Reader, config *editorConfig, editorContent []string) bool {
    shouldStateChange := false

	switch keypress {
		// TODO: fix bug where if [ key pressed it requires second [
		case "[":
			nextVal, _ := reader.ReadByte()

			switch string(nextVal) {
				case "A": // up
					if config.pos.y - 1 >= 0 {
						config.pos.y -= 1

                        if config.pos.y < config.firstRowToView {
                            config.firstRowToView -= 1
                        }
                        
                        row_len := len(editorContent[config.pos.y])
                        if row_len > 0 {
                            config.pos.x = row_len + 1
                        } else {
                            config.pos.x = 1
                        }
					}

                    shouldStateChange = true

				case "B": // down
					if config.pos.y + 1 < len(editorContent) {
						config.pos.y += 1

						offsetCount := ((config.pos.y / config.rows) - 1) * config.rows
						lastRow := config.firstRowToView + config.rows

						if offsetCount >= 0 && lastRow == config.pos.y {
							config.firstRowToView = offsetCount + (config.pos.y % config.rows) + 1
						}

						row_len := len(editorContent[config.pos.y])
						if row_len > 0 {
							config.pos.x = row_len + 1
						} else {
							config.pos.x = 1
						}
					}

                    shouldStateChange = true

				case "C": // right
                    row_len := len(editorContent[config.pos.y])

                    if config.pos.x + 1 <= row_len + 1 {
                        config.pos.x += 1
                    } 

                    shouldStateChange = true

				case "D": // left
					if config.pos.x - 1 > 0 {
						config.pos.x -= 1
					}

                    shouldStateChange = true

				default:
					editorContent[config.pos.y] += string(nextVal)
                    config.pos.x += 1
			}

		default:
			if len(editorContent[config.pos.y]) > 0 {
				firstHalf := editorContent[config.pos.y][:config.pos.x - 1]
				secondHalf := editorContent[config.pos.y][config.pos.x - 1:]

				firstHalf += keypress + secondHalf
				editorContent[config.pos.y] = firstHalf

			} else {
				editorContent[config.pos.y] += keypress
			}

            config.pos.x += 1
	}

    return shouldStateChange
}
