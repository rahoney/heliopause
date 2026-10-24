package promotion

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"os"
	"slices"
	"syscall"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

// PNPMOriginalVerifier is supplied by trusted composition. It proves every
// original file against independently acquired sources, never current Policy.
type PNPMOriginalVerifier func(context.Context, domain.InstallContext, []domain.ProjectControlFile, artifactnpm.PNPMOriginalLayout) (domain.ContentDigest, error)

func NewPNPMProjectPromotionWithOriginalVerifier(intake, evidence, cache, state string, reader ProjectEvidenceReader, operation domain.OperationID, selectedCache func(domain.ProjectDependencySnapshot) (*PNPMVerifiedCache, error), verify PNPMOriginalVerifier) (*PNPMProjectPromotion, error) {
	if verify == nil {
		return nil, errors.New("pnpm original source verifier is required")
	}
	p, err := NewPNPMProjectPromotion(intake, evidence, cache, state, reader, operation, selectedCache)
	if err != nil {
		return nil, err
	}
	p.verifyOriginal = verify
	return p, nil
}

type pnpmOriginalBuffer struct{ bytes.Buffer }

func (b *pnpmOriginalBuffer) Write(body []byte) (int, error) {
	if len(body) > artifactnpm.PNPMLayoutArchiveLimit-b.Len() {
		return 0, errors.New("pnpm original transport exceeds bound")
	}
	return b.Buffer.Write(body)
}

func (g *approvedPNPMProjectGuard) proveOriginal(ctx context.Context) error {
	if g.owner.verifyOriginal == nil || g.members["node_modules"] == nil || g.members["haa"] != nil || g.previous != nil || len(g.originalState) != 0 {
		return errors.New("pnpm original native adoption requires its independent source verifier")
	}
	if err := g.VerifyUnchanged(ctx); err != nil {
		return err
	}
	original, err := g.originalLayout(ctx)
	if err != nil {
		return err
	}
	proof, err := g.owner.verifyOriginal(ctx, g.install, append([]domain.ProjectControlFile(nil), g.controls...), original)
	if err != nil {
		return errors.Join(err, ctx.Err(), context.Cause(ctx))
	}
	if proof.String() == "" {
		return errors.New("pnpm original source proof is missing")
	}
	if err := g.VerifyUnchanged(ctx); err != nil {
		return err
	}
	g.originalProof = proof.String()
	return nil
}

func (g *approvedPNPMProjectGuard) originalLayout(ctx context.Context) (result artifactnpm.PNPMOriginalLayout, resultErr error) {
	m := g.members["node_modules"].pnpm
	if m == nil {
		return result, errors.New("pnpm original census is unavailable")
	}
	owner, ok := m.entries["."].info.Sys().(*syscall.Stat_t)
	if !ok {
		return result, errors.New("pnpm original owner is unavailable")
	}
	root, err := os.OpenRoot(g.memberPath("node_modules"))
	if err != nil {
		return result, err
	}
	defer func() { resultErr = errors.Join(resultErr, root.Close()) }()
	var names []string
	for name := range m.entries {
		names = append(names, name)
	}
	slices.Sort(names)
	var buffer pnpmOriginalBuffer
	writer := tar.NewWriter(&buffer)
	for _, name := range names {
		if err := g.verifyPinned(); err != nil {
			return result, err
		}
		if err := verifyPNPMOwnedEntry(ctx, root, m.entries, name); err != nil {
			return result, err
		}
		entry := m.entries[name]
		stat, ok := entry.info.Sys().(*syscall.Stat_t)
		if !ok || stat.Uid != owner.Uid || stat.Gid != owner.Gid {
			return result, errors.New("pnpm original has mixed ownership")
		}
		header := &tar.Header{Name: "node_modules", Mode: int64(entry.info.Mode().Perm()), Uid: int(stat.Uid), Gid: int(stat.Gid), Typeflag: tar.TypeDir}
		if name != "." {
			header.Name += "/" + name
		}
		var body []byte
		if entry.info.Mode().IsRegular() {
			body, err = readPNPMHostRegular(ctx, root, name, entry.info, artifactnpm.ArchiveFileLimit)
			if err != nil || pnpmCacheHash(body) != entry.digest {
				return result, errors.Join(errors.New("pnpm original changed during transport"), err)
			}
			header.Typeflag, header.Size = tar.TypeReg, int64(len(body))
		} else if entry.info.Mode()&os.ModeSymlink != 0 {
			header.Typeflag, header.Linkname = tar.TypeSymlink, entry.link
		}
		if err := writer.WriteHeader(header); err != nil {
			return result, err
		}
		if _, err := writer.Write(body); err != nil {
			return result, err
		}
	}
	if err := writer.Close(); err != nil {
		return result, err
	}
	if err := g.VerifyUnchanged(ctx); err != nil {
		return result, err
	}
	return artifactnpm.ParsePNPMOriginalLayout(ctx, buffer.Bytes(), int(owner.Uid), int(owner.Gid))
}
