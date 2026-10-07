package gomodule

import "github.com/rahoney/heliopause/internal/inspection/projectbuild"

type BuildSandbox = projectbuild.BuildSandbox
type BuildInspector = projectbuild.Inspector

func NewBuildInspector(backend BuildSandbox) (*BuildInspector, error) {
	return projectbuild.NewInspector(backend, "go")
}
