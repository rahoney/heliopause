package promotion

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	artifactgomodule "github.com/rahoney/heliopause/internal/artifact/gomodule"
	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestGoProjectPromotionPublishesPrivateGetTransaction(t *testing.T) {
	root := writeManagedGoProject(t)
	reference, _ := artifactgomodule.ParseReference("example.com/new@v1.2.3")
	target, _ := domain.NewInstallTarget(root)
	installContext, _ := domain.NewInstallContext(target)
	promotion, err := NewGoProjectPromotion(goProjectRunnerFixture{})
	if err != nil {
		t.Fatal(err)
	}
	if err := promotion.PromoteGet(context.Background(), reference, installContext); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil || string(body) == "module example.com/app\n\ngo 1.25\n" {
		t.Fatalf("promoted go.mod = %q, error = %v", body, err)
	}
}

type goProjectRunnerFixture struct{}

func (goProjectRunnerFixture) RunGo(_ context.Context, directory string, environment []string, arguments ...string) ([]byte, error) {
	cache := ""
	for _, entry := range environment {
		if len(entry) > len("GOMODCACHE=") && entry[:len("GOMODCACHE=")] == "GOMODCACHE=" {
			cache = entry[len("GOMODCACHE="):]
		}
	}
	if err := artifactgomodule.ValidateResolverEnvironmentForCache(environment, cache); err != nil {
		return nil, err
	}
	if len(arguments) != 2 || arguments[0] != "get" {
		return nil, os.ErrInvalid
	}
	return nil, os.WriteFile(filepath.Join(directory, "go.mod"), []byte("module example.com/app\n\ngo 1.25\n\nrequire example.com/new v1.2.3\n"), 0o600)
}
