//go:build !windows

package term

import "os"

func enableVirtual(*os.File) {}
