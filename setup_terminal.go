package main

import (
	"fmt"

	"golang.org/x/sys/unix"
)


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

	fmt.Println("\x1B\x5B\x3F\x37\x6C")

	return term
}
