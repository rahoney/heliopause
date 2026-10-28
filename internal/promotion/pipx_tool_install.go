package promotion

import (
	"context"
	"errors"
	"path/filepath"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/sandbox"
)

// These are controller-selected codecs, never artifact supplied paths or code.
// Zero retains the existing UV contract for all existing callers.
type toolClient uint8

const (
	toolClientUV toolClient = iota
	toolClientPipx
	pipxToolHostReceipt        = "pipx_metadata.json"
	pipxToolHostReceiptControl = "pipx-tool-host-receipt.json"
)

func (c toolClient) valid() bool { return c == toolClientUV || c == toolClientPipx }
func (c toolClient) name() string {
	if c == toolClientPipx {
		return "pipx"
	}
	return "uv"
}
func (c toolClient) receiptName() string {
	if c == toolClientPipx {
		return pipxToolHostReceipt
	}
	return uvToolHostReceipt
}
func (c toolClient) receiptControl() string {
	if c == toolClientPipx {
		return pipxToolHostReceiptControl
	}
	return uvToolHostReceiptControl
}
func (c toolClient) nativeReceiptControl() string {
	if c == toolClientPipx {
		return "pipx-tool-receipt.json"
	}
	return "uv-tool-receipt.toml"
}
func (c toolClient) targetControl() string {
	if c == toolClientPipx {
		return "pipx-tool-target.json"
	}
	return "uv-tool-target.json"
}
func (c toolClient) parseReceipt(body []byte) (artifactpypi.ToolReceipt, error) {
	if c == toolClientPipx {
		return artifactpypi.ParsePipxToolReceipt(body)
	}
	if c == toolClientUV {
		return artifactpypi.ParseUVToolReceipt(body)
	}
	return artifactpypi.ToolReceipt{}, errors.New("unknown tool client")
}
func (c toolClient) projectReceipt(r artifactpypi.ToolReceipt, root, bin string) (artifactpypi.ToolReceipt, error) {
	if c == toolClientPipx {
		return r.WithPipxLayout(root, bin, filepath.Join(root, "bin", "python"))
	}
	if c == toolClientUV {
		return r.WithUVLayout(root, bin)
	}
	return artifactpypi.ToolReceipt{}, errors.New("unknown tool client")
}

type pipxToolNativeSelector interface {
	ResolveToolInstall(context.Context, artifactpypi.UpgradeRequirement) (sandbox.PipxToolSelection, error)
}

// PipxToolInstallPreparation feeds unapproved pipx selection into the same held
// target session and current whole inspection/Policy/cache/publication owners.
type PipxToolInstallPreparation struct{ shared *UVToolInstallPreparation }

func NewPipxToolInstallPreparation(promoter *PyPIPromotion, resolver pipxToolNativeSelector, binRoot string, trustedRoots []string) (*PipxToolInstallPreparation, error) {
	shared, err := newToolInstallPreparation(promoter, nil, resolver, binRoot, trustedRoots)
	if err != nil {
		return nil, err
	}
	return &PipxToolInstallPreparation{shared: shared}, nil
}

func (p *PipxToolInstallPreparation) BeginInstall(ctx context.Context, reference domain.ArtifactReference, install domain.InstallContext) (ports.InstallSession, error) {
	if p == nil || p.shared == nil {
		return nil, errors.New("pipx install preparation unavailable")
	}
	return p.shared.BeginInstall(ctx, reference, install)
}

func (p *UVToolInstallPreparation) client() toolClient {
	if p.pipx != nil {
		return toolClientPipx
	}
	return toolClientUV
}

func (s *uvToolInstallSession) resolveNativeSelection(ctx context.Context) (domain.DependencyResolution, []domain.ProjectControlFile, error) {
	if s.owner.pipx != nil {
		if s.guard.original != nil {
			if s.guard.original.pipx == nil || s.owner.pipxUpgrade == nil {
				return domain.DependencyResolution{}, nil, errors.New("pipx original selector unavailable")
			}
			selected, err := s.owner.pipxUpgrade.ResolveToolUpgrade(ctx, *s.guard.original.pipx, s.originalWheels)
			return selected.Resolution(), selected.Controls(), err
		}
		selected, err := s.owner.pipx.ResolveToolInstall(ctx, s.guard.requirement)
		return selected.Resolution(), selected.Controls(), err
	}
	var selected sandbox.UVToolSelection
	var err error
	if s.guard.original != nil {
		if s.owner.upgrade == nil {
			return domain.DependencyResolution{}, nil, errors.New("UV tool original selector is unavailable")
		}
		selected, err = s.owner.upgrade.ResolveToolUpgrade(ctx, s.guard.original.input, s.originalWheels)
	} else {
		selected, err = s.owner.resolver.ResolveToolInstall(ctx, s.guard.requirement)
	}
	return selected.Resolution(), selected.Controls(), err
}

// Pipx app_paths refer to internal venv scripts. The controller independently
// derives external link destinations from the held bin root; raw receipt claims
// stay unchanged. UV receipt paths already name the external destinations.
func (c toolClient) exposedCommands(receipt artifactpypi.ToolReceipt, binRoot string) []artifactpypi.ToolCommand {
	commands := receipt.Commands()
	if c == toolClientPipx {
		for i := range commands {
			commands[i].Path = filepath.Join(binRoot, commands[i].Name)
		}
	}
	return commands
}
