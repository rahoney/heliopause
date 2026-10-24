// Package releasebootstrap prepares ordinary-user, provenance-bound first
// installation bytes. It never installs, runs candidates, or grants Policy.
package releasebootstrap

type Preparer struct{}

func New() *Preparer { return &Preparer{} }
