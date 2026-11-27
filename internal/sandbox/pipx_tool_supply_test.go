package sandbox

import (
	"archive/tar"
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

func TestPipxSupplyRequiresFixedDataIdentity(t *testing.T) {
	spec, ok := nativeRuntimeToolIdentity(nativeRuntimePipx)
	if !ok || spec.fileMode() != 0444 || spec.maxBytes != 8<<20 || spec.installedName != runtimeidentity.PipxRuntimeAssetName || spec.guestPath != "/haa-pipx/pipx-runtime.zip" {
		t.Fatal("incorrect fixed DATA descriptor")
	}
	for _, kind := range []nativeRuntimeToolKind{nativeRuntimeUV, nativeRuntimePNPM, nativeRuntimeBun} {
		s, ok := nativeRuntimeToolIdentity(kind)
		if !ok || s.fileMode() != 0555 {
			t.Fatal("existing executable mode changed")
		}
	}
	base := pnpmSupplyFileInfo{mode: 0444, size: runtimeidentity.PipxToolSize}
	base.owner.Nlink = 1
	if !nativePinnedFileMode(base, spec.size, 0444) || nativePinnedFile(base, spec.size) {
		t.Fatal("DATA confused with executable")
	}
	for _, fault := range []string{"executable", "writable", "owner", "group", "link", "symlink", "directory", "setuid", "size", "unknown"} {
		t.Run(fault, func(t *testing.T) {
			f := base
			switch fault {
			case "executable":
				f.mode = 0555
			case "writable":
				f.mode = 0644
			case "owner":
				f.owner.Uid = 1000
			case "group":
				f.owner.Gid = 1000
			case "link":
				f.owner.Nlink = 2
			case "symlink":
				f.mode |= os.ModeSymlink
			case "directory":
				f.mode |= os.ModeDir
			case "setuid":
				f.mode |= os.ModeSetuid
			case "size":
				f.size--
			case "unknown":
				f.missingOwner = true
			}
			if nativePinnedFileMode(f, spec.size, 0444) {
				t.Fatal("untrusted DATA admitted")
			}
		})
	}
	if nativePinnedFileMode(base, spec.size, 0644) || nativePinnedFileMode(nil, spec.size, 0444) {
		t.Fatal("invalid mode or unknown file admitted")
	}
	f := &uvVolumeFake{}
	if _, err := createNativeRuntimeVolume(context.Background(), f, nativeRuntimePipx); err == nil || f.volume.Name != "" {
		t.Fatal("ZIP acquired single-writer native volume")
	}
	v := nativeRuntimeVolume{kind: nativeRuntimePipx, name: "x", owner: "x", createdAt: "fixed", mountpoint: "/owned"}
	if _, err := v.mountArgument(); err == nil {
		t.Fatal("forged DATA mount admitted")
	}
}

func TestPipxSupplyRejectsCallerIdentityAndUnpreparedBytes(t *testing.T) {
	body, id := uvSyntheticTar(t, func(h *tar.Header) { h.Name = "pipx-runtime.zip"; h.Mode = 0444 }, false)
	id.mode = 0444
	if _, err := scanNativeToolTar(body, id, nativeRuntimePipx); err == nil {
		t.Fatal("caller hash became selected pipx supply")
	}
	pin := nativeToolFileIdentity{runtimeidentity.PipxToolSize, runtimeidentity.PipxToolSHA256, 0444}
	for _, fault := range []string{"size", "hash", "mode"} {
		t.Run(fault, func(t *testing.T) {
			bad := pin
			switch fault {
			case "size":
				bad.size--
			case "hash":
				bad.sha256 = strings.Repeat("0", 64)
			case "mode":
				bad.mode = 0555
			}
			f := &uvSupplyFake{}
			if _, err := readNativeToolFromContainer(context.Background(), f, strings.Repeat("b", 64), pipxToolGuestPath, bad, nativeRuntimePipx); err == nil || len(f.calls) != 0 {
				t.Fatal("unbound identity reached Docker")
			}
		})
	}
	for _, p := range []*pipxPreparedTool{nil, {}, {body: []byte("changed")}} {
		f := &uvSupplyFake{}
		if err := p.introduce(context.Background(), f, strings.Repeat("b", 64)); err == nil || len(f.calls) != 0 {
			t.Fatal("unprepared bytes reached Docker")
		}
	}
	f := &uvSupplyFake{}
	if _, err := readNativeToolFromContainer(context.Background(), f, strings.Repeat("b", 64), "/haa-runtime/uv", pin, nativeRuntimePipx); err == nil || len(f.calls) != 0 {
		t.Fatal("foreign path reached Docker")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if b, err := preparePipxTool(ctx); b != nil || !errors.Is(err, context.Canceled) {
		t.Fatal("cancellation lost", err)
	}
}
