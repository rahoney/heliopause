package promotion

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVenvFinalReconcilePreservesUncertainFiles(t *testing.T) {
	for _, retained := range []bool{false, true} {
		for _, change := range []string{"same-content-replacement", "same-length-content-change", "mode-change"} {
			t.Run(fmt.Sprintf("retained=%t/%s", retained, change), func(t *testing.T) {
				root, plan := makePypiVenvFixture(t)
				target := filepath.Join(plan.site, "new.py")
				if retained {
					runVenvFixture(t, plan, map[string]string{"site/retained.py": "keep"}, "retained")
					target = filepath.Join(plan.site, "retained.py")
				}
				tx, err := beginPyPIVenvTransaction(plan)
				if err != nil {
					t.Fatal(err)
				}
				defer tx.close()
				var changed bool
				tx.fault = func(phase string) error {
					if phase != "reconcile" {
						return nil
					}
					changed = true
					body, err := os.ReadFile(target)
					if err != nil {
						return err
					}
					switch change {
					case "same-content-replacement":
						if err := os.Rename(target, target+".outside"); err != nil {
							return err
						}
						return os.WriteFile(target, body, 0o600)
					case "same-length-content-change":
						body[0] ^= 1
						return os.WriteFile(target, body, 0o600)
					case "mode-change":
						return os.Chmod(target, 0o640)
					}
					return errors.New("unknown file identity change")
				}
				err = tx.commit(venvDestinations(t, map[string]string{"site/new.py": "new"}, "new"))
				if !changed || err == nil || !tx.recovery {
					t.Fatalf("change=%t, error=%v, recovery=%t", changed, err, tx.recovery)
				}
				if !strings.Contains(err.Error(), "drift") {
					t.Fatalf("wrong failure: %v", err)
				}
				if _, err := os.Lstat(target); err != nil {
					t.Fatal("replacement/output was removed despite uncertain identity", err)
				}
				ledgers, err := filepath.Glob(filepath.Join(root, ".heliopause-pypi-commit-*", "ledger.json"))
				if err != nil || len(ledgers) != 1 {
					t.Fatal("recovery journal missing", err, len(ledgers))
				}
			})
		}
	}
}
