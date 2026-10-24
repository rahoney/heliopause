package sandbox

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

type pnpmSupplyFileInfo struct {
	mode         os.FileMode
	size         int64
	owner        syscall.Stat_t
	missingOwner bool
}

func (s pnpmSupplyFileInfo) Name() string       { return runtimeidentity.PNPMNativeAssetName }
func (s pnpmSupplyFileInfo) Size() int64        { return s.size }
func (s pnpmSupplyFileInfo) Mode() os.FileMode  { return s.mode }
func (s pnpmSupplyFileInfo) ModTime() time.Time { return time.Time{} }
func (s pnpmSupplyFileInfo) IsDir() bool        { return s.mode.IsDir() }
func (s pnpmSupplyFileInfo) Sys() any {
	if s.missingOwner {
		return nil
	}
	return &s.owner
}

func TestPNPMNativeSourceRequiresExactProtectedFileMetadata(t *testing.T) {
	base := pnpmSupplyFileInfo{mode: 0555, size: runtimeidentity.PNPMToolSize, owner: syscall.Stat_t{Nlink: 1}}
	if !pnpmPinnedFile(base) {
		t.Fatal("valid structural Root file identity refused")
	}
	for _, mode := range []string{"uid", "gid", "writable", "special", "directory", "symlink", "hardlink", "size", "no-owner"} {
		t.Run(mode, func(t *testing.T) {
			info := base
			switch mode {
			case "uid":
				info.owner.Uid = 1000
			case "gid":
				info.owner.Gid = 1000
			case "writable":
				info.mode = 0755
			case "special":
				info.mode |= os.ModeSetuid
			case "directory":
				info.mode |= os.ModeDir
			case "symlink":
				info.mode |= os.ModeSymlink
			case "hardlink":
				info.owner.Nlink = 2
			case "size":
				info.size--
			case "no-owner":
				info.missingOwner = true
			}
			if pnpmPinnedFile(info) {
				t.Fatal("foreign or mutable source file became a pinned tool")
			}
		})
	}
	if pnpmPinnedFile(nil) || pnpmRootCurrent(nil) || pnpmRootDirectory(nil) {
		t.Fatal("unknown metadata admitted")
	}
	directory := base
	directory.mode = os.ModeDir | 0755
	if !pnpmRootDirectory(directory) {
		t.Fatal("owned protected directory refused")
	}
	directory.mode = os.ModeDir | 0775
	if pnpmRootDirectory(directory) {
		t.Fatal("shared writable directory admitted")
	}
	link := base
	link.mode = os.ModeSymlink | 0777
	if !pnpmRootCurrent(link) {
		t.Fatal("owned current pointer refused")
	}
	link.owner.Uid = 1000
	if pnpmRootCurrent(link) {
		t.Fatal("ordinary current pointer admitted")
	}
}

func TestPNPMNativeCurrentPointerCannotEscapeVersions(t *testing.T) {
	for _, valid := range []string{"versions/v1.0.0", "versions/v1.0.0-rc.1+build"} {
		if !pnpmCurrentTarget(valid) {
			t.Fatal("canonical bounded target refused", valid)
		}
	}
	for _, invalid := range []string{"", "/versions/v1.0.0", "../versions/v1.0.0", "versions/../v1.0.0", "versions/v1.0.0/other", "versions/v1.0.0/..", "versions/v", "versions/v1.0.0\n", "versions/v1.0.0\\other", "versions/v" + strings.Repeat("x", 128)} {
		if pnpmCurrentTarget(invalid) {
			t.Fatal("unsafe current target admitted", invalid)
		}
	}
}

func TestPNPMNativeSourceCancellationPrecedesFileAccess(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if prepared, err := readInstalledPNPMTool(ctx, filepath.Join(t.TempDir(), "absent")); prepared != nil || !errors.Is(err, context.Canceled) {
		t.Fatal("source cancellation was lost or produced bytes", err)
	}
	var missingContext context.Context
	if prepared, err := readInstalledPNPMTool(missingContext, "unavailable"); prepared != nil || err == nil {
		t.Fatal("nil source context accepted")
	}
	if prepared, err := preparePNPMTool(ctx); prepared != nil || !errors.Is(err, context.Canceled) {
		t.Fatal("fixed source wrapper ignored cancellation", err)
	}
}

func TestPNPMNativeIntroductionRejectsUnpreparedBytes(t *testing.T) {
	runner := &uvSupplyFake{}
	for _, tool := range []*pnpmPreparedTool{nil, {}, {body: []byte("substituted")}} {
		if err := tool.introduce(context.Background(), runner, strings.Repeat("b", 64)); err == nil {
			t.Fatal("unprepared tool introduced")
		}
	}
	if len(runner.calls) != 0 {
		t.Fatal("invalid native bytes reached Docker")
	}
}
