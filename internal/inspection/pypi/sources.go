package pypi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/rahoney/heliopause/internal/inspection/content"
)

type sourceMember struct {
	owner          *sourceCapture
	name, language string
	size           uint64
	prefix         []byte
	ready          bool
	native         bool
	received       uint64
	body           bytes.Buffer
}

type sourceCapture struct {
	members []*sourceMember
	bytes   uint64
	limited bool
	targets int
	native  *content.BinaryInspection
	ctx     context.Context
}

func (c *sourceCapture) member(name string, size uint64) io.Writer {
	language := content.SourceLanguage(name)
	if c.limited || language == "" && size == 0 && (c.native == nil || !content.IsBinary(name, nil)) {
		return nil
	}
	member := &sourceMember{owner: c, name: name, size: size}
	if c.native != nil && size == 0 {
		member.selectContent(nil)
		member.finishNative()
	} else if language != "" && c.native == nil {
		member.selectLanguage(language)
		if c.limited {
			return nil
		}
	}
	return member
}

func (m *sourceMember) selectLanguage(language string) {
	m.ready, m.language = true, language
	if language == "" {
		return
	}
	c := m.owner
	if m.size > ports.SourceAnalysisFileLimit || m.size > ports.SourceAnalysisTotalLimit-c.bytes || c.targets >= ports.SourceAnalysisTargetLimit {
		c.limited, c.members, m.language = true, nil, ""
		return
	}
	c.bytes += m.size
	c.targets++
	c.members = append(c.members, m)
}

func (m *sourceMember) selectContent(prefix []byte) {
	if m.owner.native == nil || !content.IsBinary(m.name, prefix) {
		m.selectLanguage(content.SourceLanguage(m.name, prefix))
		return
	}
	m.ready, m.native = true, true
	c := m.owner
	if m.size > ports.SourceAnalysisFileLimit || m.size > ports.SourceAnalysisTotalLimit-c.bytes || c.targets >= ports.SourceAnalysisTargetLimit {
		c.limited, c.members, m.native = true, nil, false
		// Record an invalid selected member without allocating its body.
		// Report remains incomplete and keeps earlier validated signals.
		_ = c.native.Inspect(c.ctx, content.BinaryMember{Path: m.name, Size: m.size})
		return
	}
	c.bytes += m.size
	c.targets++
}

func (m *sourceMember) finishNative() {
	if !m.native || m.received != m.size {
		return
	}
	sum := sha256.Sum256(m.body.Bytes())
	digest, _ := domain.NewSHA256Digest(hex.EncodeToString(sum[:]))
	// Provisional facts stay private until the enclosing RECORD/archive
	// owner validates every byte. Neither this body nor its code is run.
	_ = m.owner.native.Inspect(m.owner.ctx, content.BinaryMember{Path: m.name, Body: m.body.Bytes(), Digest: digest})
	m.body = bytes.Buffer{}
	m.native = false
}

func (m *sourceMember) Write(data []byte) (int, error) {
	written := len(data)
	if uint64(written) > m.size-m.received {
		return 0, errors.New("captured member exceeds its declared size")
	}
	m.received += uint64(written)
	if m.owner.limited {
		return written, nil
	}
	if !m.ready {
		n := min(int(min(m.size, 512))-len(m.prefix), len(data))
		m.prefix = append(m.prefix, data[:n]...)
		data = data[n:]
		if len(m.prefix) < int(min(m.size, 512)) {
			return written, nil
		}
		m.selectContent(m.prefix)
		if m.language != "" || m.native {
			_, _ = m.body.Write(m.prefix) // bytes.Buffer.Write never fails.
		}
		m.prefix = nil
	}
	if m.language != "" || m.native {
		_, _ = m.body.Write(data)
	}
	m.finishNative()
	return written, nil
}

func (c *sourceCapture) files(ctx context.Context) ([]ports.SourceAnalysisFile, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if c.limited {
		return nil, errors.New("PyPI source selection exceeds fixed analysis bounds")
	}
	files := make([]ports.SourceAnalysisFile, 0, len(c.members))
	for _, member := range c.members {
		files = append(files, ports.SourceAnalysisFile{Path: member.name, Language: member.language, Body: member.body.Bytes()})
	}
	return files, nil
}

// InspectWheelSources preserves the typed wheel and graph accounting while
// retaining source DATA during the same RECORD digest/CRC pass. It grants no
// content verdict, including when the source selection is empty.
func (i *StaticInspector) InspectWheelSources(ctx context.Context, artifact domain.AcquiredArtifact) (artifactpypi.WheelInspection, []ports.SourceAnalysisFile, domain.InspectionReport, error) {
	var capture sourceCapture
	wheel, report, err := i.inspectWheel(ctx, artifact, capture.member)
	if err != nil || len(report.Findings()) != 0 {
		return artifactpypi.WheelInspection{}, nil, report, err
	}
	files, err := capture.files(ctx)
	if err != nil {
		return artifactpypi.WheelInspection{}, nil, report, err
	}
	return wheel, files, report, nil
}

// InspectSdistSources returns PEP 517 metadata and source DATA only after the
// complete bounded archive and compressed subject have passed validation.
func (i *StaticInspector) InspectSdistSources(ctx context.Context, artifact domain.AcquiredArtifact) (artifactpypi.SdistInspection, []ports.SourceAnalysisFile, domain.InspectionReport, error) {
	var capture sourceCapture
	inspection, report, err := i.inspect(ctx, artifact, capture.member)
	if err != nil || len(report.Findings()) != 0 {
		return artifactpypi.SdistInspection{}, nil, report, err
	}
	sdist, ok := inspection.(artifactpypi.SdistInspection)
	if !ok {
		return artifactpypi.SdistInspection{}, nil, report, errors.New("PyPI source distribution inspection is unavailable")
	}
	files, err := capture.files(ctx)
	if err != nil {
		return artifactpypi.SdistInspection{}, nil, report, err
	}
	return sdist, files, report, nil
}

// InspectWheelContent adds incremental native facts to the same typed wheel /
// RECORD pass. Source DATA is returned for the separate fixed AST analyzer.
func (i *StaticInspector) InspectWheelContent(ctx context.Context, artifact domain.AcquiredArtifact) (artifactpypi.WheelInspection, []ports.SourceAnalysisFile, domain.InspectionReport, error) {
	native, err := content.NewBinaryInspection(artifact)
	if err != nil {
		return artifactpypi.WheelInspection{}, nil, domain.InspectionReport{}, err
	}
	capture := sourceCapture{native: native, ctx: ctx}
	wheel, static, err := i.inspectWheel(ctx, artifact, capture.member)
	if err != nil || len(static.Findings()) != 0 {
		return artifactpypi.WheelInspection{}, nil, static, err
	}
	files, captureErr := capture.files(ctx)
	report, contentErr := capture.report(ctx, static)
	return wheel, files, report, errors.Join(captureErr, contentErr)
}

func (i *StaticInspector) InspectSdistContent(ctx context.Context, artifact domain.AcquiredArtifact) (artifactpypi.SdistInspection, []ports.SourceAnalysisFile, domain.InspectionReport, error) {
	native, err := content.NewBinaryInspection(artifact)
	if err != nil {
		return artifactpypi.SdistInspection{}, nil, domain.InspectionReport{}, err
	}
	capture := sourceCapture{native: native, ctx: ctx}
	inspection, static, err := i.inspect(ctx, artifact, capture.member)
	if err != nil || len(static.Findings()) != 0 {
		return artifactpypi.SdistInspection{}, nil, static, err
	}
	sdist, ok := inspection.(artifactpypi.SdistInspection)
	if !ok {
		return artifactpypi.SdistInspection{}, nil, static, errors.New("PyPI source distribution inspection is unavailable")
	}
	files, captureErr := capture.files(ctx)
	report, contentErr := capture.report(ctx, static)
	return sdist, files, report, errors.Join(captureErr, contentErr)
}

func (c *sourceCapture) report(ctx context.Context, static domain.InspectionReport) (domain.InspectionReport, error) {
	native, nativeErr := c.native.Report(ctx)
	if len(native.Executions()) == 0 {
		return static, nativeErr
	}
	report, err := domain.NewCompositeInspectionReport([]domain.InspectionReport{static, native})
	return report, errors.Join(nativeErr, err)
}
