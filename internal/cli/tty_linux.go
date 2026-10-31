//go:build linux

package cli

import (
	"os"
	"syscall"
	"unsafe"
)

// ttyTerminal reports whether fd refers to a terminal. Files like
// /dev/null are char devices but not terminals, so an explicit
// TIOCGWINSZ ioctl is used instead of a ModeCharDevice check.
func ttyTerminal(fd uintptr) bool {
	var ws struct {
		rows, cols, xpixel, ypixel uint16
	}
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL, fd, syscall.TIOCGWINSZ, uintptr(unsafe.Pointer(&ws)),
	)
	return errno == 0
}

func ttyInteractive() bool {
	return ttyTerminal(os.Stdin.Fd()) && ttyTerminal(os.Stdout.Fd())
}
