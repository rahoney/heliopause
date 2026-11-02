package sandbox

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/rahoney/heliopause/internal/core/domain"
)

type inspectionEnvironment string

const (
	minimalEnvironment     inspectionEnvironment = "minimal-v1"
	developmentEnvironment inspectionEnvironment = "mock-development-v1"
)

type environmentFixture struct {
	name, content string
	mode          int64
	kind          byte
	canary        bool
}

// One controller-owned synthetic profile for all ecosystem backends. None of
// these values come from the Host or an artifact. Configuration fixtures are
// distinct from the existing exact honeytoken path.
type inspectionEnvironmentProfile struct {
	kind        inspectionEnvironment
	identity    string
	environment []string
	fixtures    []environmentFixture
}

func newInspectionEnvironmentProfile(kind inspectionEnvironment, session domain.SandboxSessionID) (inspectionEnvironmentProfile, error) {
	if session.String() == "" || (kind != minimalEnvironment && kind != developmentEnvironment) {
		return inspectionEnvironmentProfile{}, errors.New("inspection environment profile is invalid")
	}
	marker := sha256.Sum256([]byte(string(kind) + "\x00" + session.String()))
	invalid := "haa-invalid-" + hex.EncodeToString(marker[:16])
	profile := inspectionEnvironmentProfile{kind: kind, environment: isolatedContainerEnvironmentArguments(), fixtures: []environmentFixture{
		{name: ".haa-honeytoken", content: invalid + "\n", mode: 0400, kind: tar.TypeReg, canary: true},
	}}
	if kind == developmentEnvironment {
		for index, value := range profile.environment {
			if value == "HOME=/tmp" {
				profile.environment[index] = "HOME=/tmp/home"
			}
		}
		for _, name := range []string{"GITHUB_TOKEN", "NPM_TOKEN", "AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY"} {
			profile.environment = append(profile.environment, "--env", name+"="+invalid)
		}
		profile.fixtures = append(profile.fixtures,
			environmentFixture{name: "home", mode: 0700, kind: tar.TypeDir},
			environmentFixture{name: "home/.aws", mode: 0700, kind: tar.TypeDir},
			environmentFixture{name: "home/.ssh", mode: 0700, kind: tar.TypeDir},
			environmentFixture{name: "project", mode: 0700, kind: tar.TypeDir},
			environmentFixture{name: "project/.env", content: "APP_ENV=development\nAPI_TOKEN=" + invalid + "\n", mode: 0600, kind: tar.TypeReg},
			environmentFixture{name: "home/.aws/credentials", content: "[default]\naws_access_key_id=" + invalid + "\naws_secret_access_key=" + invalid + "\n", mode: 0600, kind: tar.TypeReg},
			environmentFixture{name: "home/.npmrc", content: "//registry.npmjs.org/:_authToken=" + invalid + "\n", mode: 0600, kind: tar.TypeReg},
			environmentFixture{name: "home/.pypirc", content: "[pypi]\nusername=__token__\npassword=" + invalid + "\n", mode: 0600, kind: tar.TypeReg},
			environmentFixture{name: "home/.ssh/id_ed25519", content: "HAA INVALID SYNTHETIC KEY " + invalid + "\n", mode: 0600, kind: tar.TypeReg},
			environmentFixture{name: "project/README.md", content: "Synthetic inspection workspace.\n", mode: 0600, kind: tar.TypeReg},
		)
	}
	hash := sha256.New()
	_, _ = fmt.Fprintf(hash, "%s\x00%s\n", kind, session.String())
	for _, entry := range profile.environment {
		_, _ = fmt.Fprintf(hash, "%d:%s\n", len(entry), entry)
	}
	for _, file := range profile.fixtures {
		_, _ = fmt.Fprintf(hash, "%s\x00%o:%d:%t:%d:%s\n", file.name, file.mode, file.kind, file.canary, len(file.content), file.content)
	}
	profile.identity = hex.EncodeToString(hash.Sum(nil))
	return profile, nil
}

func (p inspectionEnvironmentProfile) payload() ([]byte, error) {
	if p.identity == "" || len(p.fixtures) == 0 {
		return nil, errors.New("inspection environment payload is unavailable")
	}
	var body bytes.Buffer
	archive := tar.NewWriter(&body)
	for _, file := range p.fixtures {
		if err := archive.WriteHeader(&tar.Header{Name: file.name, Mode: file.mode, Typeflag: file.kind, Size: int64(len(file.content)), Uid: 1000, Gid: 1000}); err != nil {
			return nil, err
		}
		if _, err := archive.Write([]byte(file.content)); err != nil {
			return nil, err
		}
	}
	if err := archive.Close(); err != nil {
		return nil, err
	}
	return body.Bytes(), nil
}

func (p inspectionEnvironmentProfile) verifyPlacement(state environmentState) error {
	if p.identity == "" || state.digest == "" || len(p.fixtures) == 0 {
		return errEnvironmentState
	}
	for _, expected := range p.fixtures {
		actual, present := state.files[expected.name]
		if !present || actual.kind != expected.kind || actual.mode != expected.mode || actual.uid != 1000 || actual.gid != 1000 || actual.size != int64(len(expected.content)) {
			return errEnvironmentState
		}
		if expected.kind == tar.TypeReg {
			digest := sha256.Sum256([]byte(expected.content))
			if actual.digest != hex.EncodeToString(digest[:]) {
				return errEnvironmentState
			}
		}
	}
	return nil
}
