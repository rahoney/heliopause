package bootstrap

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
	inspectionnpm "github.com/rahoney/heliopause/internal/inspection/npm"
)

func TestBunOriginalProofRequiresIndependentOldArchive(t *testing.T) {
	for _, change := range []string{"none", "integrity", "registry", "payload", "foreign", "missing-lock", "cancel"} {
		t.Run(change, func(t *testing.T) {
			root, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			manifest := `{"name":"is-number","version":"6.0.0"}`
			var compressed bytes.Buffer
			gz := gzip.NewWriter(&compressed)
			tw := tar.NewWriter(gz)
			for _, file := range [][2]string{{"package/package.json", manifest}, {"package/index.js", "module.exports = 6;\n"}} {
				if err := tw.WriteHeader(&tar.Header{Name: file[0], Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len(file[1]))}); err != nil {
					t.Fatal(err)
				}
				if _, err := tw.Write([]byte(file[1])); err != nil {
					t.Fatal(err)
				}
			}
			if err := tw.Close(); err != nil {
				t.Fatal(err)
			}
			if err := gz.Close(); err != nil {
				t.Fatal(err)
			}
			hash := sha512.Sum512(compressed.Bytes())
			sri := "sha512-" + base64.StdEncoding.EncodeToString(hash[:])
			lock := `{"lockfileVersion":2,"configVersion":1,"workspaces":{"":{"dependencies":{"is-number":"6.0.0"}}},"packages":{"is-number":["is-number@6.0.0","",{},"` + sri + `"]}}`
			controls := []domain.ProjectControlFile{}
			for _, item := range [][2]string{{"package.json", `{"dependencies":{"is-number":"6.0.0"}}`}, {"bun.lock", lock}} {
				present := change != "missing-lock" || item[0] != "bun.lock"
				body := []byte(item[1])
				if !present {
					body = nil
				}
				control, err := domain.NewProjectControlFile(item[0], body, present)
				if err != nil {
					t.Fatal(err)
				}
				controls = append(controls, control)
			}
			calls := []string{}
			client := &http.Client{Transport: pnpmOriginalTransport(func(request *http.Request) (*http.Response, error) {
				calls = append(calls, request.URL.String())
				if change == "registry" {
					return nil, errors.New("controlled registry unavailable")
				}
				var body []byte
				contentType := "application/json"
				if request.URL.String() == "https://registry.npmjs.org/is-number/6.0.0" {
					body, _ = json.Marshal(map[string]any{"name": "is-number", "version": "6.0.0", "dist": map[string]string{"integrity": sri, "tarball": "https://registry.npmjs.org/is-number/-/is-number-6.0.0.tgz"}})
				} else if request.URL.String() == "https://registry.npmjs.org/is-number/-/is-number-6.0.0.tgz" {
					body = bytes.Clone(compressed.Bytes())
					contentType = "application/octet-stream"
					if change == "integrity" {
						body[len(body)-1] ^= 1
					}
				} else {
					t.Fatal("unexpected old source request", request.URL)
					return nil, errors.New("unexpected request")
				}
				return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {contentType}}, Body: io.NopCloser(bytes.NewReader(body)), ContentLength: int64(len(body))}, nil
			})}
			intake := filepath.Join(root, "intake")
			source, err := artifactnpm.NewResolver("https://registry.npmjs.org/", client, intake)
			if err != nil {
				t.Fatal(err)
			}
			static, err := inspectionnpm.NewInspector(intake)
			if err != nil {
				t.Fatal(err)
			}
			files := [][2]string{{"node_modules/is-number/package.json", manifest}, {"node_modules/is-number/index.js", "module.exports = 6;\n"}}
			if change == "payload" {
				files[1][1] = "module.exports = 7;\n"
			}
			if change == "foreign" {
				files = append(files, [2]string{"node_modules/foreign.js", "foreign"})
			}
			var transport bytes.Buffer
			writer := tar.NewWriter(&transport)
			write := func(header tar.Header, body string) {
				t.Helper()
				header.Uid, header.Gid = 1234, 2345
				header.Size = int64(len(body))
				if err := writer.WriteHeader(&header); err != nil {
					t.Fatal(err)
				}
				if _, err := writer.Write([]byte(body)); err != nil {
					t.Fatal(err)
				}
			}
			for _, name := range []string{"node_modules", "node_modules/is-number"} {
				write(tar.Header{Name: name, Typeflag: tar.TypeDir, Mode: 0o755}, "")
			}
			for _, file := range files {
				write(tar.Header{Name: file[0], Typeflag: tar.TypeReg, Mode: 0o644}, file[1])
			}

			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			original, err := artifactnpm.ParseBunOriginalLayout(context.Background(), transport.Bytes(), 1234, 2345)
			if err != nil {
				t.Fatal(err)
			}
			target, _ := domain.NewInstallTarget(filepath.Join(root, "project"))
			install, err := domain.NewNPMProjectInstallContext(target)
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("controlled original cancellation")
			if change == "cancel" {
				cancel(cause)
			}
			proof, err := (bunProjectBindings{source: source, static: static}).proveBunOriginal(ctx, install, controls, original)
			if change == "none" {
				if err != nil || proof.String() == "" || len(calls) != 2 {
					t.Fatal("independent old source proof failed", calls, err)
				}
			} else if err == nil || proof.String() != "" {
				t.Fatal("unproven old archive accepted", change)
			}
			if change == "cancel" && !errors.Is(err, cause) {
				t.Fatal("original cancel cause lost", err)
			}
			if (change == "cancel" || change == "missing-lock") && len(calls) != 0 {
				t.Fatal("invalid original fetched sources", calls)
			}
			if change == "integrity" && !strings.Contains(err.Error(), "integrity") {
				t.Fatal("mismatch did not fail at integrity", err)
			}
		})
	}
}
