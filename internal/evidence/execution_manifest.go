package evidence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"

	"github.com/rahoney/heliopause/internal/core/domain"
)

type executionManifestPayload struct {
	Schema      string                  `json:"schema"`
	OperationID string                  `json:"operation_id"`
	Resolver    manifestResolver        `json:"resolver"`
	Primary     string                  `json:"primary"`
	Policy      manifestPolicy          `json:"policy"`
	Entries     []manifestEntry         `json:"entries"`
	Edges       []manifestEdge          `json:"edges"`
	Execution   executionManifestMember `json:"execution"`
}

type executionManifestMember struct {
	Command           string `json:"command"`
	Path              string `json:"path"`
	SHA256            string `json:"sha256"`
	Size              uint64 `json:"size"`
	RuntimeIdentity   string `json:"runtime_identity"`
	SourceFingerprint string `json:"source_fingerprint"`
	PolicyFingerprint string `json:"policy_fingerprint"`
}

// BuildExecution records the current approved set without an installation
// target. Arguments and program output never enter these storage records.
// The bundle is not an execution request or an admission authorization.
func (Generator) BuildExecution(ctx context.Context, execution domain.ExecutionContext) (domain.VerifiedBundle, error) {
	if ctx == nil || !execution.Valid() || !validContextText(execution.RuntimeIdentity()) || !validContextText(execution.Resolution().RuntimeIdentity()) {
		return domain.VerifiedBundle{}, errors.New("execution records require a valid context and approved set")
	}
	if err := ctx.Err(); err != nil {
		return domain.VerifiedBundle{}, err
	}
	set := execution.Set()
	entries, edges, _, err := manifestEntries(set)
	if err != nil {
		return domain.VerifiedBundle{}, err
	}
	decision := set.Decision()
	reasons := decision.Reasons()
	sort.Strings(reasons)
	payload := executionManifestPayload{
		Schema: "helox.verified-execution-manifest/v1", OperationID: execution.OperationID().String(),
		Resolver: manifestResolver{execution.Resolution().RuntimeIdentity(), execution.SourceFingerprint().String()},
		Primary:  set.Inspected().Graph().Primary().String(),
		Policy:   manifestPolicy{string(decision.Decision()), decision.PolicyID(), decision.Version(), reasons},
		Entries:  entries, Edges: edges,
		Execution: executionManifestMember{
			Command: execution.Command(), Path: execution.EntryPath(), SHA256: execution.EntryDigest().String(),
			Size: execution.EntrySize(), RuntimeIdentity: execution.RuntimeIdentity(),
			SourceFingerprint: execution.SourceFingerprint().String(), PolicyFingerprint: execution.PolicyFingerprint().String(),
		},
	}
	unsigned, err := canonicalManifestObject(payload)
	if err != nil {
		return domain.VerifiedBundle{}, err
	}
	sum := sha256.Sum256(unsigned)
	manifestID, err := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
	if err != nil {
		return domain.VerifiedBundle{}, err
	}
	document, err := json.Marshal(struct {
		ManifestID string `json:"manifest_id"`
		executionManifestPayload
	}{manifestID.String(), payload})
	if err != nil {
		return domain.VerifiedBundle{}, err
	}
	sbom, err := buildSBOM(manifestID, set)
	if err != nil {
		return domain.VerifiedBundle{}, err
	}
	return domain.NewVerifiedBundle(manifestID, set, document, sbom)
}
