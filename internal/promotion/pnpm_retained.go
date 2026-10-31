package promotion

import (
	"context"
	"errors"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// Saved approval is used for ownership only. A new selection still requires
// fresh inspection/Policy and the current cache's full frozen-context checks.
func verifyPNPMRecordedApproval(ctx context.Context, reader ProjectEvidenceReader, doc pnpmCacheDocument) error {
	if ctx == nil || reader == nil || doc.Schema != 2 || len(doc.Controls) != 2 || (len(doc.Entries) == 0) != doc.DependencyFree || len(doc.Entries) > 4096 || doc.Policy == "" || doc.PolicyVersion == 0 || doc.Runtime == "" {
		return errors.New("pnpm retained approval coverage is invalid")
	}
	if _, err := domain.ParseOperationID(doc.Operation); err != nil {
		return err
	}
	for _, value := range []string{doc.Graph, doc.Project, doc.Manifest} {
		if _, err := domain.NewSHA256Digest(value); err != nil {
			return err
		}
	}
	if !doc.DependencyFree {
		if _, err := domain.NewSHA256Digest(doc.Layout); err != nil {
			return err
		}
	} else if doc.Layout != "" {
		return errors.New("empty pnpm retained approval invented a native layout")
	}
	source, _ := domain.NewSourceID("npm")
	seen := map[domain.ResolvedArtifactIdentity]bool{}
	runs := map[domain.RunID]bool{}
	for _, entry := range doc.Entries {
		identity, err := domain.NewResolvedArtifactIdentity(source, entry.Package, entry.Version, "tarball")
		if err != nil || entry.Source != "npm" || seen[identity] || entry.Policy == "" || entry.PolicyVersion == 0 || len(entry.Evidence) == 0 || len(entry.Evidence) > 4096 {
			return errors.New("pnpm retained entry binding is invalid")
		}
		seen[identity] = true
		run, err := domain.ParseRunID(entry.Run)
		if err != nil || runs[run] {
			return errors.New("pnpm retained Run binding is invalid")
		}
		runs[run] = true
		digest, err := domain.NewSHA256Digest(entry.Digest)
		if err != nil {
			return err
		}
		ids := map[domain.EvidenceID]bool{}
		for _, record := range entry.Evidence {
			id, err := domain.NewEvidenceID(record.ID)
			if err != nil || ids[id] {
				return errors.New("pnpm retained Evidence binding is invalid")
			}
			ids[id] = true
			ref, err := domain.NewEvidenceReference(id, "evidence:"+run.String()+":"+id.String())
			if err != nil {
				return err
			}
			_, actual, err := reader.ReadReference(ctx, run, ref, identity, digest)
			if err != nil || actual.String() != record.SHA256 {
				return errors.Join(errors.New("pnpm retained recorded Evidence changed"), err)
			}
		}
	}
	return nil
}
