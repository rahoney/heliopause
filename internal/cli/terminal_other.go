//go:build !linux && !darwin

package cli

import "os"

func terminalWidth(*os.File) (int, bool) { return 0, false }
