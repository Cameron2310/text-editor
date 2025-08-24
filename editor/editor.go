package editor

import (
	"bufio"
	"fmt"
	"log"
)

type Buffer struct {
	Text string
	Length int
}

type Position struct {
	X int
	Y int
}


func (buf *Buffer) appendText(text string) {
	newBuf := buf.Text + text

	if len(newBuf) == 0 {
		return
	}

	buf.Text = newBuf
	buf.Length += len(newBuf)
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


func RefreshScreen(config *EditorConfig, buf *Buffer) {
	buf.appendText("\x1b[6 q")
	buf.appendText("\x1b[?25l")
	buf.appendText("\x1b[H")

	drawLeftBorder(config.Rows, buf)

    start := config.firstRowToView
    end := start + config.Rows
    text := config.Content[start : end]

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
	if len(config.Content[config.Pos.Y]) > 0 {
		cursorPos = fmt.Sprintf("\x1b[%d;%dH", (config.Pos.Y - start) + 1, config.Pos.X)

	} else {
		cursorPos = fmt.Sprintf("\x1b[%d;%dH", (config.Pos.Y - start) + 1, config.Pos.X + 1)
	}

	buf.appendText(cursorPos)
	buf.appendText("\x1b[?25h")

	fmt.Print(buf.Text)
}


func HandleControlKeys(keypress byte, config *EditorConfig, prevStates []*Snapshot) ([]string, bool) {
	changeState := false

	switch keypress {
        // Enter
		case '\x0d':
			firstHalf := config.Content[:config.Pos.Y + 1]
            secondHalf := make([]string, len(config.Content))

            copy(secondHalf, config.Content)
            secondHalf = secondHalf[config.Pos.Y + 1:]

			newEditorContent := append(firstHalf, "")
			newEditorContent = append(newEditorContent, secondHalf...)

			config.Content = newEditorContent

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
				changeState = true

				restoredContent, restoredPos := prevStates[config.StateIdx].Restore() 
				config.Pos.X = restoredPos.X
				config.Pos.Y = restoredPos.Y

                copy(config.Content, restoredContent)

				if len(config.Content) == 0  || len(restoredContent) == 0{
					log.Println("editor content is nil. Resetting...")
					config.Content = make([]string, config.Rows)
                    config.StateIdx = 1
				}
			}

		// Ctrl-r Redo
		case '\x12':
			if config.StateIdx + 1 < len(prevStates) {
				config.StateIdx += 1
				changeState = true

				restoredContent, restoredPos := prevStates[config.StateIdx].Restore() 
				config.Pos.X = restoredPos.X
				config.Pos.Y = restoredPos.Y

                copy(config.Content, restoredContent)
			}

		// Backspace
		case '\x7f':
			if config.Pos.X > 1 {
				config.Pos.X -= 1
				runes := []rune(config.Content[config.Pos.Y])

				if config.Pos.X >= len(runes) {
					config.Content[config.Pos.Y] = string(runes[:config.Pos.X - 1])
				} else {
					config.Content[config.Pos.Y] = string(runes[:config.Pos.X]) + string(runes[config.Pos.X + 1:])
				}
			}
	}

	return config.Content, changeState
}


func HandleKeyPress(keypress string, reader *bufio.Reader, config *EditorConfig) bool {
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
                        
                        row_len := len(config.Content[config.Pos.Y])
                        if row_len > 0 {
                            config.Pos.X = row_len + 1

                        } else {
                            config.Pos.X = 1
                        }
					}

                    shouldStateChange = true

				case "B": // down
					if config.Pos.Y + 1 < len(config.Content) {
						config.Pos.Y += 1

						offsetCount := ((config.Pos.Y / config.Rows) - 1) * config.Rows
						lastRow := config.firstRowToView + config.Rows

						if offsetCount >= 0 && lastRow == config.Pos.Y {
							config.firstRowToView = offsetCount + (config.Pos.Y % config.Rows) + 1
						}

						row_len := len(config.Content[config.Pos.Y])
						if row_len > 0 {
							config.Pos.X = row_len + 1

						} else {
							config.Pos.X = 1
						}
					}

                    shouldStateChange = true

				case "C": // right
                    row_len := len(config.Content[config.Pos.Y])

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
					config.Content[config.Pos.Y] += string(nextVal)
                    config.Pos.X += 1
			}

		default:
			if len(config.Content[config.Pos.Y]) > 0 {
				firstHalf := config.Content[config.Pos.Y][:config.Pos.X - 1]
				secondHalf := config.Content[config.Pos.Y][config.Pos.X - 1:]

				firstHalf += keypress + secondHalf
				config.Content[config.Pos.Y] = firstHalf

			} else {
				config.Content[config.Pos.Y] += keypress
			}
            config.Pos.X += 1
	}

    return shouldStateChange
}
