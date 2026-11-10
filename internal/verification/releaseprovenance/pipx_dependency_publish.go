package releaseprovenance

import (
	"context"
	"errors"
	"regexp"

	"golang.org/x/mod/semver"
)

var pipxDependencyVersion = regexp.MustCompile(`^[0-9]+\.[0-9]+(?:\.[0-9]+)?$`)

// PipxDependencyPublishIdentity requests one of pipx's reviewed pure-Python
// dependency publishers. It is not a general PyPI publisher configuration.
type PipxDependencyPublishIdentity struct {
	Project      string `json:"project"`
	Version      string `json:"version"`
	SourceCommit string `json:"source_commit"`
	WorkflowRun  string `json:"workflow_run"`
	WheelSHA256  string `json:"wheel_sha256"`
}

func (i PipxDependencyPublishIdentity) valid() bool {
	_, ok := pipxDependencyPolicy(i.Project, i.Version)
	return ok && len(i.Version) <= 64 && pipxDependencyVersion.MatchString(i.Version) && semver.IsValid("v"+i.Version) && hexCommit.MatchString(i.SourceCommit) && runNumber.MatchString(i.WorkflowRun) && hexSHA256.MatchString(i.WheelSHA256)
}

func (i PipxDependencyPublishIdentity) WheelFilename() string {
	if !i.valid() {
		return ""
	}
	return i.Project + "-" + i.Version + "-py3-none-any.whl"
}

func (i *PipxDependencyPublishIdentity) UnmarshalJSON(body []byte) error {
	if i == nil {
		return errPipxJSON
	}
	*i = PipxDependencyPublishIdentity{}
	if err := boundPipxJSON(body, 4096); err != nil {
		return err
	}
	o, err := pipxObject(body, "project", "version", "source_commit", "workflow_run", "wheel_sha256")
	if err != nil {
		return err
	}
	v := PipxDependencyPublishIdentity{pipxString(o["project"]), pipxString(o["version"]), pipxString(o["source_commit"]), pipxString(o["workflow_run"]), pipxString(o["wheel_sha256"])}
	if !v.valid() {
		return errPipxJSON
	}
	*i = v
	return nil
}

type PipxDependencyPublishReceipt struct {
	identity                     PipxDependencyPublishIdentity
	provenanceSHA256, invocation string
}

func (r PipxDependencyPublishReceipt) Identity() PipxDependencyPublishIdentity { return r.identity }
func (r PipxDependencyPublishReceipt) ProvenanceSHA256() string                { return r.provenanceSHA256 }
func (r PipxDependencyPublishReceipt) Invocation() string                      { return r.invocation }

// VerifyPipxDependencyPublish uses fixed publisher policies, including the
// upstream packaging main/workflow_dispatch release rather than a fabricated tag.
// Userpath is intentionally unsupported here: a missing proof is not a signature.
func VerifyPipxDependencyPublish(ctx context.Context, body []byte, expected PipxDependencyPublishIdentity) (PipxDependencyPublishReceipt, error) {
	if !expected.valid() {
		return PipxDependencyPublishReceipt{}, errors.New("pipx dependency publish identity is unsupported")
	}
	policy, _ := pipxDependencyPolicy(expected.Project, expected.Version)
	proof, err := verifyPyPIPublish(ctx, body, pypiPublishRequest{expected.SourceCommit, expected.WorkflowRun, expected.WheelSHA256, expected.WheelFilename()}, policy)
	if err != nil {
		return PipxDependencyPublishReceipt{}, err
	}
	return PipxDependencyPublishReceipt{expected, proof.provenanceSHA256, proof.invocation}, nil
}

func pipxDependencyPolicy(project, version string) (pypiPublisherPolicy, bool) {
	switch project {
	case "argcomplete":
		return pypiPublisherPolicy{"https://github.com/kislyuk/argcomplete", "release.yml", "refs/tags/v" + version, "6762625", "kislyuk", "862013", "push", "release"}, true
	case "click":
		return pypiPublisherPolicy{"https://github.com/pallets/click", "publish.yaml", "refs/tags/" + version, "19103692", "pallets", "16748505", "push", "publish"}, true
	case "filelock":
		return pypiPublisherPolicy{"https://github.com/tox-dev/filelock", "release.yaml", "refs/tags/" + version, "17098256", "tox-dev", "20345659", "push", "release"}, true
	case "packaging":
		return pypiPublisherPolicy{"https://github.com/pypa/packaging", "publish.yml", "refs/heads/main", "19837758", "pypa", "647025", "workflow_dispatch", "pypi"}, true
	case "platformdirs":
		return pypiPublisherPolicy{"https://github.com/tox-dev/platformdirs", "release.yaml", "refs/tags/" + version, "367032142", "tox-dev", "20345659", "push", "release"}, true
	case "pytest":
		return pypiPublisherPolicy{"https://github.com/pytest-dev/pytest", "deploy.yml", "refs/heads/release-" + version, "37489525", "pytest-dev", "8897583", "workflow_dispatch", "deploy"}, true
	case "iniconfig":
		return pypiPublisherPolicy{"https://github.com/pytest-dev/iniconfig", "test.yml", "refs/tags/v" + version, "66012563", "pytest-dev", "8897583", "push", "pypi-upload"}, true
	case "pluggy":
		return pypiPublisherPolicy{"https://github.com/pytest-dev/pluggy", "deploy.yml", "refs/tags/" + version, "34739117", "pytest-dev", "8897583", "push", ""}, true
	default:
		return pypiPublisherPolicy{}, false
	}
}
