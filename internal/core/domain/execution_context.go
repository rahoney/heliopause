package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"path"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	MaximumExecutionArguments     = 128
	MaximumExecutionArgumentBytes = 32 << 10
)

// ValidateExecutionArguments can run before resolution or acquisition. Values
// remain argv data, including empty strings, whitespace and newlines.
func ValidateExecutionArguments(arguments []string) error {
	if len(arguments) > MaximumExecutionArguments {
		return errors.New("execution arguments exceed count limit")
	}
	total := 0
	for _, argument := range arguments {
		if len(argument) > MaximumExecutionArgumentBytes-total || !utf8.ValidString(argument) || strings.IndexByte(argument, 0) >= 0 {
			return errors.New("execution argument encoding or byte limit is invalid")
		}
		total += len(argument)
	}
	return nil
}

// ValidateExecutionCommand validates an explicit command before any intake.
// Omitted selection is handled separately by the ecosystem's exact bin parser.
func ValidateExecutionCommand(command string) error {
	if validateBoundedText(command, 214, "execution command") != nil || command == "." || command == ".." || strings.ContainsAny(command, "/\\:") {
		return errors.New("execution command requires a bounded single name")
	}
	return nil
}

// ExecutionContextData joins an already verified set to one requested member.
// Composition supplies the required runtime and the adapter's measured member
// facts. Arguments are private request data; do not log or persist this input.
type ExecutionContextData struct {
	OperationID     OperationID
	Resolution      DependencyResolution
	Set             VerifiedSet
	EntryArtifact   AcquiredArtifact
	Command         string
	EntryPath       string
	EntryDigest     ContentDigest
	EntrySize       uint64
	RuntimeIdentity string
	Arguments       []string
}

// ExecutionContext freezes non-install metadata and argv after whole-set ALLOW.
// It does not authenticate member facts, admit a runtime, or authorize a Host
// command. Fresh inspection and introduction-time byte checks remain required.
// Its zero value is invalid.
type ExecutionContext struct {
	operationID OperationID
	resolution  DependencyResolution
	set         VerifiedSet
	artifact    AcquiredArtifact
	command     string
	entryPath   string
	entryDigest ContentDigest
	entrySize   uint64
	runtime     string
	arguments   []string
	policy      ContentDigest
}

func NewExecutionContext(data ExecutionContextData) (ExecutionContext, error) {
	invalid := errors.New("execution context requires matching approved graph, primary entry and bounded request data")
	graph, approved := data.Resolution.Graph(), data.Set.Inspected().Graph()
	if data.OperationID.value == "" || !data.Set.Valid() || data.Resolution.LockfileDigest().value == "" ||
		graph.primary.value == "" || graph.primary != approved.primary || len(graph.derivations) != 0 || len(approved.derivations) != 0 ||
		!slices.Equal(graph.nodes, approved.nodes) || !slices.Equal(graph.edges, approved.edges) {
		return ExecutionContext{}, invalid
	}
	if validateBoundedText(data.RuntimeIdentity, maxInstallTargetLength, "execution runtime identity") != nil ||
		ValidateExecutionCommand(data.Command) != nil || data.EntryDigest.value == "" || !validExecutionMemberPath(data.EntryPath) {
		return ExecutionContext{}, invalid
	}
	primaryMatches := 0
	for _, inspection := range data.Set.Inspected().Inspections() {
		if inspection.Node() == graph.Primary() && inspection.Artifact() == data.EntryArtifact {
			primaryMatches++
		}
	}
	if primaryMatches != 1 {
		return ExecutionContext{}, invalid
	}
	if err := ValidateExecutionArguments(data.Arguments); err != nil {
		return ExecutionContext{}, err
	}
	policy, err := executionPolicyFingerprint(data.Set)
	if err != nil {
		return ExecutionContext{}, err
	}
	return ExecutionContext{
		operationID: data.OperationID, resolution: data.Resolution, set: data.Set,
		artifact: data.EntryArtifact, command: data.Command, entryPath: data.EntryPath,
		entryDigest: data.EntryDigest, entrySize: data.EntrySize, runtime: data.RuntimeIdentity,
		arguments: append([]string(nil), data.Arguments...), policy: policy,
	}, nil
}

func validExecutionMemberPath(value string) bool {
	return value != "" && value != "." && len(value) <= maxInstallTargetLength && utf8.ValidString(value) &&
		path.Clean(value) == value && !path.IsAbs(value) && value != ".." && !strings.HasPrefix(value, "../") &&
		!strings.ContainsAny(value, "\\:") && !strings.ContainsFunc(value, unicode.IsControl)
}

// This fingerprint contains normalized Policy identities only. Program argv
// never becomes part of an artifact, policy or cache identity.
func executionPolicyFingerprint(set VerifiedSet) (ContentDigest, error) {
	return currentExecutionPolicyFingerprint(set.Decision(), set.Inspected().Inspections())
}

func currentExecutionPolicyFingerprint(decision PolicyDecision, inspections []DependencyInspection) (ContentDigest, error) {
	type policyVersion struct {
		Node, ID string
		Version  uint64
	}
	identities := []policyVersion{{ID: decision.PolicyID(), Version: decision.Version()}}
	for _, inspection := range inspections {
		entry := inspection.PolicyDecision()
		identities = append(identities, policyVersion{inspection.Node().String(), entry.PolicyID(), entry.Version()})
	}
	body, err := json.Marshal(identities)
	if err != nil {
		return ContentDigest{}, errors.New("execution Policy identity cannot be encoded")
	}
	sum := sha256.Sum256(body)
	return NewSHA256Digest(hex.EncodeToString(sum[:]))
}

func (c ExecutionContext) Valid() bool {
	return c.operationID.value != "" && c.set.Valid() && c.policy.value != "" && c.entryDigest.value != ""
}

func (c ExecutionContext) OperationID() OperationID         { return c.operationID }
func (c ExecutionContext) Resolution() DependencyResolution { return c.resolution }
func (c ExecutionContext) Set() VerifiedSet                 { return c.set }
func (c ExecutionContext) EntryArtifact() AcquiredArtifact  { return c.artifact }
func (c ExecutionContext) Command() string                  { return c.command }
func (c ExecutionContext) EntryPath() string                { return c.entryPath }
func (c ExecutionContext) EntryDigest() ContentDigest       { return c.entryDigest }
func (c ExecutionContext) EntrySize() uint64                { return c.entrySize }
func (c ExecutionContext) RuntimeIdentity() string          { return c.runtime }
func (c ExecutionContext) SourceFingerprint() ContentDigest { return c.resolution.LockfileDigest() }
func (c ExecutionContext) PolicyFingerprint() ContentDigest { return c.policy }
func (c ExecutionContext) Arguments() []string              { return append([]string(nil), c.arguments...) }
