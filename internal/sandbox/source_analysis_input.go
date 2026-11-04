package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/rahoney/heliopause/internal/core/ports"
)

const (
	sourceAnalysisRoot           = "/haa-source"
	sourceAnalysisTargetsName    = "targets.json"
	sourceAnalysisFileLimit      = ports.SourceAnalysisFileLimit
	sourceAnalysisTotalLimit     = ports.SourceAnalysisTotalLimit
	sourceAnalysisTargetLimit    = ports.SourceAnalysisTargetLimit
	sourceAnalysisMetadataLimit  = 8 << 20
	sourceAnalysisVolumeCapacity = sourceAnalysisTotalLimit + sourceAnalysisMetadataLimit
)

type sourceAnalysisTarget struct {
	name, original, language, sha256 string
	size                             int64
}

// Files remain private controller DATA for the duration of one call. The
// writer authenticates them again while copying; mutable input is not adopted.
type sourceAnalysisPreparedInput struct {
	files                      []ports.SourceAnalysisFile
	targets                    []sourceAnalysisTarget
	metadata                   []byte
	metadataSHA256, manifestID string
}

func (p *sourceAnalysisPreparedInput) scanTar(input io.Reader) error {
	failure := errors.New("source analysis input readback is unavailable or changed")
	if p == nil || len(p.targets) == 0 || len(p.metadata) == 0 || len(p.manifestID) != 64 {
		return failure
	}
	expected := make(map[string]readonlyInputFile, len(p.targets)+1)
	for _, target := range p.targets {
		expected[target.name] = readonlyInputFile{target.size, target.sha256}
	}
	expected[sourceAnalysisTargetsName] = readonlyInputFile{int64(len(p.metadata)), p.metadataSHA256}
	return scanReadonlyInputTar(input, expected, failure)
}

func prepareSourceAnalysisInput(ctx context.Context, input ports.SourceAnalysisInput) (*sourceAnalysisPreparedInput, error) {
	if ctx == nil {
		return nil, errors.New("source analysis context is unavailable")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if input.Artifact.SizeBytes() == 0 || input.Artifact.Digest().String() == "" || len(input.Files) == 0 || len(input.Files) > sourceAnalysisTargetLimit {
		return nil, errors.New("source analysis input is empty, unbound or excessive")
	}
	prepared := &sourceAnalysisPreparedInput{files: input.Files}
	seen := make(map[string]bool, len(input.Files))
	rows := make([]any, 0, len(input.Files))
	var total int64
	manifest := sha256.New()
	_, _ = io.WriteString(manifest, input.Artifact.Digest().String()+"\n")
	for index, file := range input.Files {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if file.Path == "" || len(file.Path) > 4096 || !utf8.ValidString(file.Path) || strings.HasPrefix(file.Path, "/") || strings.Contains(file.Path, "\\") || path.Clean(file.Path) != file.Path || file.Path == "." || file.Path == ".." || strings.HasPrefix(file.Path, "../") || seen[file.Path] {
			return nil, errors.New("source analysis path is unsafe or duplicated")
		}
		for _, character := range file.Path {
			if unicode.IsControl(character) {
				return nil, errors.New("source analysis path contains a control character")
			}
		}
		seen[file.Path] = true
		extension := ""
		switch file.Language {
		case "javascript":
			extension = ".js"
		case "typescript":
			extension = ".ts"
		case "python":
			extension = ".py"
		case "go":
			extension = ".go"
		case "rust":
			extension = ".rs"
		case "hcl":
			extension = ".tf"
		default:
			return nil, errors.New("source analysis language is unsupported")
		}
		// Preserve the selected JSX grammar without exposing the original name
		// as a filesystem path or command argument inside the runtime.
		if suffix := strings.ToLower(path.Ext(file.Path)); (file.Language == "javascript" && suffix == ".jsx") || (file.Language == "typescript" && suffix == ".tsx") {
			extension = suffix
		}
		size := int64(len(file.Body))
		if size > sourceAnalysisFileLimit || size > sourceAnalysisTotalLimit-total {
			return nil, errors.New("source analysis byte limit exceeded")
		}
		total += size
		digest := sha256.Sum256(file.Body)
		target := sourceAnalysisTarget{name: fmt.Sprintf("source-%05d%s", index, extension), original: file.Path, language: file.Language, sha256: hex.EncodeToString(digest[:]), size: size}
		prepared.targets = append(prepared.targets, target)
		rows = append(rows, []any{"CodeTarget", map[string]any{"path": map[string]string{"fpath": sourceAnalysisRoot + "/" + target.name, "ppath": "/" + file.Path}, "analyzer": file.Language, "products": []string{"sast"}}})
		// A JSON record avoids delimiter ambiguity in an allowed source name.
		if err := json.NewEncoder(manifest).Encode([]any{target.name, target.original, target.language, target.sha256, target.size}); err != nil {
			return nil, err
		}
	}
	metadata, err := json.Marshal([]any{"Targets", rows})
	if err != nil || len(metadata) > sourceAnalysisMetadataLimit {
		return nil, errors.New("source analysis target metadata is excessive")
	}
	prepared.metadata = metadata
	digest := sha256.Sum256(metadata)
	prepared.metadataSHA256 = hex.EncodeToString(digest[:])
	_, _ = io.WriteString(manifest, prepared.metadataSHA256)
	prepared.manifestID = hex.EncodeToString(manifest.Sum(nil))
	return prepared, nil
}

func (p *sourceAnalysisPreparedInput) writeTar(ctx context.Context, output io.Writer) (resultErr error) {
	if p == nil || ctx == nil || output == nil || len(p.files) != len(p.targets) || len(p.metadata) == 0 {
		return errors.New("source analysis input writer is unavailable")
	}
	archive := tar.NewWriter(output)
	defer func() { resultErr = errors.Join(resultErr, archive.Close()) }()
	for index, target := range p.targets {
		if err := ctx.Err(); err != nil {
			return err
		}
		body := p.files[index].Body
		if int64(len(body)) != target.size {
			return errors.New("source analysis input changed before copy")
		}
		if err := archive.WriteHeader(&tar.Header{Name: target.name, Typeflag: tar.TypeReg, Mode: 0444, Size: target.size}); err != nil {
			return err
		}
		digest := sha256.New()
		count, err := io.CopyN(io.MultiWriter(archive, digest), ports.DownloadReader(ctx, bytes.NewReader(body), target.size), target.size)
		if err != nil || count != target.size || hex.EncodeToString(digest.Sum(nil)) != target.sha256 {
			return errors.Join(errors.New("source analysis input bytes changed during copy"), err)
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	digest := sha256.Sum256(p.metadata)
	if hex.EncodeToString(digest[:]) != p.metadataSHA256 {
		return errors.New("source analysis target metadata changed")
	}
	if err := archive.WriteHeader(&tar.Header{Name: sourceAnalysisTargetsName, Typeflag: tar.TypeReg, Mode: 0444, Size: int64(len(p.metadata))}); err != nil {
		return err
	}
	_, err := archive.Write(p.metadata)
	return err
}

func (p *sourceAnalysisPreparedInput) introduce(ctx context.Context, runner CommandRunner, container string) error {
	failure := errors.New("source analysis input transport is unavailable")
	if p == nil || len(p.targets) == 0 {
		return failure
	}
	if err := copyReadonlyInputTar(ctx, runner, container, sourceAnalysisRoot, func(output io.Writer) error { return p.writeTar(ctx, output) }, failure); err != nil {
		return err
	}
	return p.readback(ctx, runner, container)
}

func (p *sourceAnalysisPreparedInput) readback(ctx context.Context, runner CommandRunner, container string) error {
	failure := errors.New("source analysis input readback is unavailable")
	if p == nil || len(p.targets) == 0 {
		return failure
	}
	return readReadonlyInputTar(ctx, runner, container, sourceAnalysisRoot, p.scanTar, failure)
}

func createSourceAnalysisVolume(ctx context.Context, runner CommandRunner, transaction string, input *sourceAnalysisPreparedInput) (*readonlyToolVolume, error) {
	if input == nil || len(input.manifestID) != 64 || len(input.targets) == 0 {
		return nil, errors.New("source analysis input volume is unbound")
	}
	volume, err := createInputVolume(ctx, runner, transaction+"-source-analysis", input.manifestID, sourceAnalysisVolumeCapacity, false, projectBuildVolumeConfig{destination: sourceAnalysisRoot})
	if volume.name == "" {
		return nil, err
	}
	return &readonlyToolVolume{source: input, volume: volume}, err
}
