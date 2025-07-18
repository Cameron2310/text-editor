package main

import (
	"bufio"
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)


type state struct {
	term unix.Termios
}


func main() {
	clearScreen()
	drawRows()
	ioctlGet, ioctlSet, err := determineReadWriteOptions()

	if err != nil {
		panic(err)
	}

	fd := unix.Stdin
	term, err := unix.IoctlGetTermios(fd, ioctlGet)
	oldState := state{term: *term}

	if err != nil {
		panic(err)
	}

	enableRawMode(term, fd, ioctlSet)

	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadByte()
	savedData := []string{string(text)}

	quitCmd := 17

	for (text != byte(quitCmd)) {
		text, _ = reader.ReadByte()
		savedData = append(savedData, string(text))

		if (int(text) > 0 && int(text) <= 31) {
			// fmt.Print(int(text))
		} else {
			handleKeyPress(string(text), *reader)
		}
	}
	
	// Disable raw mode at exit
	defer disableRawMode(&oldState, fd, ioctlSet)
	// defer clearScreen()
	fmt.Println(savedData)
}


func handleKeyPress(keypress string, reader bufio.Reader) {
	switch keypress {
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
			}
			
		default:
			fmt.Print(keypress)
	}
}


func drawRows() {
	for range 24 {
		fmt.Print("~\r\n")
	}

	fmt.Print("\x1b[H")
}


func clearScreen() {
	fmt.Println("\x1b[2J\x1b[H")	
}


func disableRawMode(state *state, fd int, ioctlSet uint) {
	err := unix.IoctlSetTermios(fd, ioctlSet, &state.term) 

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
