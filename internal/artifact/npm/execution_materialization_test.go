package npm

import (
	"archive/tar"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/rahoney/heliopause/internal/core/domain"
)

func TestMaterializedNodeLayoutDiscardsOnlyUntrustedNativeCache(t *testing.T) {
	t.Parallel()
	for _, cache := range []string{"", `{"untrusted":"not approval"}`, "not JSON or program authority"} {
		data, inventories, entries := executionLayoutFixture(t, "")
		execution, err := domain.NewExecutionContext(data)
		if err != nil {
			t.Fatal(err)
		}
		if cache != "" {
			entries = append(entries, layoutTestEntry("node_modules/.package-lock.json", tar.TypeReg, cache, ""))
		}
		native, err := ParsePNPMLayoutArchive(context.Background(), layoutTarBody(t, entries))
		if err != nil {
			t.Fatal(err)
		}
		bound, err := BindMaterializedNodeExecutionLayout(context.Background(), execution, inventories, native)
		if err != nil || !bound.MatchesExecution(execution) {
			t.Fatal(err)
		}
		for _, entry := range bound.Entries() {
			if entry.Name == "node_modules/.package-lock.json" {
				t.Fatal("native metadata entered source")
			}
		}
		if _, err := bound.ReadFile("node_modules/.package-lock.json"); err == nil {
			t.Fatal("native metadata remained readable")
		}
	}
}

func TestMaterializedNodeLayoutKeepsWholeSourceGuards(t *testing.T) {
	t.Parallel()
	for _, mutation := range []string{"foreign", "changed dependency", "cache link", "cache directory", "cache executable", "oversized cache"} {
		t.Run(mutation, func(t *testing.T) {
			data, inventories, entries := executionLayoutFixture(t, "")
			execution, err := domain.NewExecutionContext(data)
			if err != nil {
				t.Fatal(err)
			}
			cache := layoutTestEntry("node_modules/.package-lock.json", tar.TypeReg, "metadata", "")
			switch mutation {
			case "foreign":
				entries = append(entries, layoutTestEntry("node_modules/foreign", tar.TypeReg, "payload", ""))
			case "changed dependency":
				for i := range entries {
					if entries[i].header.Name == "node_modules/child/cli.js" {
						entries[i].body = strings.Replace(entries[i].body, "1", "2", 1)
					}
				}
			case "cache link":
				cache = layoutTestEntry(cache.header.Name, tar.TypeSymlink, "", "primary/package.json")
			case "cache directory":
				cache = layoutTestEntry(cache.header.Name, tar.TypeDir, "", "")
			case "cache executable":
				cache.header.Mode = 0o755
			case "oversized cache":
				cache = layoutTestEntry(cache.header.Name, tar.TypeReg, strings.Repeat("x", (4<<20)+1), "")
			}
			entries = append(entries, cache)
			native, err := ParsePNPMLayoutArchive(context.Background(), layoutTarBody(t, entries))
			if err != nil {
				t.Fatal("fixture envelope", err)
			}
			if bound, err := BindMaterializedNodeExecutionLayout(context.Background(), execution, inventories, native); err == nil || bound.Valid() {
				t.Fatal("changed source or metadata admitted")
			}
		})
	}
}

func TestMaterializedNodeLayoutPreservesCancellation(t *testing.T) {
	t.Parallel()
	data, inventories, entries := executionLayoutFixture(t, "")
	execution, err := domain.NewExecutionContext(data)
	if err != nil {
		t.Fatal(err)
	}
	native, err := ParsePNPMLayoutArchive(context.Background(), layoutTarBody(t, entries))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("fixture stopped")
	cancel(cause)
	if bound, err := BindMaterializedNodeExecutionLayout(ctx, execution, inventories, native); bound.Valid() || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
		t.Fatal("cancellation cause lost", err)
	}
}
