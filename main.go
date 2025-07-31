package main

import (
	"bufio"
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

type editorConfig struct {
	rows int
	cols int
	x int
	y int
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

	// filePath := os.Args[1]
	fd := unix.Stdin

	editorConfig := getEditorConfig(fd, unix.TIOCGWINSZ)
	buf := &buffer{text: "", length: 0}
	editorContent := make([]string, editorConfig.rows)

	ioctlGet, ioctlSet, err := determineReadWriteOptions()

	if err != nil {
		panic(err)
	}

	term, err := unix.IoctlGetTermios(fd, ioctlGet)
	oldState := *term

	if err != nil {
		panic(err)
	}

	enableRawMode(term, fd, ioctlSet)
	// data := readData(filePath)

	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadByte()

	// if len(data) > 0 {
	// 	for i, val := range data {
	// 		fmt.Printf("\033[%d;0H", i)
	// 		fmt.Print("~ " + val)
	// 	}
	// }

	quitCmd := 17

	for text != byte(quitCmd) {
		refreshScreen(editorConfig, buf, editorContent)
		text, _ = reader.ReadByte()

		if (int(text) > 0 && int(text) <= 31) || int(text) == 127 {
			handleControlKeys(int(text))
		} else {
			handleKeyPress(string(text), reader, editorConfig, editorContent)
		}
	}

	// Disable raw mode at exit
	defer disableRawMode(&oldState, fd, ioctlSet)
	defer fmt.Println("\x1b[2J")
}

func getEditorConfig(fd int, req uint) *editorConfig {
	winConfig, err := unix.IoctlGetWinsize(fd, req)

	if err != nil {
		panic(err)
	}

	return &editorConfig{rows: int(winConfig.Row), cols: int(winConfig.Col), x: 1, y: 0}
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
func handleControlKeys(keypress int) {
	switch keypress {
		case 127:
			fmt.Print("\010\033[P")
	}
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
                            config.x = row_len
                        } else {
                            config.x = 1
                        }
					}

				case "B": // down
					config.y += 1
                    row_len := len(editorContent[config.y])

                    if row_len > 0 {
                        config.x = row_len
                    } else {
                        config.x = 1
                    }

				case "C": // right
                    row_len := len(editorContent[config.y])
                    if config.x + 1 <= row_len {
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
