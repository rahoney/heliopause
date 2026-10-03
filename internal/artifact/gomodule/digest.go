package gomodule

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"

	"github.com/rahoney/heliopause/internal/core/domain"
)

// FreezeResolutionDigest binds normalized module/checksum identity, the complete
// requirement graph and project controls. Adapter-local cache filenames and
// origin metadata do not become reproducible graph identity or trust authority.
func FreezeResolutionDigest(records []DownloadRecord, graphOutput, goMod, goSum []byte) (domain.ContentDigest, error) {
	if len(goSum) == 0 || len(goSum) > MaxProjectControlBytes {
		return domain.ContentDigest{}, errors.New("go project sums exceed bound")
	}
	if _, err := normalizeProjectGraph(graphOutput, records, goMod); err != nil {
		return domain.ContentDigest{}, err
	}
	type moduleIdentity struct{ Path, Version, Sum, GoModSum string }
	identities := make([]moduleIdentity, 0, len(records))
	for _, record := range records {
		identities = append(identities, moduleIdentity{record.Path, record.Version, record.Sum, record.GoModSum})
	}
	sort.Slice(identities, func(i, j int) bool {
		return identities[i].Path < identities[j].Path
	})
	modHash, sumHash, graphHash := sha256.Sum256(goMod), sha256.Sum256(goSum), sha256.Sum256(graphOutput)
	body, err := json.Marshal(struct {
		Source, SumDB   string
		Modules         []moduleIdentity
		Mod, Sum, Graph [32]byte
	}{proxyEndpoint, sumDBEndpoint, identities, modHash, sumHash, graphHash})
	if err != nil {
		return domain.ContentDigest{}, errors.New("encode frozen Go resolution identity")
	}
	digest := sha256.Sum256(body)
	return domain.NewSHA256Digest(hex.EncodeToString(digest[:]))
}
