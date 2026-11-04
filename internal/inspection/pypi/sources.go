package pypi

import (
	"bytes"
	"context"
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
	body           bytes.Buffer
}

type sourceCapture struct {
	members []*sourceMember
	bytes   uint64
	limited bool
}

func (c *sourceCapture) member(name string, size uint64) io.Writer {
	language := content.SourceLanguage(name)
	if c.limited || language == "" && size == 0 {
		return nil
	}
	member := &sourceMember{owner: c, name: name, size: size}
	if language != "" {
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
	if m.size > ports.SourceAnalysisFileLimit || m.size > ports.SourceAnalysisTotalLimit-c.bytes || len(c.members) >= ports.SourceAnalysisTargetLimit {
		c.limited, c.members, m.language = true, nil, ""
		return
	}
	c.bytes += m.size
	c.members = append(c.members, m)
}

func (m *sourceMember) Write(data []byte) (int, error) {
	written := len(data)
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
		m.selectLanguage(content.SourceLanguage(m.name, m.prefix))
		if m.language != "" {
			_, _ = m.body.Write(m.prefix) // bytes.Buffer.Write never fails.
		}
		m.prefix = nil
	}
	if m.language != "" {
		_, _ = m.body.Write(data)
	}
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
