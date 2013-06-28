package terminal

import (
	"fmt"
	"os"
	"runtime"
	"syscall"
	"unsafe"
)

type winsize struct {
	Row    uint16
	Col    uint16
	Xpixel uint16
	Ypixel uint16
}

func getWinsize() (*winsize, error) {
	ws := new(winsize)

	var _TIOCGWINSZ int64

	switch runtime.GOOS {
	case "linux":
		_TIOCGWINSZ = 0x5413
	case "darwin":
		_TIOCGWINSZ = 1074295912
	}

	r1, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(syscall.Stdin),
		uintptr(_TIOCGWINSZ),
		uintptr(unsafe.Pointer(ws)),
	)

	if int(r1) == -1 {
		return nil, os.NewSyscallError("GetWinsize", errno)
	}
	return ws, nil
}

// Clear screen and move coursor to top left corner
// http://stackoverflow.com/questions/1348563/clearing-output-of-a-terminal-program-linux-c-c
func Clear() {
	fmt.Print("\033[2J\033[1;1H")
}

func MoveTo(str string, x int, y int) string {
	return fmt.Sprintf("\033[%d;%dH%s", x, y, str)
}

func Bold(str string) string {
	return fmt.Sprintf("\033[1m%s\033[0m", str)
}

func Width() int {
	ws, _ := getWinsize()
	return int(ws.Col)
}

func Height() int {
	ws, _ := getWinsize()
	return int(ws.Col)
}
