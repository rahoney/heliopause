package ports

import (
	"context"
	"errors"
	"testing"
)

func TestBootstrapFailureKeepsCauseWithoutRenderingUntrustedDetails(t *testing.T) {
	cause := errors.New("untrusted transport detail")
	err := NewBootstrapPreparationFailure(BootstrapProvenanceFailed, cause)
	if err.Error() != string(BootstrapProvenanceFailed) || !errors.Is(err, cause) {
		t.Fatal("bootstrap failure exposed or lost cause")
	}
	err = NewBootstrapPreparationFailure(BootstrapExportFailed, context.Canceled)
	var failure *BootstrapPreparationFailure
	if !errors.As(err, &failure) || failure.Code() != BootstrapCancelled || !errors.Is(err, context.Canceled) {
		t.Fatal("cancellation cause lost")
	}
}
