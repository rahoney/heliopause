package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"slices"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
)

var errPNPMInspectionPayload = errors.New("pnpm inspection payload differs from its bound execution data")

type pnpmPayloadEntry struct {
	header      artifactnpm.PNPMLayoutEntry
	logicalSize int64
}

// Only this controller-owned renderer constructs a payload. It is not a native
// archive, original project controls, Policy decision or Host publication.
type pnpmInspectionPayload struct {
	kind           nativeRuntimeToolKind
	body           []byte
	digest, layout domain.ContentDigest
	entries        map[string]pnpmPayloadEntry
	groups         map[string]string
}

func preparePNPMInspectionPayload(ctx context.Context, layout artifactnpm.PNPMInspectionLayout) (pnpmInspectionPayload, error) {
	return prepareNPMInspectionPayload(ctx, layout, nativeRuntimePNPM)
}

// Both callers supply an opaque, fully bound layout; no runtime implementation
// or execution authority is exposed through these data getters.
type npmInspectionPayloadLayout interface {
	Digest() domain.ContentDigest
	InspectionManifest() []byte
	Entries() []artifactnpm.PNPMLayoutEntry
	ReadFile(string) ([]byte, error)
}

func prepareNPMInspectionPayload(ctx context.Context, layout npmInspectionPayloadLayout, kind nativeRuntimeToolKind) (pnpmInspectionPayload, error) {
	if _, ok := nativeProjectIdentity(kind); !ok {
		return pnpmInspectionPayload{}, errPNPMInspectionPayload
	}

	if ctx == nil || layout.Digest().String() == "" || len(layout.Entries()) == 0 {
		return pnpmInspectionPayload{}, errPNPMInspectionPayload
	}
	if err := ctx.Err(); err != nil {
		return pnpmInspectionPayload{}, errors.Join(errPNPMInspectionPayload, err, context.Cause(ctx))
	}
	manifest := layout.InspectionManifest()
	if len(manifest) == 0 || len(manifest) > artifactnpm.ProjectControlLimit {
		return pnpmInspectionPayload{}, errPNPMInspectionPayload
	}
	selected := map[string]artifactnpm.PNPMLayoutEntry{}
	for _, entry := range layout.Entries() {
		if kind == nativeRuntimePNPM && (entry.Name == "node_modules/.modules.yaml" || entry.Name == "node_modules/.pnpm/lock.yaml" || entry.Name == "node_modules/.pnpm-workspace-state-v1.json") {
			continue
		}
		selected[entry.Name] = entry
	}
	groups, err := pnpmPayloadFileGroups(layout.Entries(), selected)
	if err != nil {
		return pnpmInspectionPayload{}, err
	}
	payload := pnpmInspectionPayload{kind: kind, layout: layout.Digest(), entries: map[string]pnpmPayloadEntry{}, groups: map[string]string{}}
	var output bytes.Buffer
	writer := tar.NewWriter(&output)
	emit := func(entry artifactnpm.PNPMLayoutEntry, logicalSize int64, body []byte) error {
		if err := ctx.Err(); err != nil {
			return errors.Join(errPNPMInspectionPayload, err, context.Cause(ctx))
		}
		if output.Len() > artifactnpm.PNPMLayoutArchiveLimit-len(body) {
			return errPNPMInspectionPayload
		}
		header := tar.Header{Name: entry.Name, Typeflag: entry.Kind, Mode: entry.Mode, Uid: entry.UID, Gid: entry.GID, Size: entry.Size, Linkname: entry.Link}
		if err := writer.WriteHeader(&header); err != nil {
			return errors.Join(errPNPMInspectionPayload, err)
		}
		if _, err := writer.Write(body); err != nil {
			return errors.Join(errPNPMInspectionPayload, err)
		}
		if output.Len() > artifactnpm.PNPMLayoutArchiveLimit {
			return errPNPMInspectionPayload
		}
		payload.entries[entry.Name] = pnpmPayloadEntry{entry, logicalSize}
		return nil
	}
	root := artifactnpm.PNPMLayoutEntry{Name: "project", Kind: tar.TypeDir, Mode: 0o700, UID: 1000, GID: 1000}
	if err := emit(root, 0, nil); err != nil {
		return pnpmInspectionPayload{}, err
	}
	manifestDigest, err := pnpmPayloadSHA256(manifest)
	if err != nil {
		return pnpmInspectionPayload{}, err
	}
	rootManifest := artifactnpm.PNPMLayoutEntry{Name: "project/package.json", Kind: tar.TypeReg, Mode: 0o600, UID: 1000, GID: 1000, Size: int64(len(manifest)), Digest: manifestDigest}
	if err := emit(rootManifest, rootManifest.Size, manifest); err != nil {
		return pnpmInspectionPayload{}, err
	}
	payload.groups[rootManifest.Name] = rootManifest.Name
	names := make([]string, 0, len(selected))
	for name := range selected {
		names = append(names, name)
	}
	slices.Sort(names)
	// Directories first, then one regular representative per inode class, then
	// aliases and symlinks. No forward hardlink extraction dependency remains.
	for _, name := range names {
		entry := selected[name]
		if entry.Kind != tar.TypeDir {
			continue
		}
		entry.Name = "project/" + name
		if err := emit(entry, 0, nil); err != nil {
			return pnpmInspectionPayload{}, err
		}
	}
	for _, name := range names {
		if groups[name] != name {
			continue
		}
		entry := selected[name]
		body, err := layout.ReadFile(name)
		if err != nil {
			return pnpmInspectionPayload{}, errors.Join(errPNPMInspectionPayload, err)
		}
		digest, err := pnpmPayloadSHA256(body)
		if err != nil || digest != entry.Digest {
			return pnpmInspectionPayload{}, errors.Join(errPNPMInspectionPayload, err)
		}
		entry.Name = "project/" + name
		entry.Kind = tar.TypeReg
		entry.Link = ""
		entry.Size = int64(len(body))
		if err := emit(entry, entry.Size, body); err != nil {
			return pnpmInspectionPayload{}, err
		}
		payload.groups[entry.Name] = entry.Name
	}
	for _, name := range names {
		entry := selected[name]
		switch entry.Kind {
		case tar.TypeReg, tar.TypeLink:
			representative := groups[name]
			if representative == name {
				continue
			}
			target, present := payload.entries["project/"+representative]
			if !present || entry.Mode != target.header.Mode || entry.UID != target.header.UID || entry.GID != target.header.GID || entry.Digest != target.header.Digest {
				return pnpmInspectionPayload{}, errPNPMInspectionPayload
			}
			entry.Name = "project/" + name
			entry.Kind = tar.TypeLink
			entry.Link = "project/" + representative
			entry.Size = 0
			if err := emit(entry, target.logicalSize, nil); err != nil {
				return pnpmInspectionPayload{}, err
			}
			payload.groups[entry.Name] = target.header.Name
		case tar.TypeSymlink:
			entry.Name = "project/" + name
			if err := emit(entry, 0, nil); err != nil {
				return pnpmInspectionPayload{}, err
			}
		case tar.TypeDir:
		default:
			return pnpmInspectionPayload{}, errPNPMInspectionPayload
		}
	}
	if err := writer.Close(); err != nil {
		return pnpmInspectionPayload{}, errors.Join(errPNPMInspectionPayload, err)
	}
	if output.Len() > artifactnpm.PNPMLayoutArchiveLimit || len(payload.entries) != len(selected)+2 {
		return pnpmInspectionPayload{}, errPNPMInspectionPayload
	}
	payload.body = output.Bytes()
	payload.digest, err = pnpmPayloadSHA256(payload.body)
	if err != nil {
		return pnpmInspectionPayload{}, err
	}
	parsed, err := artifactnpm.ParsePNPMInspectionPayloadArchive(ctx, payload.body)
	if err != nil {
		return pnpmInspectionPayload{}, errors.Join(errPNPMInspectionPayload, err, ctx.Err(), context.Cause(ctx))
	}
	if err := comparePNPMInspectionPayload(ctx, payload, parsed); err != nil {
		return pnpmInspectionPayload{}, err
	}
	return payload, nil
}

func pnpmPayloadSHA256(body []byte) (domain.ContentDigest, error) {
	sum := sha256.Sum256(body)
	return domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
}

// Project the complete alias closure onto selected payload members. The
// lexicographic representative is independent of native tar traversal order.
func pnpmPayloadFileGroups(all []artifactnpm.PNPMLayoutEntry, selected map[string]artifactnpm.PNPMLayoutEntry) (map[string]string, error) {
	index := make(map[string]artifactnpm.PNPMLayoutEntry, len(all))
	for _, entry := range all {
		index[entry.Name] = entry
	}
	roots := map[string]string{}
	minimum := map[string]string{}
	for name, entry := range selected {
		if entry.Kind != tar.TypeReg && entry.Kind != tar.TypeLink {
			continue
		}
		current := name
		closed := false
		for depth := 0; depth <= 64; depth++ {
			e, present := index[current]
			if !present {
				return nil, errPNPMInspectionPayload
			}
			if e.Kind == tar.TypeReg {
				closed = true
				break
			}
			if e.Kind != tar.TypeLink {
				return nil, errPNPMInspectionPayload
			}
			current = e.Link
		}
		if !closed {
			return nil, errPNPMInspectionPayload
		}
		roots[name] = current
		if previous, exists := minimum[current]; !exists || name < previous {
			minimum[current] = name
		}
	}
	groups := make(map[string]string, len(roots))
	for name, root := range roots {
		groups[name] = minimum[root]
	}
	return groups, nil
}

func comparePNPMInspectionPayload(ctx context.Context, expected pnpmInspectionPayload, actual artifactnpm.PNPMLayoutArchive) error {
	if ctx == nil || expected.digest.String() == "" || expected.layout.String() == "" || len(expected.entries) == 0 {
		return errPNPMInspectionPayload
	}
	entries := actual.Entries()
	if len(entries) != len(expected.entries) {
		return errPNPMInspectionPayload
	}
	selected := make(map[string]artifactnpm.PNPMLayoutEntry, len(entries))
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return errors.Join(errPNPMInspectionPayload, err, context.Cause(ctx))
		}
		wanted, present := expected.entries[entry.Name]
		if !present || entry.Mode != wanted.header.Mode || entry.UID != wanted.header.UID || entry.GID != wanted.header.GID {
			return errPNPMInspectionPayload
		}
		switch wanted.header.Kind {
		case tar.TypeDir:
			if entry.Kind != tar.TypeDir {
				return errPNPMInspectionPayload
			}
		case tar.TypeSymlink:
			if entry.Kind != tar.TypeSymlink || entry.Link != wanted.header.Link {
				return errPNPMInspectionPayload
			}
		case tar.TypeReg, tar.TypeLink:
			if entry.Kind != tar.TypeReg && entry.Kind != tar.TypeLink || entry.Digest != wanted.header.Digest {
				return errPNPMInspectionPayload
			}
			body, err := actual.ReadFile(entry.Name)
			if err != nil || int64(len(body)) != wanted.logicalSize {
				return errors.Join(errPNPMInspectionPayload, err)
			}
		default:
			return errPNPMInspectionPayload
		}
		selected[entry.Name] = entry
	}
	groups, err := pnpmPayloadFileGroups(entries, selected)
	if err != nil || len(groups) != len(expected.groups) {
		return errors.Join(errPNPMInspectionPayload, err)
	}
	for name, group := range expected.groups {
		if groups[name] != group {
			return errPNPMInspectionPayload
		}
	}
	return nil
}

func copyPNPMInspectionPayload(ctx context.Context, runner CommandRunner, container string, payload pnpmInspectionPayload) error {
	return copyNPMInspectionPayload(ctx, runner, container, payload, nativeRuntimePNPM)
}

func copyNPMInspectionPayload(ctx context.Context, runner CommandRunner, container string, payload pnpmInspectionPayload, kind nativeRuntimeToolKind) error {
	spec, known := nativeProjectIdentity(kind)
	if !known || payload.kind != kind {
		return errPNPMInspectionPayload
	}

	input, ok := runner.(inputCommandRunner)
	_, streamOK := runner.(streamingDockerOutput)
	if ctx == nil || !ok || !streamOK || !exactObservationContainerID(container) || len(payload.body) == 0 || len(payload.body) > artifactnpm.PNPMLayoutArchiveLimit || payload.layout.String() == "" {
		return errPNPMInspectionPayload
	}
	if err := ctx.Err(); err != nil {
		return errors.Join(errPNPMInspectionPayload, err, context.Cause(ctx))
	}
	digest, err := pnpmPayloadSHA256(payload.body)
	if err != nil || digest != payload.digest {
		return errors.Join(errPNPMInspectionPayload, err)
	}
	if err := input.RunInput(ctx, bytes.NewReader(payload.body), "docker", "cp", "--archive", "-", container+":"+spec.root); err != nil {
		return errors.Join(errPNPMInspectionPayload, err, ctx.Err(), context.Cause(ctx))
	}
	return verifyNPMInspectionPayload(ctx, runner, container, payload, kind)
}

func verifyNPMInspectionPayload(ctx context.Context, runner CommandRunner, container string, payload pnpmInspectionPayload, kind nativeRuntimeToolKind) error {
	spec, known := nativeProjectIdentity(kind)
	if !known || payload.kind != kind {
		return errPNPMInspectionPayload
	}

	stream, ok := runner.(streamingDockerOutput)
	if ctx == nil || !ok || !exactObservationContainerID(container) || payload.digest.String() == "" {
		return errPNPMInspectionPayload
	}
	if err := ctx.Err(); err != nil {
		return errors.Join(errPNPMInspectionPayload, err, context.Cause(ctx))
	}
	output := nativeToolArchiveOutput{limit: artifactnpm.PNPMLayoutArchiveLimit}
	if err := stream.RunOutput(ctx, &output, "docker", "cp", container+":"+spec.directory, "-"); err != nil || output.failed {
		return errors.Join(errPNPMInspectionPayload, err, ctx.Err(), context.Cause(ctx))
	}
	actual, err := artifactnpm.ParsePNPMInspectionPayloadArchive(ctx, output.body)
	if err != nil {
		return errors.Join(errPNPMInspectionPayload, err, ctx.Err(), context.Cause(ctx))
	}
	return comparePNPMInspectionPayload(ctx, payload, actual)
}
