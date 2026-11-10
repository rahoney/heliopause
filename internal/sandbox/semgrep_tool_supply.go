package sandbox

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"github.com/rahoney/heliopause/internal/runtimeidentity"
)

const semgrepRuntimeRoot = "/haa-semgrep"

// The fixed transport maxima are 256MiB core and 1MiB rules. Artifact target
// data never shares this executable volume.
const semgrepSupplyVolumeCapacity int64 = 257 << 20

type semgrepPreparedTool struct{ core, rules []byte }

func semgrepSupplyDigest() string {
	pin := runtimeidentity.SelectedSemgrepSupply()
	sum := sha256.Sum256([]byte(pin.Tool.SHA256 + ":" + pin.Rules.SHA256))
	return hex.EncodeToString(sum[:])
}

func readInstalledSemgrepTool(ctx context.Context, installation string) (*semgrepPreparedTool, error) {
	core, err := readInstalledNativeTool(ctx, installation, nativeRuntimeSemgrep)
	if err != nil {
		return nil, err
	}
	rules, err := readInstalledNativeTool(ctx, installation, nativeRuntimeSemgrepRules)
	if err != nil {
		return nil, err
	}
	return &semgrepPreparedTool{core, rules}, nil
}

func (p *semgrepPreparedTool) introduce(ctx context.Context, runner CommandRunner, container string) error {
	pin := runtimeidentity.SelectedSemgrepSupply()
	if p == nil || int64(len(p.core)) != pin.Tool.Size || int64(len(p.rules)) != pin.Rules.Size {
		return errors.New("selected semgrep supply is unavailable")
	}
	if err := introduceNativeTool(ctx, runner, container, nativeRuntimeSemgrep, p.core); err != nil {
		return err
	}
	return introduceNativeTool(ctx, runner, container, nativeRuntimeSemgrepRules, p.rules)
}

func createSemgrepSupplyVolume(ctx context.Context, runner CommandRunner, transaction string) (*readonlyToolVolume, error) {
	volume, err := createInputVolume(ctx, runner, transaction+"-semgrep-supply", semgrepSupplyDigest(), semgrepSupplyVolumeCapacity, false, projectBuildVolumeConfig{destination: semgrepRuntimeRoot, executable: true})
	if volume.name == "" {
		return nil, err
	}
	return &readonlyToolVolume{volume: volume, semgrep: true}, err
}
