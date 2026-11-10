package promotion

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
)

// freezePythonRequirements shares the held venv/source witnesses with sync,
// while retaining the distinct additive request and every included file.
func freezePythonRequirements(ctx context.Context, transaction *pypiVenvTransaction, sources, constraints, positional, trustedRoots []string) (artifactpypi.RequirementsInput, error) {
	invalid := errors.New("requirements sources must remain within their original input root")
	if ctx == nil || transaction == nil || transaction.before == nil || len(trustedRoots) == 0 {
		return artifactpypi.RequirementsInput{}, invalid
	}
	if err := ctx.Err(); err != nil {
		return artifactpypi.RequirementsInput{}, err
	}
	if err := transaction.checkBefore(); err != nil {
		return artifactpypi.RequirementsInput{}, err
	}
	initial := append(append([]string(nil), sources...), constraints...)
	if len(initial) > artifactpypi.MaximumRequirementsFiles {
		return artifactpypi.RequirementsInput{}, invalid
	}
	var roots []artifactpypi.RequirementsSource
	files := map[string][]byte{}
	if len(initial) > 0 {
		base := filepath.Dir(initial[0])
		relative := func(name string) (string, error) {
			if !filepath.IsAbs(name) || filepath.Clean(name) != name {
				return "", invalid
			}
			rel, err := filepath.Rel(base, name)
			if err != nil || !artifactpypi.ValidRequirementsSourceName(filepath.ToSlash(rel)) {
				return "", invalid
			}
			return filepath.ToSlash(rel), nil
		}
		pending := []string{}
		scheduled := map[string]bool{}
		for index, name := range initial {
			rel, err := relative(name)
			if err != nil {
				return artifactpypi.RequirementsInput{}, err
			}
			roots = append(roots, artifactpypi.RequirementsSource{Name: rel, Constraint: index >= len(sources)})
			if !scheduled[name] {
				pending = append(pending, name)
				scheduled[name] = true
			}
		}
		total := 0
		for index := 0; index < len(pending); index++ {
			if len(files) >= artifactpypi.MaximumRequirementsFiles {
				return artifactpypi.RequirementsInput{}, invalid
			}
			name := pending[index]
			rel, err := relative(name)
			if err != nil {
				return artifactpypi.RequirementsInput{}, err
			}
			body, snapshot, err := freezePythonSourceFile(ctx, transaction, name, trustedRoots, int64(artifactpypi.RequirementsInputByteLimit-total))
			if err != nil {
				return artifactpypi.RequirementsInput{}, err
			}
			document, err := artifactpypi.ParseRequirementsFile(body)
			if err != nil {
				return artifactpypi.RequirementsInput{}, fmt.Errorf("requirements %q: %w", rel, err)
			}
			total += len(body)
			files[rel] = body
			transaction.before[name] = &snapshot
			for _, line := range document.Lines() {
				if line.Include == "" {
					continue
				}
				child := filepath.Join(filepath.Dir(name), filepath.FromSlash(line.Include))
				if _, err := relative(child); err != nil {
					return artifactpypi.RequirementsInput{}, err
				}
				if !scheduled[child] {
					if len(scheduled) >= artifactpypi.MaximumRequirementsFiles {
						return artifactpypi.RequirementsInput{}, invalid
					}
					scheduled[child] = true
					pending = append(pending, child)
				}
			}
		}
	}
	input, err := artifactpypi.NewRequirementsInput(roots, files, positional)
	if err != nil {
		return artifactpypi.RequirementsInput{}, err
	}
	if err := transaction.checkBefore(); err != nil {
		return artifactpypi.RequirementsInput{}, err
	}
	return input, nil
}
