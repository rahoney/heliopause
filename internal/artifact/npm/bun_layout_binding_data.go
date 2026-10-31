package npm

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"slices"
	"strings"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func bunInspectionManifest(lock BunLock) ([]byte, error) {
	packages := map[string]BunLockedPackage{}
	for _, pkg := range lock.packages {
		packages[pkg.Key] = pkg
	}
	manifest := map[string]any{"private": true}
	groups := map[string]map[string]string{}
	for _, root := range lock.roots {
		pkg, present := packages[root.Instance]
		if !present || pkg.Name != root.Name || root.Group != "dependencies" && root.Group != "devDependencies" {
			return nil, errBunInspectionLayout
		}
		if groups[root.Group] == nil {
			groups[root.Group] = map[string]string{}
		}
		if previous, present := groups[root.Group][root.Name]; present && previous != pkg.Version {
			return nil, errBunInspectionLayout
		}
		groups[root.Group][root.Name] = pkg.Version
	}
	for group, dependencies := range groups {
		manifest[group] = dependencies
	}
	return json.Marshal(manifest)
}

func bunInspectionLayoutDigest(input BunInspectionInput, layout BunLayoutArchive, manifest []byte) (domain.ContentDigest, error) {
	type subject struct {
		Source, Name, Version, Variant, Digest, Declared, Observed, Handle string
		Size                                                               uint64
	}
	var subjects []subject
	for _, artifact := range input.artifacts {
		declared, _ := artifact.DeclaredIntegrity()
		observed, _ := artifact.ObservedIntegrity()
		identity := artifact.Identity()
		subjects = append(subjects, subject{identity.Source().String(), identity.Name(), identity.Version(), identity.Variant(), artifact.Digest().String(), declared, observed, artifact.ContentHandle(), artifact.SizeBytes()})
	}
	slices.SortFunc(subjects, func(a, b subject) int { return strings.Compare(a.Name+"@"+a.Version, b.Name+"@"+b.Version) })
	type entry struct {
		Name, Link, Digest string
		Kind               byte
		Mode, Size         int64
		UID, GID           int
	}
	entries := layout.Entries()
	slices.SortFunc(entries, func(a, b BunLayoutEntry) int { return strings.Compare(a.Name, b.Name) })
	records := make([]entry, 0, len(entries))
	for _, e := range entries {
		records = append(records, entry{e.Name, e.Link, e.Digest.String(), e.Kind, e.Mode, e.Size, e.UID, e.GID})
	}
	document := struct {
		Schema, Graph, Runtime string
		Subjects               []subject
		Entries                []entry
		Manifest               json.RawMessage
	}{"bun-inspection-layout/v1", input.context.bindingDigest().String(), input.context.runtime, subjects, records, manifest}
	body, err := json.Marshal(document)
	if err != nil {
		return domain.ContentDigest{}, err
	}
	sum := sha256.Sum256(body)
	return domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
}
