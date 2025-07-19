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
}

func main() {
	filePath := os.Args[1]
	fd := unix.Stdin

	if len(filePath) == 0 {
		fmt.Println("File path not found...")
		return
	}

	clearScreen()
	editorConfig := getEditorConfig(fd, unix.TIOCGWINSZ)
	drawTerminal(editorConfig)
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
	savedData := []string{string(text)}

	// if data != nil {
	// 	savedData = data
	//
	// 	for _, val := range savedData {
	// 		fmt.Print(val)
	// 	}
	// }


	quitCmd := 17

	for text != byte(quitCmd) {
		text, _ = reader.ReadByte()
		savedData = append(savedData, string(text))

		if (int(text) > 0 && int(text) <= 31) || int(text) == 127 {
			handleControlKeys(int(text))
		} else {
			handleKeyPress(string(text), reader)
		}
	}

	// Disable raw mode at exit
	defer disableRawMode(&oldState, fd, ioctlSet)
	// defer clearScreen()

	// defer writeData(filePath, savedData)
}

func getEditorConfig(fd int, req uint) editorConfig {
	winConfig, err := unix.IoctlGetWinsize(fd, req)

	if err != nil {
		panic(err)
	}

	return editorConfig{rows: int(winConfig.Row), cols: int(winConfig.Col)}
}


func drawTerminal(config editorConfig) {
	drawRows(config.rows)
}


// TODO: make all keymappings either octal or hex
func handleControlKeys(keypress int) {
	switch keypress {
		case 127:
			fmt.Print("\010\033[P")
	}
}

func handleKeyPress(keypress string, reader *bufio.Reader) {
	switch keypress {
		// TODO: fix bug where if [ key pressed it requires second [
		case "[":
			nextVal, _ := reader.ReadByte()

			switch string(nextVal) {
				case "D":
					fmt.Print("\033[D")

				case "C":
					fmt.Print("\033[C")

				case "A":
					fmt.Print("\033[A")

				case "B":
					fmt.Print("\033[B")

				default:
					fmt.Print(string(nextVal))
			}

		default:
			fmt.Print(keypress)
	}
}

func drawRows(rows int) {
	for range rows - 1 {
		fmt.Print("~\r\n")
	}

	fmt.Print("~")
	fmt.Print("\x1b[H")
}

func clearScreen() {
	fmt.Println("\x1b[2J\x1b[H")
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

	// Apply new terminal settings
	err := unix.IoctlSetTermios(fd, ioctlSet, term)

	if err != nil {
		panic(err)
	}

	return term
}
