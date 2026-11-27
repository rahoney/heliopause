package promotion

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"path/filepath"
	"strings"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/sandbox"
)

// This concrete proof binds original DATA to a held transaction. It neither
// authenticates an archive nor enrolls ownership or authorizes publication.
type uvNativeToolScriptProof struct {
	tx          *pypiVenvTransaction
	interpreter []toolInterpreterMember
	client      toolClient
	exposed     map[string]bool
	records     map[string]nativeToolRecordClaim
}

type nativeToolRecordClaim struct {
	digest       string
	size         int64
	actualDigest string
	actualSize   int64
}

func newUVNativeToolScriptProof(ctx context.Context, tx *pypiVenvTransaction) (*uvNativeToolScriptProof, error) {
	return newUVNativeToolScriptProofForClient(ctx, tx, toolClientUV, nil)
}

func newUVNativeToolScriptProofForClient(ctx context.Context, tx *pypiVenvTransaction, client toolClient, commands []artifactpypi.ToolCommand) (*uvNativeToolScriptProof, error) {
	if ctx == nil || tx == nil || !client.valid() {
		return nil, errors.New("native UV script proof requires a held target")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := tx.checkBefore(); err != nil {
		return nil, err
	}
	if err := toolOwnedDirectory(tx.plan.root); err != nil {
		return nil, err
	}
	layout, err := discoverUVPythonVenv(tx.plan.root)
	if err != nil || layout != tx.plan {
		return nil, errors.New("native UV script target layout differs from its guard")
	}
	body, err := readBoundedPromotionFile(filepath.Join(tx.plan.root, "pyvenv.cfg"), 64<<10)
	if err != nil {
		return nil, err
	}
	version, err := uvPythonVenvConfigVersion(body)
	if err != nil || version != sandbox.PinnedPythonRuntime().PythonVersion {
		return nil, errors.New("native UV scripts require the pinned original Python layout")
	}
	interpreter, err := snapshotToolInterpreter(filepath.Join(tx.plan.root, "bin", "python"))
	if err != nil {
		return nil, err
	}
	p := &uvNativeToolScriptProof{tx: tx, interpreter: interpreter, client: client}
	if client == toolClientPipx {
		if len(commands) == 0 {
			return nil, errors.New("native pipx exposed command input is absent")
		}
		p.exposed = map[string]bool{}
		for _, command := range commands {
			if command.Name == "" || command.Name == "." || command.Name == ".." || filepath.Base(command.Name) != command.Name || command.Path != filepath.Join(tx.plan.root, "bin", command.Name) || p.exposed[command.Name] {
				return nil, errors.New("native pipx exposed command input is invalid")
			}
			p.exposed[command.Name] = true
		}
	}
	if err := p.verify(ctx); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *uvNativeToolScriptProof) verify(ctx context.Context) error {
	if p == nil || p.tx == nil || ctx == nil || len(p.interpreter) == 0 ||
		p.interpreter[0].path != filepath.Join(p.tx.plan.root, "bin", "python") {
		return errors.New("native UV script proof is unavailable")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := p.tx.checkBefore(); err != nil {
		return err
	}
	return verifyToolInterpreterSnapshot(p.interpreter)
}

func (p *uvNativeToolScriptProof) scriptClaim(project *pypiNativeDistribution, entry artifactpypi.EntryPoint) (string, error) {
	if entry.Group != "console_scripts" || len(entry.Extras) != 0 || entry.Name == "" ||
		entry.Name == "." || entry.Name == ".." || filepath.Base(entry.Name) != entry.Name ||
		strings.ContainsAny(entry.Name, "/\\\x00\r\n") {
		return "", errors.New("native UV tool entrypoint is unsupported")
	}
	key := "scripts/" + entry.Name
	claim, ok := project.files[key]
	if !ok || claim.unhashed || claim.destination.Scheme != "scripts" ||
		claim.destination.Relative != entry.Name || claim.destination.Final != filepath.Join(p.tx.plan.root, "bin", entry.Name) ||
		claim.original.Size <= 0 || claim.original.Size > 32<<10 ||
		claim.original.Mode.Perm()&0111 == 0 || claim.original.Mode.Perm()&0022 != 0 ||
		!claim.original.matches(claim.destination.Final) {
		return "", errors.New("native UV console script lacks its exact held executable claim")
	}
	body, err := readBoundedPromotionFile(claim.destination.Final, 32<<10)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(body)
	if int64(len(body)) != claim.original.Size || hex.EncodeToString(sum[:]) != claim.original.Digest ||
		!claim.original.matches(claim.destination.Final) {
		return "", errors.New("native UV console script bytes changed")
	}
	if p.client == toolClientPipx && p.exposed[entry.Name] {
		original, err := artifactpypi.PipxNativeUVScriptRecordBody(p.interpreter[0].path, body)
		record, found := p.records[key]
		sum := sha256.Sum256(original)
		if err != nil || !found || record.actualDigest != claim.original.Digest || record.actualSize != claim.original.Size || record.digest != hex.EncodeToString(sum[:]) || record.size != int64(len(original)) {
			return "", errors.New("native pipx script lacks its exact held pre-exposure RECORD binding")
		}
		body = original
	}
	if err := artifactpypi.ValidateUVNativeConsoleScript(entry, p.interpreter[0].path, body); err != nil {
		return "", err
	}
	return key, nil
}

func (p *uvNativeToolScriptProof) prove(ctx context.Context, project *pypiNativeDistribution, wheel artifactpypi.WheelInspection, digest string) ([]pypiDestination, []pypiNativeProofFile, error) {
	if project == nil {
		return nil, nil, errors.New("native UV original distribution is unavailable")
	}
	if err := p.verify(ctx); err != nil {
		return nil, nil, err
	}
	if _, err := domain.NewSHA256Digest(digest); err != nil {
		return nil, nil, err
	}
	destinations, files, err := proveNativePythonWheelFootprintWithUVToolScripts(nativeUVClient, project, wheel, digest,
		func() (pypiNativeRuntime, error) {
			return pypiNativeRuntime{}, errors.New("native UV tool proof cannot execute an original interpreter")
		}, p)
	if err != nil {
		return nil, nil, err
	}
	if err := p.verify(ctx); err != nil {
		return nil, nil, err
	}
	return destinations, files, nil
}

// This freezes an unapproved claim only. The authentic old wheel entrypoint and
// complete UV recipe must still be matched by scriptClaim before enrollment.
func (p *uvNativeToolScriptProof) capturePipxRecord(ctx context.Context, destination pypiDestination, snapshot pypiFileSnapshot, digest string, size int64) error {
	if p == nil || p.tx == nil || p.client != toolClientPipx || destination.Scheme != "scripts" || !p.exposed[destination.Relative] || destination.Final != filepath.Join(p.tx.plan.root, "bin", destination.Relative) {
		return errors.New("native pipx script RECORD lacks its held exposed destination")
	}
	if err := p.verify(ctx); err != nil {
		return err
	}
	body, err := readBoundedPromotionFile(destination.Final, 32<<10)
	if err != nil {
		return err
	}
	actual := sha256.Sum256(body)
	if int64(len(body)) != snapshot.Size || hex.EncodeToString(actual[:]) != snapshot.Digest || !snapshot.matches(destination.Final) {
		return errors.New("native pipx script snapshot changed before RECORD binding")
	}
	original, err := artifactpypi.PipxNativeUVScriptRecordBody(p.interpreter[0].path, body)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(original)
	if int64(len(original)) != size || hex.EncodeToString(sum[:]) != digest {
		return errors.New("native pipx RECORD differs from exact pre-exposure UV bytes")
	}
	if p.records == nil {
		p.records = map[string]nativeToolRecordClaim{}
	}
	key := destination.key()
	if _, present := p.records[key]; present {
		return errors.New("native pipx script RECORD binding is duplicated")
	}
	p.records[key] = nativeToolRecordClaim{digest, size, snapshot.Digest, snapshot.Size}
	return p.verify(ctx)
}
