package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"slices"

	"golang.org/x/sys/unix"
)

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

type buffer struct {
	text string
	length int
}


func (buf *buffer) appendText(text string) {
	newBuf := buf.text + text

	if len(newBuf) == 0 {
		return
	}

	buf.text = newBuf
	buf.length += len(newBuf)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("File path not found...")
		return
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

	if len(data) > 0 {
		copy(editorContent, data)
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

				prevStates = append(prevStates, editorState{content: tmp, cursorPos: position{x: editorConfig.x, y: editorConfig.y}})
				editorConfig.stateIdx += 1
			}
		}

		buf.text = ""
	}

	defer disableRawMode(&oldState, fd, ioctlSet)
	defer writeData(filePath, editorContent)
	defer fmt.Println("\x1b[2J")
}


func getEditorConfig(fd int, req uint) *editorConfig {
	winConfig, err := unix.IoctlGetWinsize(fd, req)

	if err != nil {
		panic(err)
	}

	return &editorConfig{rows: int(winConfig.Row), cols: int(winConfig.Col), x: 1, y: 0, stateIdx: 0}
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
				config.stateIdx -= 1
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


func disableRawMode(term *unix.Termios, fd int, ioctlSet uint) {
	err := unix.IoctlSetTermios(fd, ioctlSet, term)

	if err != nil {
		panic(err)
	}
}

func enableRawMode(term *unix.Termios, fd int, ioctlSet uint) *unix.Termios {
	term.Cflag |= unix.CS8 // sets char mask to 8 bits
	term.Iflag &^= unix.IXON | unix.ICRNL | unix.BRKINT | unix.INPCK | unix.ISTRIP
	term.Lflag &^= unix.ECHO | unix.ECHONL | unix.ICANON | unix.ISIG | unix.IEXTEN
	term.Oflag &^= unix.OPOST

	term.Cc[unix.VMIN] = 1
	term.Cc[unix.VTIME] = 0

	err := unix.IoctlSetTermios(fd, ioctlSet, term)

	if err != nil {
		panic(err)
	}

	return term
}
