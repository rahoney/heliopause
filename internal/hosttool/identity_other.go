//go:build !unix

package hosttool

import (
	"errors"
	"os"
)

func verifyTrustedOwner(os.FileInfo) error {
	return errors.New("Host tool ownership verification is unsupported")
}
func verifyEndpointOwner(os.FileInfo) error {
	return errors.New("Docker endpoint ownership verification is unsupported")
}
