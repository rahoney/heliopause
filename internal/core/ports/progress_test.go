package ports

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestDownloadProgressPreservesBoundedBytesAndError(t *testing.T) {
	var events []Progress
	ctx := WithProgress(context.Background(), func(p Progress) { events = append(events, p) })
	body, err := io.ReadAll(DownloadReader(ctx, io.LimitReader(strings.NewReader("abcdef"), 3), 6))
	if err != nil || string(body) != "abc" || len(events) == 0 || events[len(events)-1].Current != 3 || events[len(events)-1].Total != 6 {
		t.Fatalf("bounded download changed: %q %v %#v", body, err, events)
	}
	expected := errors.New("original read failure")
	_, err = io.ReadAll(DownloadReader(ctx, progressFaultReader{expected}, -1))
	if !errors.Is(err, expected) {
		t.Fatalf("read error changed: %v", err)
	}
}

type progressFaultReader struct{ err error }

func (r progressFaultReader) Read([]byte) (int, error) { return 0, r.err }
