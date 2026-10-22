package promotion

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"golang.org/x/sys/unix"
)

func uvSyncSourceFixture(t *testing.T, body string) (*pypiVenvTransaction, string, string) {
	t.Helper()
	_, plan := makePypiVenvFixture(t)
	tx, _ := heldCompleteTestTransaction(t, plan)
	base := realPromotionRoot(t)
	if err := os.Mkdir(filepath.Join(base, "inputs"), 0o700); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(base, "inputs", "requirements.txt")
	if err := os.WriteFile(source, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return tx, source, realPromotionRoot(t)
}

func TestUVSyncSourceFreezesIntentAndSurvivesCompletePublication(t *testing.T) {
	for _, example := range []struct {
		name, body string
		format     artifactpypi.UVSyncSourceFormat
		allow      bool
	}{
		{"requirements", "alpha==1.0\n", artifactpypi.UVSyncRequirements, false},
		{"explicit-requirements-empty", "# intentionally empty\n", artifactpypi.UVSyncRequirements, true},
		{"explicit-pylock-empty", "lock-version = \"1.0\"\ncreated-by = \"uv\"\nrequires-python = \">=3.14.7\"\npackages = []\n", artifactpypi.UVSyncPylock, false},
	} {
		t.Run(example.name, func(t *testing.T) {
			tx, source, trusted := uvSyncSourceFixture(t, example.body)
			input, err := freezeUVSyncSource(context.Background(), tx, source, example.format, example.allow, []string{trusted})
			if err != nil || input.ShouldSkip() || input.SourceFormat() != example.format || input.AllowEmptyRequirements() != example.allow {
				t.Fatal("positive input intent changed", err)
			}
			body := input.Body()
			body[0] = 'x'
			if string(input.Body()) != example.body {
				t.Fatal("caller changed frozen source")
			}
			var destinations []pypiDestination
			if example.name == "requirements" {
				destinations = venvDestinations(t, map[string]string{"site/alpha.py": "synthetic approved output"}, "alpha")
			}
			if err := tx.commitComplete(destinations, len(destinations) == 0); err != nil {
				t.Fatal(err)
			}
			if body, err := os.ReadFile(source); err != nil || string(body) != example.body {
				t.Fatal("publication changed external source", err)
			}
		})
	}
}

func TestUVSyncSourceDriftRefusesMutation(t *testing.T) {
	for _, change := range []string{"content", "inode", "mode", "parent"} {
		t.Run(change, func(t *testing.T) {
			tx, source, trusted := uvSyncSourceFixture(t, "alpha==1.0\n")
			if _, err := freezeUVSyncSource(context.Background(), tx, source, artifactpypi.UVSyncRequirements, false, []string{trusted}); err != nil {
				t.Fatal(err)
			}
			var err error
			switch change {
			case "content":
				err = os.WriteFile(source, []byte("alpha==2.0\n"), 0o600)
			case "inode":
				err = os.Rename(source, source+".old")
				if err == nil {
					err = os.WriteFile(source, []byte("alpha==1.0\n"), 0o600)
				}
			case "mode":
				err = os.Chmod(source, 0o400)
			case "parent":
				parent := filepath.Dir(source)
				err = os.Rename(parent, parent+"-old")
				if err == nil {
					err = os.Mkdir(parent, 0o700)
				}
				if err == nil {
					err = os.WriteFile(source, []byte("alpha==1.0\n"), 0o600)
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			if err := tx.checkBefore(); err == nil {
				t.Fatal("source witness accepted drift")
			}
			if err := tx.commitComplete(nil, true); err == nil || tx.backup != "" {
				t.Fatal("source drift allowed mutation", err)
			}
		})
	}
}

func TestUVSyncSourceRejectsAliasedAndUnsupportedInput(t *testing.T) {
	for _, invalid := range []string{"file-symlink", "parent-symlink", "hardlink", "target", "trusted", "trusted-alias", "noncanonical", "directory", "fifo", "oversize", "invalid", "blank", "cancelled", "missing-trusted"} {
		t.Run(invalid, func(t *testing.T) {
			tx, source, trusted := uvSyncSourceFixture(t, "alpha==1.0\n")
			ctx := context.Background()
			roots := []string{trusted}
			var err error
			switch invalid {
			case "file-symlink":
				err = os.Symlink(source, source+"-link")
				source += "-link"
			case "parent-symlink":
				link := filepath.Dir(source) + "-link"
				err = os.Symlink(filepath.Dir(source), link)
				source = filepath.Join(link, filepath.Base(source))
			case "hardlink":
				err = os.Link(source, source+"-link")
			case "target":
				source = filepath.Join(tx.plan.root, "pyvenv.cfg")
			case "trusted":
				source = filepath.Join(trusted, "source")
				err = os.WriteFile(source, []byte("alpha==1.0\n"), 0o600)
			case "trusted-alias":
				roots = []string{filepath.Join(trusted, "alias")}
				err = os.Symlink(filepath.Dir(source), roots[0])
			case "noncanonical":
				source = filepath.Dir(source) + "/../inputs/requirements.txt"
			case "directory":
				source = filepath.Dir(source)
			case "fifo":
				source += "-pipe"
				err = unix.Mkfifo(source, 0o600)
			case "oversize":
				err = os.WriteFile(source, []byte(strings.Repeat("#", artifactpypi.UVSyncSourceByteLimit+1)), 0o600)
			case "invalid":
				err = os.WriteFile(source, []byte("alpha>=1.0\n"), 0o600)
			case "blank":
				err = os.WriteFile(source, []byte("# blank\n"), 0o600)
			case "cancelled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			case "missing-trusted":
				roots = nil
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err := freezeUVSyncSource(ctx, tx, source, artifactpypi.UVSyncRequirements, false, roots); err == nil {
				t.Fatal("unsafe source accepted")
			} else if invalid == "cancelled" && !errors.Is(err, context.Canceled) {
				t.Fatal("cancellation cause lost", err)
			}
			if tx.backup != "" {
				t.Fatal("refused input started publication")
			}
		})
	}
}
