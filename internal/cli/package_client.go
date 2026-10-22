package cli

import (
	"errors"
	"strings"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/spf13/cobra"
)

// PackageClientID selects a compiled client binding, independently of registry
// SourceID. Only implemented command families belong in this registry.
type PackageClientID string

const selectedPackageClientAnnotation = "helox.selected-package-client"

const (
	PipPackageClient   PackageClientID = "pip"
	UVPipPackageClient PackageClientID = "uv-pip"
)

type packageClientParent struct{ en, ko string }
type packageClientDescriptor struct {
	id                                           PackageClientID
	path                                         []string
	parents                                      []packageClientParent
	route, use, shortEN, shortKO, longEN, longKO string
	namedSources                                 bool
	validate                                     func(*cobra.Command, []string) error
	validateRaw                                  func([]string) error
	parse                                        func(string, domain.SourceID) (domain.ArtifactReference, error)
	flags                                        func(*cobra.Command)
}

const uvPipInstallEnglish = "Inspect a public PyPI wheel set selected by UV and install only its approved files.\n\nThis initial path requires an empty existing virtual environment for the pinned Python version. Use one package name or exact name==version (name@version is also accepted). Existing distributions require installed-aware selection and are currently refused. Other indexes, builds, upgrades and sync are not available in this path."
const uvPipInstallKorean = "UV가 선택한 공개 PyPI wheel과 전체 의존성을 검사하고 허용된 파일만 설치합니다.\n\n현재 경로는 고정된 Python 버전의 비어 있는 기존 가상환경을 요구합니다. 패키지 이름 또는 정확한 name==version을 지정하세요(name@version도 가능). 이미 설치된 패키지가 있는 환경은 기존 상태를 반영한 선택 기능이 필요하므로 현재 거부합니다. 다른 index, build, 업데이트와 sync는 이 경로에서 아직 제공하지 않습니다."

func packageClientDescriptors() []packageClientDescriptor {
	return []packageClientDescriptor{
		{id: PipPackageClient, path: []string{"pip", "install"}, parents: []packageClientParent{{"Inspect and install PyPI distributions", "PyPI 배포 파일 검사·설치"}}, route: "pip", use: "install <project>[@<version>]", shortEN: "Install a PyPI distribution into the active virtual environment", shortKO: "Python 배포 파일과 의존성을 검사한 뒤 설치·업데이트(--upgrade/-U)합니다", longEN: pypiUpgradeDescriptionEnglish, longKO: pypiUpgradeDescriptionKorean, namedSources: true, validate: validatePyPIInstallArgs, parse: artifactpypi.ParseReferenceForSource, flags: configurePipInstallFlags},
		{id: UVPipPackageClient, path: []string{"uv", "pip", "install"}, parents: []packageClientParent{{"Quarantine Python packages with UV", "UV로 Python 패키지를 검역합니다"}, {"Inspect and install public PyPI distributions with UV", "UV pip으로 공개 PyPI 파일을 검사·설치합니다"}}, route: "pypi", use: "install <project>[==<version>]", shortEN: "Quarantine and install a public PyPI wheel set selected by UV", shortKO: "UV가 선택한 공개 PyPI wheel과 의존성을 검역한 뒤 설치합니다", longEN: uvPipInstallEnglish, longKO: uvPipInstallKorean, validate: validateUVPipInstallArgs, validateRaw: validateUVPipRawArgs, parse: parseUVPipInstallReference},
	}
}

func configurePipInstallFlags(command *cobra.Command) {
	command.Flags().BoolP("upgrade", "U", false, "Inspect the complete selected closure before upgrading one distribution (only-if-needed)")
	command.Flags().Bool("adopt", false, "Enroll supported native wheel files after independent ownership verification and full inspection")
	command.Flags().String("source", "pypi", pythonSourceHelp())
	command.Flags().String("inspection-prerequisites", "", "Explicit pinned JSON for one inspection-only non-root leaf prerequisite; not installed into the target")
}

func packageClientByID(id PackageClientID) (packageClientDescriptor, bool) {
	for _, descriptor := range packageClientDescriptors() {
		if descriptor.id == id {
			return descriptor, true
		}
	}
	return packageClientDescriptor{}, false
}

func relativeCommandPath(command *cobra.Command) string {
	var names []string
	for current := command; current.Parent() != nil; current = current.Parent() {
		names = append([]string{current.Name()}, names...)
	}
	return strings.Join(names, " ")
}

func packageClientForCommand(command *cobra.Command) (packageClientDescriptor, bool) {
	path := relativeCommandPath(command)
	for _, descriptor := range packageClientDescriptors() {
		if path == strings.Join(descriptor.path, " ") {
			return descriptor, true
		}
	}
	return packageClientDescriptor{}, false
}

func initializePackageClientCommands(root *cobra.Command) {
	for _, descriptor := range packageClientDescriptors() {
		parent := root
		for index, segment := range descriptor.path[:len(descriptor.path)-1] {
			if descriptor.id == PipPackageClient && index == 0 {
				parent = ensurePipCommand(root)
				continue
			}
			var child *cobra.Command
			for _, candidate := range parent.Commands() {
				if candidate.Name() == segment {
					child = candidate
					break
				}
			}
			if child == nil {
				child = &cobra.Command{Use: segment, Short: descriptor.parents[index].en}
				parent.AddCommand(child)
			}
			parent = child
		}
		command := newStaticLeaf(descriptor.use, descriptor.shortEN, true)
		command.Long, command.Args = descriptor.longEN, descriptor.validate
		if descriptor.flags != nil {
			descriptor.flags(command)
		}
		parent.AddCommand(command)
	}
}

// PackageClientInstallCommand obtains the actual registered command from the
// same descriptor used by preflight, help and binding.
func PackageClientInstallCommand(root *cobra.Command, id PackageClientID) (*cobra.Command, error) {
	descriptor, ok := packageClientByID(id)
	if root == nil || !ok {
		return nil, errors.New("unsupported package client command")
	}
	command, remaining, err := root.Find(descriptor.path)
	if err != nil || len(remaining) != 0 || relativeCommandPath(command) != strings.Join(descriptor.path, " ") {
		return nil, errors.New("package client install command is not registered")
	}
	return command, nil
}

func SelectedPackageClient(root *cobra.Command) PackageClientID {
	if root == nil {
		return ""
	}
	return PackageClientID(root.Annotations[selectedPackageClientAnnotation])
}

func localizePackageClientCommand(command *cobra.Command, p *presentation) {
	path := relativeCommandPath(command)
	for _, descriptor := range packageClientDescriptors() {
		if path == strings.Join(descriptor.path, " ") {
			command.Short, command.Long = p.text(descriptor.shortEN, descriptor.shortKO), p.text(descriptor.longEN, descriptor.longKO)
		}
		for index, parent := range descriptor.parents {
			if path == strings.Join(descriptor.path[:index+1], " ") {
				command.Short = p.text(parent.en, parent.ko)
			}
		}
	}
}

func parseUVPipInstallReference(input string, source domain.SourceID) (domain.ArtifactReference, error) {
	if source != artifactpypi.PublicPyPIProfile().Source() {
		return domain.ArtifactReference{}, errors.New("UV pip requires public PyPI")
	}
	if strings.Contains(input, "==") {
		if strings.Count(input, "==") != 1 || strings.Contains(input, "@") {
			return domain.ArtifactReference{}, errors.New("UV exact version is ambiguous")
		}
		name, version, _ := strings.Cut(input, "==")
		input = name + "@" + version
	}
	reference, err := artifactpypi.ParseReference(input)
	if err != nil {
		return domain.ArtifactReference{}, err
	}
	version, exact, err := artifactpypi.RequestedVersion(reference)
	if err != nil || exact && !artifactpypi.IsFinalVersion(version) {
		return domain.ArtifactReference{}, errors.New("UV pip requires a supported final version")
	}
	return reference, nil
}

func validateUVPipInstallArgs(command *cobra.Command, args []string) error {
	if err := cobra.ExactArgs(1)(command, args); err != nil {
		return err
	}
	if _, err := parseUVPipInstallReference(args[0], artifactpypi.PublicPyPIProfile().Source()); err != nil {
		return err
	}
	target, err := command.Flags().GetString("target")
	if err != nil {
		return err
	}
	value, err := activePythonVenvForCommand(target, "uv pip install")
	if err != nil {
		return err
	}
	if strings.Contains(value.String(), ",") {
		return errors.New("UV pip target cannot contain a comma")
	}
	return nil
}

func validateUVPipRawArgs(args []string) error {
	count := 0
	for _, value := range args {
		if value == "--target" || strings.HasPrefix(value, "--target=") {
			count++
		}
	}
	if count > 1 {
		return errors.New("UV pip requires one unambiguous target option")
	}
	return nil
}

func AddUVPipInstall(root *cobra.Command, installer Installer) error {
	if installer == nil {
		return errors.New("UV pip install requires a prepared use case")
	}
	return addPackageClientInstall(root, UVPipPackageClient, map[string]Installer{"pypi": installer}, nil)
}
