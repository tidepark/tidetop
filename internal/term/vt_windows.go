//go:build windows

package term

import (
	"os"

	"golang.org/x/sys/windows"
)

// 윈도우 콘솔에서 색 부호(가상 터미널)를 켠다. 옛 cmd.exe 에서도 색이 깨지지 않게 한다.
func enableVirtual(out *os.File) {
	h := windows.Handle(out.Fd())
	var mode uint32
	if windows.GetConsoleMode(h, &mode) == nil {
		windows.SetConsoleMode(h, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
	}
}
