//go:build !linux && !darwin

package cli

import (
	"errors"
	"os"
)

func launcherGeometry(*os.File, *os.File) (int, int, bool) { return 0, 0, false }

func openLauncherTerminal(*os.File, *os.File) (launcherTerminal, error) {
	return nil, errors.New("interactive launcher requires a supported terminal")
}
