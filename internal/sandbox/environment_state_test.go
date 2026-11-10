package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

type environmentTreeTransport struct {
	body      []byte
	failure   error
	arguments []string
}

func (*environmentTreeTransport) Output(context.Context, string, ...string) ([]byte, error) {
	return nil, errors.New("unexpected DATA command")
}

func (r *environmentTreeTransport) RunOutput(_ context.Context, output io.Writer, _ string, args ...string) error {
	r.arguments = append([]string(nil), args...)
	_, err := output.Write(r.body)
	return errors.Join(err, r.failure)
}

func TestEnvironmentStateTransportRequiresCompleteProducer(t *testing.T) {
	const container = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	limits := environmentStateLimits{entries: 16, bytes: 64, archiveBytes: 16 << 10}
	body := environmentTarFixture(t, []tar.Header{{Name: "data", Size: 4}}, "data")
	for _, failed := range []bool{false, true} {
		runner := &environmentTreeTransport{body: body}
		if failed {
			runner.failure = errors.New("producer did not complete")
		}
		state, err := captureEnvironmentState(context.Background(), runner, container, limits)
		if failed {
			if !errors.Is(err, runner.failure) || state.digest != "" || state.files != nil {
				t.Fatal("valid prefix concealed failed producer", err)
			}
		} else if err != nil || state.digest == "" {
			t.Fatal("complete producer rejected", err)
		}
		if !sameStrings(runner.arguments, []string{"cp", container + ":/tmp/.", "-"}) {
			t.Fatal("DATA transport used another root", runner.arguments)
		}
	}
	runner := &environmentTreeTransport{body: body}
	if err := readContainerTree(context.Background(), runner, container, "/proc", func(io.Reader) error { return nil }); err == nil || runner.arguments != nil {
		t.Fatal("non-workspace root reached DATA transport")
	}
}

func TestEnvironmentStateChangesIncludeContentModeKindAndOwnership(t *testing.T) {
	before := scanEnvironmentFixture(t, []tar.Header{
		{Name: "state", Mode: 0600, Size: 4},
		{Name: "gone", Mode: 0600},
		{Name: "mode", Mode: 0600},
		{Name: "owner", Mode: 0600},
		{Name: "kind", Mode: 0700, Typeflag: tar.TypeDir},
	}, "same")
	after := scanEnvironmentFixture(t, []tar.Header{
		{Name: "state", Mode: 0600, Size: 4},
		{Name: "new", Mode: 0600},
		{Name: "mode", Mode: 0640},
		{Name: "owner", Mode: 0600, Uid: 42},
		{Name: "kind", Mode: 0700},
	}, "diff")
	changes, err := compareEnvironmentStates(before, after)
	if err != nil || changes != (environmentChanges{created: 1, deleted: 1, modified: 4}) || before.digest == after.digest {
		t.Fatalf("changes=%+v err=%v", changes, err)
	}
	reordered := scanEnvironmentFixture(t, []tar.Header{
		{Name: "kind", Mode: 0700},
		{Name: "owner", Mode: 0600, Uid: 42},
		{Name: "mode", Mode: 0640},
		{Name: "new", Mode: 0600},
		{Name: "state", Mode: 0600, Size: 4},
	}, "diff")
	if reordered.digest != after.digest {
		t.Fatal("identity depends on TAR order")
	}
	if _, err := compareEnvironmentStates(environmentState{}, after); err == nil {
		t.Fatal("missing baseline admitted")
	}
}

func TestEnvironmentStateRejectsUnboundedIncompleteAndEscapingTrees(t *testing.T) {
	limits := environmentStateLimits{entries: 16, bytes: 64, archiveBytes: 16 << 10}
	for _, test := range []struct {
		name    string
		headers []tar.Header
		limits  environmentStateLimits
	}{
		{"parent escape", []tar.Header{{Name: "../outside"}}, limits},
		{"absolute", []tar.Header{{Name: "/outside"}}, limits},
		{"duplicate", []tar.Header{{Name: "same"}, {Name: "same"}}, limits},
		{"absolute link", []tar.Header{{Name: "link", Typeflag: tar.TypeSymlink, Linkname: "/proc/self/environ"}}, limits},
		{"link escape", []tar.Header{{Name: "link", Typeflag: tar.TypeSymlink, Linkname: "../secret"}}, limits},
		{"link ancestor", []tar.Header{{Name: "a/b"}, {Name: "a", Typeflag: tar.TypeSymlink, Linkname: "other"}}, limits},
		{"file ancestor", []tar.Header{{Name: "a/b"}, {Name: "a"}}, limits},
		{"hard link", []tar.Header{{Name: "link", Typeflag: tar.TypeLink, Linkname: "other"}}, limits},
		{"hardlink forward", []tar.Header{{Name: "link", Typeflag: tar.TypeLink, Linkname: "file"}, {Name: "file"}}, limits},
		{"hardlink cycle", []tar.Header{{Name: "first", Typeflag: tar.TypeLink, Linkname: "second"}, {Name: "second", Typeflag: tar.TypeLink, Linkname: "first"}}, limits},
		{"hardlink absolute", []tar.Header{{Name: "file"}, {Name: "link", Typeflag: tar.TypeLink, Linkname: "/file"}}, limits},
		{"hardlink escape", []tar.Header{{Name: "file"}, {Name: "link", Typeflag: tar.TypeLink, Linkname: "../file"}}, limits},
		{"hardlink noncanonical", []tar.Header{{Name: "file"}, {Name: "link", Typeflag: tar.TypeLink, Linkname: "a/../file"}}, limits},
		{"hardlink directory", []tar.Header{{Name: "dir", Typeflag: tar.TypeDir}, {Name: "link", Typeflag: tar.TypeLink, Linkname: "dir"}}, limits},
		{"hardlink symlink", []tar.Header{{Name: "file", Typeflag: tar.TypeSymlink, Linkname: "other"}, {Name: "link", Typeflag: tar.TypeLink, Linkname: "file"}}, limits},
		{"hardlink mode", []tar.Header{{Name: "file", Mode: 0600}, {Name: "link", Mode: 0644, Typeflag: tar.TypeLink, Linkname: "file"}}, limits},
		{"hardlink owner", []tar.Header{{Name: "file", Uid: 42}, {Name: "link", Typeflag: tar.TypeLink, Linkname: "file"}}, limits},
		{"hardlink group", []tar.Header{{Name: "file", Gid: 42}, {Name: "link", Typeflag: tar.TypeLink, Linkname: "file"}}, limits},
		{"device", []tar.Header{{Name: "device", Typeflag: tar.TypeChar}}, limits},
		{"FIFO", []tar.Header{{Name: "pipe", Typeflag: tar.TypeFifo}}, limits},
		{"entries", []tar.Header{{Name: "a"}}, environmentStateLimits{entries: 1, bytes: 64, archiveBytes: 16 << 10}},
		{"bytes", []tar.Header{{Name: "a", Size: 65}}, limits},
		{"archive bytes", []tar.Header{{Name: "a"}}, environmentStateLimits{entries: 16, bytes: 64, archiveBytes: 512}},
	} {
		t.Run(test.name, func(t *testing.T) {
			body := environmentTarFixture(t, test.headers, strings.Repeat("x", 65))
			state, err := scanEnvironmentState(context.Background(), bytes.NewReader(body), test.limits)
			if !errors.Is(err, errEnvironmentState) || state.digest != "" || state.files != nil {
				t.Fatalf("incomplete tree published: %+v, %v", state, err)
			}
			if strings.Contains(err.Error(), "outside") || strings.Contains(err.Error(), "secret") {
				t.Fatal("raw path leaked through error")
			}
		})
	}
	for _, body := range [][]byte{nil, []byte("broken"), environmentTarFixture(t, []tar.Header{{Name: "data", Size: 4}}, "data")[:1025]} {
		if _, err := scanEnvironmentState(context.Background(), bytes.NewReader(body), limits); err == nil {
			t.Fatal("empty, broken or truncated snapshot admitted")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := scanEnvironmentState(ctx, bytes.NewReader(nil), limits); err == nil {
		t.Fatal("canceled snapshot admitted")
	}
}

func TestEnvironmentStateInventoriesLinksWithoutFollowingThem(t *testing.T) {
	state := scanEnvironmentFixture(t, []tar.Header{
		{Name: "bin", Mode: 0700, Typeflag: tar.TypeDir},
		{Name: "bin/tool", Mode: 0777, Typeflag: tar.TypeSymlink, Linkname: "../tool"},
		{Name: "tool", Mode: 0600, Size: 4},
	}, "data")
	if state.bytes != 4 || len(state.files) != 4 || state.files["bin/tool"].kind != tar.TypeSymlink {
		t.Fatal("link target was followed or omitted")
	}
}

func TestEnvironmentStateInventoriesRegularHardlinkContent(t *testing.T) {
	// Docker DATA may emit the same inode as one regular file and subsequent
	// hardlinks. Its choice of representative must not change the inventory.
	first := scanEnvironmentFixture(t, []tar.Header{
		{Name: "COPY", Mode: 0644, Size: 4},
		{Name: "LICENSE", Mode: 0644, Typeflag: tar.TypeLink, Linkname: "./COPY"},
	}, "data")
	second := scanEnvironmentFixture(t, []tar.Header{
		{Name: "LICENSE", Mode: 0644, Size: 4},
		{Name: "COPY", Mode: 0644, Typeflag: tar.TypeLink, Linkname: "LICENSE"},
	}, "data")
	if first.digest != second.digest || first.bytes != 4 || first.files["COPY"] != first.files["LICENSE"] {
		t.Fatal("regular hardlink content was omitted or depends on TAR representation")
	}
	chained := scanEnvironmentFixture(t, []tar.Header{
		{Name: "COPY", Mode: 0644, Size: 4},
		{Name: "LICENSE", Mode: 0644, Typeflag: tar.TypeLink, Linkname: "COPY"},
		{Name: "other", Mode: 0644, Typeflag: tar.TypeLink, Linkname: "LICENSE"},
	}, "data")
	if chained.files["other"] != first.files["COPY"] || chained.bytes != 4 {
		t.Fatal("already resolved alias was not inventoried once")
	}
	changed := scanEnvironmentFixture(t, []tar.Header{
		{Name: "COPY", Mode: 0644, Size: 4},
		{Name: "LICENSE", Mode: 0644, Typeflag: tar.TypeLink, Linkname: "COPY"},
	}, "edit")
	changes, err := compareEnvironmentStates(first, changed)
	if err != nil || changes != (environmentChanges{modified: 2}) {
		t.Fatalf("hardlinked content change lost: %+v, %v", changes, err)
	}
}

func scanEnvironmentFixture(t *testing.T, headers []tar.Header, content string) environmentState {
	t.Helper()
	state, err := scanEnvironmentState(context.Background(), bytes.NewReader(environmentTarFixture(t, headers, content)), environmentStateLimits{entries: 32, bytes: 1024, archiveBytes: 32 << 10})
	if err != nil {
		t.Fatal(err)
	}
	return state
}

func environmentTarFixture(t *testing.T, headers []tar.Header, content string) []byte {
	t.Helper()
	var body bytes.Buffer
	archive := tar.NewWriter(&body)
	root := tar.Header{Name: "./", Typeflag: tar.TypeDir, Mode: 0700, Uid: 1000, Gid: 1000}
	for _, header := range append([]tar.Header{root}, headers...) {
		if err := archive.WriteHeader(&header); err != nil {
			t.Fatal(err)
		}
		if header.Size > 0 {
			if int64(len(content)) < header.Size {
				t.Fatal("fixture content missing")
			}
			if _, err := io.WriteString(archive, content[:header.Size]); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	return body.Bytes()
}
