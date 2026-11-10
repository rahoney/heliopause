//go:build linux

package releaseactivation

import (
	"context"
	"errors"
	"testing"
)

func TestActivationAndRecoveryRejectIncompleteStateBeforeHostIO(t *testing.T) {
	ctx := context.Background()
	if _, err := prepareRootActivation(ctx, nil, nil); err == nil {
		t.Fatal("missing protected stage admitted")
	}
	if err := (*rootActivation)(nil).activate(ctx); !errors.Is(err, ErrTransition) {
		t.Fatal("missing transaction admitted")
	}
	if err := restorableRuntimeConfigs(ctx, updateJournal{}); err == nil {
		t.Fatal("unissued snapshot admitted")
	}
	if _, err := recoverPreparedStage(ctx, updateJournal{}, InstalledRelease{}, nil); err == nil {
		t.Fatal("invalid journal selected a filesystem stage")
	}
}
