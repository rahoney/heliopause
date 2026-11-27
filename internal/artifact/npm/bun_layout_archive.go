package npm

import "context"

// BunLayoutEntry is unverified bounded tree data, not source or execution proof.
type BunLayoutEntry = PNPMLayoutEntry

// BunLayoutArchive reuses only the existing bounded data envelope. PNPM store,
// metadata and shim semantics are not applied to the Bun hoisted tree.
type BunLayoutArchive struct{ native PNPMLayoutArchive }

func ParseBunLayoutArchive(ctx context.Context, body []byte) (BunLayoutArchive, error) {
	layout, err := parsePNPMTreeArchive(ctx, body, "node_modules")
	if err != nil {
		return BunLayoutArchive{}, err
	}
	return BunLayoutArchive{layout}, nil
}

func (l BunLayoutArchive) Entries() []BunLayoutEntry            { return l.native.Entries() }
func (l BunLayoutArchive) ReadFile(name string) ([]byte, error) { return l.native.ReadFile(name) }

// RestrictDirectoryWrites changes an unverified entry view only. It retains
// native raw bytes and all regular file/link/mode checks for the full binder.
func (l BunLayoutArchive) RestrictDirectoryWrites() BunLayoutArchive {
	return BunLayoutArchive{l.native.RestrictDirectoryWrites()}
}
