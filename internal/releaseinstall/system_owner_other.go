//go:build !linux && !darwin

package releaseinstall

import "errors"

func verifySystemDirectory(string, bool) error {
	return errors.New("protected release installation requires Linux amd64")
}
func verifySystemRegularFile(string) error {
	return errors.New("protected release installation requires Linux amd64")
}
