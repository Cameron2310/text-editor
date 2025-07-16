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
	quitCmd := 17

	for (text != byte(quitCmd)) {
		text, _ = reader.ReadByte()

		if (int(text) > 0 && int(text) <= 31) {
			fmt.Print(int(text))
		} else {
			fmt.Printf("%v\r\n", string(text))
		}
	}
	
	// Disable raw mode at exit
	defer disableRawMode(&oldState, fd, ioctlSet)
	defer clearScreen()
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
