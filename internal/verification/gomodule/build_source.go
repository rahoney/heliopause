package gomodule

import "github.com/rahoney/heliopause/internal/verification/projectbuild"

type BuildSourceReader = projectbuild.BuildSourceReader
type BuildSourceVerifier = projectbuild.SourceVerifier

func NewBuildSourceVerifier(reader BuildSourceReader) (*BuildSourceVerifier, error) {
	return projectbuild.NewSourceVerifier(reader, "go")
}
