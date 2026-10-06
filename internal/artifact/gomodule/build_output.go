package gomodule

import (
	"context"
	"errors"
	projectbuild "github.com/rahoney/heliopause/internal/artifact/projectbuild"
	"github.com/rahoney/heliopause/internal/core/domain"
	"io"
)

const (
	MaxBuildOutputFiles        = projectbuild.MaxBuildOutputFiles
	MaxBuildOutputBytes        = projectbuild.MaxBuildOutputBytes
	MaxBuildOutputFileBytes    = projectbuild.MaxBuildOutputFileBytes
	MaxBuildOutputArchiveBytes = projectbuild.MaxBuildOutputArchiveBytes
)

type BuildOutputFile = projectbuild.BuildOutputFile
type CapturedBuildOutput = projectbuild.CapturedBuildOutput

func CaptureBuildOutput(ctx context.Context, intake string, inputs domain.ProjectBuildInputs, stream io.Reader) (*CapturedBuildOutput, error) {
	if inputs.Kind() != "go" {
		return nil, errors.New("go build output binding is invalid")
	}
	return projectbuild.CaptureBuildOutput(ctx, intake, inputs, stream)
}
func ReadBuildOutput(ctx context.Context, intake string, artifact domain.AcquiredArtifact, consume func(string, int64, io.Reader) error) ([]BuildOutputFile, error) {
	return projectbuild.ReadBuildOutput(ctx, intake, artifact, "go-output", consume)
}
