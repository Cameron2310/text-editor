package editor

import (
	"bufio"
	"fmt"
	"log"

	"golang.org/x/sys/unix"
)

type Buffer struct {
	Text string
	Length int
}

type Position struct {
	X int
	Y int
}

type EditorState struct {
	Content []string
	CursorPos Position
}

type EditorConfig struct {
	Rows int
	cols int
	Pos Position
	StateIdx int
    firstRowToView int
	firstColToView int
}

func (buf *Buffer) appendText(text string) {
	newBuf := buf.Text + text

	if len(newBuf) == 0 {
		return
	}

	buf.Text = newBuf
	buf.Length += len(newBuf)
}


func GetEditorConfig(fd int, req uint) *EditorConfig {
	winConfig, err := unix.IoctlGetWinsize(fd, req)

	if err != nil {
		panic(err)
	}

	return &EditorConfig{Rows: int(winConfig.Row), cols: int(winConfig.Col), Pos: Position{X: 1, Y: 0}, StateIdx: 1, firstRowToView: 0, firstColToView: 0}
}


func drawLeftBorder(rows int, buf *Buffer) {
	for i := range rows {
		buf.appendText("~")
		buf.appendText("\x1b[K")

		if i < rows - 1 {
			buf.appendText("\r\n")
		} 
	}
}


func RefreshScreen(config *EditorConfig, buf *Buffer, editorContent []string) {
	buf.appendText("\x1b[6 q")
	buf.appendText("\x1b[?25l")
	buf.appendText("\x1b[H")

	drawLeftBorder(config.Rows, buf)

    start := config.firstRowToView
    end := start + config.Rows
    text := editorContent[start : end]

	offsetCount := ((config.Pos.X / config.cols) - 1) * config.cols
	lastCol := config.firstColToView + config.cols

	for i, s := range text {
		if offsetCount >= 0 && lastCol >= config.Pos.X {
			if len(s) > config.firstColToView {
				config.firstColToView = offsetCount + (config.Pos.X % config.cols) + 1
				s = s[config.firstColToView:]

			} else {
				s = " "
			} 
		}

		buf.appendText(fmt.Sprintf("\x1b[%d;%dH", i + 1, 0))
		buf.appendText(s)
	}

	var cursorPos string
	if len(editorContent[config.Pos.Y]) > 0 {
		cursorPos = fmt.Sprintf("\x1b[%d;%dH", (config.Pos.Y - start) + 1, config.Pos.X)

	} else {
		cursorPos = fmt.Sprintf("\x1b[%d;%dH", (config.Pos.Y - start) + 1, config.Pos.X + 1)
	}

	buf.appendText(cursorPos)
	buf.appendText("\x1b[?25h")

	fmt.Print(buf.Text)
}


// TODO: fix bug where initial state gets overwritten
func HandleControlKeys(keypress byte, config *EditorConfig, editorContent []string, prevStates []EditorState) ([]string, bool) {
	goBackToPrevState := false

	switch keypress {
        // Enter
		case '\x0d':
			firstHalf := editorContent[:config.Pos.Y + 1]
            secondHalf := make([]string, len(editorContent))

            copy(secondHalf, editorContent)
            secondHalf = secondHalf[config.Pos.Y + 1:]

			newEditorContent := append(firstHalf, "")
			newEditorContent = append(newEditorContent, secondHalf...)

			editorContent = newEditorContent

			config.Pos.Y += 1
			config.Pos.X = 1

            if config.firstRowToView + config.Rows == config.Pos.Y {
                config.firstRowToView += 1
            }

		// Ctrl-z undo
		case '\x1a':
			if config.StateIdx > 0 {
				if config.StateIdx == len(prevStates) && config.StateIdx > 1{
					config.StateIdx -= 2
				} else {
					config.StateIdx -= 1
				}
				goBackToPrevState = true

				config.Pos.X = prevStates[config.StateIdx].CursorPos.X
				config.Pos.Y = prevStates[config.StateIdx].CursorPos.Y
                copy(editorContent, prevStates[config.StateIdx].Content)

				if len(editorContent) == 0  || len(prevStates[config.StateIdx].Content) == 0{
					log.Println("editor content is nil. Resetting...")
					editorContent = make([]string, config.Rows)
                    config.StateIdx = 1
				}
			}

		// Ctrl-r Redo
		case '\x12':
			if config.StateIdx + 1 < len(prevStates) {
				config.StateIdx += 1
				goBackToPrevState = true

				config.Pos.X = prevStates[config.StateIdx].CursorPos.X
				config.Pos.Y = prevStates[config.StateIdx].CursorPos.Y
                copy(editorContent, prevStates[config.StateIdx].Content)
			}

		// Backspace
		case '\x7f':
			if config.Pos.X > 1 {
				config.Pos.X -= 1
				runes := []rune(editorContent[config.Pos.Y])

				if config.Pos.X >= len(runes) {
					editorContent[config.Pos.Y] = string(runes[:config.Pos.X - 1])
				} else {
					editorContent[config.Pos.Y] = string(runes[:config.Pos.X]) + string(runes[config.Pos.X + 1:])
				}
			}
	}

	return editorContent, goBackToPrevState
}


func HandleKeyPress(keypress string, reader *bufio.Reader, config *EditorConfig, editorContent []string) bool {
    shouldStateChange := false

	switch keypress {
		// TODO: fix bug where if [ key pressed it requires second [
		case "[":
			nextVal, _ := reader.ReadByte()

			switch string(nextVal) {
				case "A": // up
					if config.Pos.Y - 1 >= 0 {
						config.Pos.Y -= 1

                        if config.Pos.Y < config.firstRowToView {
                            config.firstRowToView -= 1
                        }
                        
                        row_len := len(editorContent[config.Pos.Y])
                        if row_len > 0 {
                            config.Pos.X = row_len + 1
                        } else {
                            config.Pos.X = 1
                        }
					}

                    shouldStateChange = true

				case "B": // down
					if config.Pos.Y + 1 < len(editorContent) {
						config.Pos.Y += 1

						offsetCount := ((config.Pos.Y / config.Rows) - 1) * config.Rows
						lastRow := config.firstRowToView + config.Rows

						if offsetCount >= 0 && lastRow == config.Pos.Y {
							config.firstRowToView = offsetCount + (config.Pos.Y % config.Rows) + 1
						}

						row_len := len(editorContent[config.Pos.Y])
						if row_len > 0 {
							config.Pos.X = row_len + 1
						} else {
							config.Pos.X = 1
						}
					}

                    shouldStateChange = true

				case "C": // right
                    row_len := len(editorContent[config.Pos.Y])

                    if config.Pos.X + 1 <= row_len + 1 {
                        config.Pos.X += 1
                    } 

                    shouldStateChange = true

				case "D": // left
					if config.Pos.X - 1 > 0 {
						config.Pos.X -= 1
					}

                    shouldStateChange = true

				default:
					editorContent[config.Pos.Y] += string(nextVal)
                    config.Pos.X += 1
			}

		default:
			if len(editorContent[config.Pos.Y]) > 0 {
				firstHalf := editorContent[config.Pos.Y][:config.Pos.X - 1]
				secondHalf := editorContent[config.Pos.Y][config.Pos.X - 1:]

				firstHalf += keypress + secondHalf
				editorContent[config.Pos.Y] = firstHalf

			} else {
				editorContent[config.Pos.Y] += keypress
			}

            config.Pos.X += 1
	}

    return shouldStateChange
}
