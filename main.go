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
	fd := unix.Stdin
	term, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	oldState := state{term: *term}

	if err != nil {
		panic(err)
	}

	enableRawMode(term, fd)

	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadByte()
	for (text != byte('q')) {
		text, _ = reader.ReadByte()

		if (int(text) > 0 && int(text) <= 31) {
			fmt.Print(int(text))
		} else {
			fmt.Print(string(text))
		}
	}
	
	// Disable raw mode at exit
	defer disableRawMode(&oldState, fd)
}


func disableRawMode(state *state, fd int) {
	err := unix.IoctlSetTermios(fd, unix.TCSETSF, &state.term) 

	if err != nil {
		panic(err)
	}
}


func enableRawMode(term *unix.Termios, fd int) *unix.Termios {
	term.Lflag &^= unix.ECHO | unix.ECHONL | unix.ICANON
	
	// Apply new terminal settings
	err := unix.IoctlSetTermios(fd, unix.TCSETAF, term) 

	if err != nil {
		panic(err)
	}

	return term
}
