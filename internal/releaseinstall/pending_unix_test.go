//go:build linux || darwin

package releaseinstall

import (
	"bytes"
	"context"
	"errors"
	"testing"
)

func TestSystemCopyPrefixComparisonRejectsPartialMismatchAndCancellation(t *testing.T) {
	source := bytes.Repeat([]byte("verified-source"), 10000)
	for _, size := range []int{0, 1, 65536, len(source)} {
		if err := compareCopyPrefix(context.Background(), bytes.NewReader(source[:size]), bytes.NewReader(source), int64(size)); err != nil {
			t.Fatal(err)
		}
	}
	if err := compareCopyPrefix(context.Background(), bytes.NewReader([]byte("bad")), bytes.NewReader(source), 3); err == nil {
		t.Fatal("wrong prefix accepted")
	}
	if err := compareCopyPrefix(context.Background(), bytes.NewReader(source[:1]), bytes.NewReader(source), 2); err == nil {
		t.Fatal("short partial accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := compareCopyPrefix(ctx, bytes.NewReader(source), bytes.NewReader(source), int64(len(source))); !errors.Is(err, context.Canceled) {
		t.Fatal("cancellation lost", err)
	}
}
