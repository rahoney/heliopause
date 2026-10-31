package sandbox

import (
	"archive/tar"
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

func TestBunInstalledSupplyUsesSeparateImmutableIdentity(t *testing.T) {
	base := pnpmSupplyFileInfo{mode: 0555, size: runtimeidentity.BunToolSize}
	base.owner.Nlink = 1
	if !nativePinnedFile(base, runtimeidentity.BunToolSize) || nativePinnedFile(base, runtimeidentity.PNPMToolSize) {
		t.Fatal("Bun size was rejected or became PNPM identity")
	}
	for _, kind := range []nativeRuntimeToolKind{nativeRuntimeUV, nativeRuntimePNPM, nativeRuntimeBun} {
		spec, known := nativeRuntimeToolIdentity(kind)
		if !known || spec.size <= 0 || spec.size > spec.maxBytes {
			t.Fatal("unusable compiled tool bound", kind)
		}
		want := int64(64 << 20)
		if kind == nativeRuntimeBun {
			want = 128 << 20
		}
		if spec.maxBytes != want {
			t.Fatal("another tool inherited Bun limit", kind)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if got, err := readInstalledBunTool(ctx, filepath.Join(t.TempDir(), "absent")); got != nil || !errors.Is(err, context.Canceled) {
		t.Fatal("Bun fixed reader lost cancellation", err)
	}
	if got, err := prepareBunTool(ctx); got != nil || !errors.Is(err, context.Canceled) {
		t.Fatal("fixed Bun wrapper lost cancellation", err)
	}
	for _, kind := range []nativeRuntimeToolKind{nativeRuntimeUV, nativeRuntimeToolKind(99)} {
		if body, err := readInstalledNativeTool(context.Background(), "unavailable", kind); body != nil || err == nil {
			t.Fatal("non-installed kind acquired a source")
		}
	}
	runner := &uvSupplyFake{}
	for _, tool := range []*bunPreparedTool{nil, {}, {body: []byte("substituted")}} {
		if err := tool.introduce(context.Background(), runner, strings.Repeat("b", 64)); err == nil {
			t.Fatal("unprepared Bun bytes introduced")
		}
	}
	if len(runner.calls) != 0 {
		t.Fatal("unverified Bun reached Docker")
	}
}

func TestBunTransportPreservesRootTarBoundaryAndKind(t *testing.T) {
	body, id := uvSyntheticTar(t, func(h *tar.Header) { h.Name = "bun" }, false)
	if _, err := scanNativeToolTar(body, id, nativeRuntimeBun); err != nil {
		t.Fatal("structural Bun transport refused", err)
	}
	for _, kind := range []nativeRuntimeToolKind{nativeRuntimeUV, nativeRuntimePNPM, nativeRuntimeToolKind(99)} {
		if _, err := scanNativeToolTar(body, id, kind); err == nil {
			t.Fatal("Bun tar acquired another tool role")
		}
	}
	for _, fault := range []string{"path", "owner", "group", "mode", "link", "duplicate", "hash", "trailing", "truncated", "oversize"} {
		t.Run(fault, func(t *testing.T) {
			body, id := uvSyntheticTar(t, func(h *tar.Header) {
				h.Name = "bun"
				switch fault {
				case "path":
					h.Name = "../bun"
				case "owner":
					h.Uid = 1000
				case "group":
					h.Gid = 1000
				case "mode":
					h.Mode = 0777
				case "link":
					h.Typeflag = tar.TypeSymlink
					h.Linkname = "/elsewhere"
					h.Size = 0
				}
			}, fault == "duplicate")
			switch fault {
			case "hash":
				id.sha256 = strings.Repeat("0", 64)
			case "trailing":
				body = append(body, 1)
			case "truncated":
				body = body[:520]
			case "oversize":
				id.size = 128<<20 + 1
			}
			if _, err := scanNativeToolTar(body, id, nativeRuntimeBun); err == nil {
				t.Fatal("unbound Bun transport admitted")
			}
		})
	}
	f, v, _ := nativeVolumeFixture(t, nativeRuntimeBun)
	v.kind = nativeRuntimePNPM
	if err := v.verify(context.Background(), f); err == nil {
		t.Fatal("Bun volume became PNPM authority")
	}
}
