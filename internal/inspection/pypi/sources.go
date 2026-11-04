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
	name, language string
	body           bytes.Buffer
}

type sourceCapture struct {
	members []sourceMember
	bytes   uint64
	limited bool
}

func (c *sourceCapture) member(name string, size uint64) io.Writer {
	language := content.SourceLanguage(name)
	if language == "" || c.limited {
		return nil
	}
	if size > ports.SourceAnalysisFileLimit || size > ports.SourceAnalysisTotalLimit-c.bytes || len(c.members) >= ports.SourceAnalysisTargetLimit {
		c.limited = true
		c.members = nil
		return nil
	}
	c.bytes += size
	c.members = append(c.members, sourceMember{name: name, language: language})
	return &c.members[len(c.members)-1].body
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
