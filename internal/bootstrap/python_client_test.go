package bootstrap

import (
	"testing"

	"github.com/rahoney/heliopause/internal/cli"
)

func TestPythonPackageClientCompositionDoesNotFallback(t *testing.T) {
	legacy, err := pythonPackageClientFor("")
	if err != nil || legacy.id != cli.PipPackageClient || !legacy.derivation || legacy.resolver == nil || legacy.bind == nil || legacy.checkPlatform("darwin", "arm64") != nil {
		t.Fatal("legacy composition changed", err)
	}
	uv, err := pythonPackageClientFor(cli.UVPipPackageClient)
	if err != nil || uv.id != cli.UVPipPackageClient || uv.derivation || uv.resolver == nil || uv.bind == nil || uv.checkPlatform("linux", "amd64") != nil {
		t.Fatal("UV composition missing", err)
	}
	for _, target := range [][2]string{{"darwin", "amd64"}, {"linux", "arm64"}, {"windows", "amd64"}} {
		if err := uv.checkPlatform(target[0], target[1]); err == nil {
			t.Fatal("UV target accepted", target)
		}
		if _, err := uv.resolver(target[0], target[1], nil, nil); err == nil {
			t.Fatal("UV resolver fell back on unsupported platform", target)
		}
	}
	if _, err := pythonPackageClientFor("unregistered"); err == nil {
		t.Fatal("unregistered factory accepted")
	}
	if err := bindUVPipPackageClient(nil, pythonClientBindings{}); err == nil {
		t.Fatal("unprepared UV transaction accepted")
	}
}
