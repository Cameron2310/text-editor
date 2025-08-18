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

	return &editorConfig{rows: int(winConfig.Row), cols: int(winConfig.Col), x: 1, y: 0, stateIdx: 1, firstRowToView: 0, firstColToView: 0}
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

	offsetCount := ((config.x / config.cols) - 1) * config.cols
	lastCol := config.firstColToView + config.cols

	for i, s := range text {
		if offsetCount >= 0 && lastCol >= config.x {
			if len(s) > config.firstColToView {
				config.firstColToView = offsetCount + (config.x % config.cols) + 1
				s = s[config.firstColToView:]

			} else {
				s = " "
			} 
		}

		buf.appendText(fmt.Sprintf("\x1b[%d;%dH", i + 1, 0))
		buf.appendText(s)
	}

	var cursorPos string
	if len(editorContent[config.y]) > 0 {
		cursorPos = fmt.Sprintf("\x1b[%d;%dH", (config.y - start) + 1, config.x)

	} else {
		cursorPos = fmt.Sprintf("\x1b[%d;%dH", (config.y - start) + 1, config.x + 1)
	}

	buf.appendText(cursorPos)
	buf.appendText("\x1b[?25h")

	fmt.Print(buf.text)
}


// TODO: make all keymappings either octal or hex
// TODO: fix bug where initial state gets overwritten
func handleControlKeys(keypress int, config *editorConfig, editorContent []string, prevStates []editorState) ([]string, bool) {
	goBackToPrevState := false

	switch keypress {
		case 13:
			firstHalf := editorContent[:config.y + 1]
            secondHalf := make([]string, len(editorContent))

            copy(secondHalf, editorContent)
            secondHalf = secondHalf[config.y + 1:]

			newEditorContent := append(firstHalf, "")
			newEditorContent = append(newEditorContent, secondHalf...)

			editorContent = newEditorContent

			config.y += 1
			config.x = 1

            if config.firstRowToView + config.rows == config.y {
                config.firstRowToView += 1
            }

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

		// Ctrl-r Redo
		case 18:
			if config.stateIdx + 1 < len(prevStates) {
				config.stateIdx += 1
				goBackToPrevState = true

				config.x = prevStates[config.stateIdx].cursorPos.x
				config.y = prevStates[config.stateIdx].cursorPos.y
				editorContent = prevStates[config.stateIdx].content
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

                        if config.y < config.firstRowToView {
                            config.firstRowToView -= 1
                        }
                        
                        row_len := len(editorContent[config.y])
                        if row_len > 0 {
                            config.x = row_len + 1
                        } else {
                            config.x = 1
                        }
					}

				case "B": // down
					if config.y + 1 < len(editorContent) {
						config.y += 1

						offsetCount := ((config.y / config.rows) - 1) * config.rows
						lastRow := config.firstRowToView + config.rows

						if offsetCount >= 0 && lastRow == config.y {
							config.firstRowToView = offsetCount + (config.y % config.rows) + 1
						}

						row_len := len(editorContent[config.y])
						if row_len > 0 {
							config.x = row_len + 1
						} else {
							config.x = 1
						}
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
			if len(editorContent[config.y]) > 0 {
				firstHalf := editorContent[config.y][:config.x - 1]
				secondHalf := editorContent[config.y][config.x - 1:]

				firstHalf += keypress + secondHalf
				editorContent[config.y] = firstHalf

			} else {
				editorContent[config.y] += keypress
			}

            config.x += 1
	}
}
