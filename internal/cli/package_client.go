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
const selectedPackageClientOperationAnnotation = "helox.selected-package-client-operation"

const (
	PipPackageClient       PackageClientID = "pip"
	UVPipPackageClient     PackageClientID = "uv-pip"
	UVProjectPackageClient PackageClientID = "uv-project"
	PNPMPackageClient      PackageClientID = "pnpm"
	BunPackageClient       PackageClientID = "bun"
)

type packageUpgradeMode uint8

const (
	packageNoUpgrade packageUpgradeMode = iota
	packageUpgradeRoot
	packageUpgradeClosure
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
	upgrade                                      func(*cobra.Command, []string) (packageUpgradeMode, error)
	examples                                     []string
	flagText                                     map[string]packageClientParent
}

const uvPipInstallEnglish = "Inspect the complete public PyPI wheel set selected by UV before installing into an existing virtual environment for the pinned Python version.\n\nUse one package name or exact name==version (name@version is also accepted). Ordinary install retains compatible installed versions. --upgrade/-U updates the selected root and its dependencies; --upgrade-package <root> updates only that requested root unless a dependency change is needed. These two options cannot be combined. Existing selected files require verified HAA ownership or explicit --adopt with an update option. Native default public UV wheels are independently compared with the authentic old wheel before enrollment. Native generated scripts, compiled caches, cross-scheme and local/direct/VCS/build installs are unsupported. Other indexes and builds are not available. Use uv pip sync for complete-file synchronization. Every selected wheel is freshly inspected, including unchanged versions."
const uvPipInstallKorean = "UV가 선택한 공개 PyPI wheel과 전체 의존성을 검사하고 고정된 Python 버전의 기존 가상환경에 설치합니다.\n\n패키지 이름 또는 정확한 name==version을 지정하세요(name@version도 가능). 일반 설치는 호환되는 기존 버전을 유지합니다. --upgrade/-U는 요청 패키지와 선택된 의존성을 갱신하고, --upgrade-package <root>는 필요할 때만 의존성을 변경하며 요청 패키지만 갱신합니다. 두 옵션을 함께 사용할 수 없고, <root>에는 요청한 패키지 이름만 지정할 수 있습니다. 선택된 기존 파일은 검증된 HAA 소유 기록이 필요합니다. 일반 UV 설치를 인계받으려면 업데이트 옵션과 --adopt를 함께 사용하세요. 기본 공개 wheel의 기존 파일을 공식 이전 wheel과 독립 대조한 뒤 소유권을 등록합니다. 일반 설치의 생성 스크립트·compiled cache·다른 scheme·local/direct/VCS/build, 다른 index는 제공하지 않습니다. 같은 버전도 선택한 wheel 전체를 새로 검사합니다."

func packageClientDescriptors() []packageClientDescriptor {
	return []packageClientDescriptor{
		{id: UVProjectPackageClient, path: []string{"uv", "lock"}, parents: []packageClientParent{{"Quarantine Python packages with UV", "UV로 Python 패키지를 검역합니다"}}, route: "pypi", use: "lock --upgrade", shortEN: "Quarantine the selected project before upgrading only its lock file", shortKO: "선택된 프로젝트 전체를 검역한 뒤 잠금파일만 갱신합니다", longEN: uvProjectLockEnglish, longKO: uvProjectLockKorean, validate: validateUVProjectLockArgs, validateRaw: validateUVProjectLockRawArgs, flags: configureUVProjectLockFlags, flagText: uvProjectLockFlagText, examples: []string{"helox uv lock --upgrade", "helox uv lock --upgrade --project /absolute/path/project"}},
		{id: UVProjectPackageClient, path: []string{"uv", "add"}, parents: []packageClientParent{{"Quarantine Python packages with UV", "UV로 Python 패키지를 검역합니다"}}, route: "pypi", use: "add <name-or-requirement>", shortEN: "Quarantine and add a public dependency to a virtual project", shortKO: "공개 의존성 전체를 검역한 뒤 virtual project에 추가합니다", longEN: uvProjectEnglish, longKO: uvProjectKorean, validate: validateUVProjectArgs, validateRaw: validateUVProjectRawArgs, flags: configureUVProjectFlags, flagText: uvProjectFlagText, examples: []string{"helox uv add packaging==25.0", "helox uv add idna --project /absolute/path/project --target /absolute/path/venv"}},
		{id: UVProjectPackageClient, path: []string{"uv", "sync"}, parents: []packageClientParent{{"Quarantine Python packages with UV", "UV로 Python 패키지를 검역합니다"}}, route: "pypi", use: "sync", shortEN: "Quarantine the complete virtual project before synchronizing its venv", shortKO: "Virtual project 전체를 검역한 뒤 가상환경을 동기화합니다", longEN: uvProjectEnglish, longKO: uvProjectKorean, validate: validateUVProjectArgs, validateRaw: validateUVProjectRawArgs, flags: configureUVProjectFlags, flagText: uvProjectFlagText, examples: []string{"helox uv sync", "helox uv sync --project /absolute/path/project --target /absolute/path/venv"}},
		{id: BunPackageClient, path: []string{"bun", "add"}, parents: []packageClientParent{{"Quarantine project packages with Bun", "Bun으로 프로젝트 패키지를 검역합니다"}}, route: "npm", use: "add <package>[@<version-or-range>]", shortEN: "Quarantine and add a public npm package with Bun", shortKO: "Bun으로 공개 npm 패키지와 전체 의존성을 검역한 뒤 추가합니다", longEN: bunProjectEnglish, longKO: bunProjectKorean, validate: validateBunProjectArgs, validateRaw: validateBunProjectRawArgs, flags: configureBunProjectFlags, flagText: bunProjectFlagText, examples: []string{"helox bun add is-number@7.0.0", "helox bun add @scope/package --target /absolute/path/project"}},
		{id: BunPackageClient, path: []string{"bun", "install"}, parents: []packageClientParent{{"Quarantine project packages with Bun", "Bun으로 프로젝트 패키지를 검역합니다"}}, route: "npm", use: "install", shortEN: "Quarantine the complete Bun project before installation", shortKO: "Bun 프로젝트 전체를 검역한 뒤 설치합니다", longEN: bunProjectEnglish, longKO: bunProjectKorean, validate: validateBunProjectArgs, validateRaw: validateBunProjectRawArgs, flags: configureBunProjectFlags, flagText: bunProjectFlagText, examples: []string{"helox bun install", "helox bun install --target /absolute/path/project"}},
		{id: BunPackageClient, path: []string{"bun", "update"}, parents: []packageClientParent{{"Quarantine project packages with Bun", "Bun으로 프로젝트 패키지를 검역합니다"}}, route: "npm", use: "update [package-name]", shortEN: "Quarantine and update the complete Bun project", shortKO: "Bun 프로젝트의 선택 패키지 전체를 검역한 뒤 업데이트합니다", longEN: bunProjectEnglish, longKO: bunProjectKorean, validate: validateBunProjectArgs, validateRaw: validateBunProjectRawArgs, flags: configureBunProjectFlags, flagText: bunProjectFlagText, examples: []string{"helox bun update", "helox bun update is-number"}},
		{id: PNPMPackageClient, path: []string{"pnpm", "add"}, parents: []packageClientParent{{"Quarantine project packages with PNPM", "PNPM으로 프로젝트 패키지를 검역합니다"}}, route: "npm", use: "add <package>[@<version-or-range>]", shortEN: "Quarantine and add a public npm package with PNPM", shortKO: "PNPM으로 공개 npm 패키지와 전체 의존성을 검역한 뒤 추가합니다", longEN: pnpmProjectEnglish, longKO: pnpmProjectKorean, validate: validatePNPMProjectArgs, validateRaw: validatePNPMProjectRawArgs, flags: configurePNPMProjectFlags, flagText: pnpmProjectFlagText, examples: []string{"helox pnpm add is-number@7.0.0", "helox pnpm add @scope/package --target /absolute/path/project"}},
		{id: PNPMPackageClient, path: []string{"pnpm", "install"}, parents: []packageClientParent{{"Quarantine project packages with PNPM", "PNPM으로 프로젝트 패키지를 검역합니다"}}, route: "npm", use: "install", shortEN: "Quarantine the complete PNPM project before installation", shortKO: "PNPM 프로젝트 전체를 검역한 뒤 설치합니다", longEN: pnpmProjectEnglish, longKO: pnpmProjectKorean, validate: validatePNPMProjectArgs, validateRaw: validatePNPMProjectRawArgs, flags: configurePNPMProjectFlags, flagText: pnpmProjectFlagText, examples: []string{"helox pnpm install", "helox pnpm install --target /absolute/path/project"}},
		{id: PNPMPackageClient, path: []string{"pnpm", "update"}, parents: []packageClientParent{{"Quarantine project packages with PNPM", "PNPM으로 프로젝트 패키지를 검역합니다"}}, route: "npm", use: "update [package-name]", shortEN: "Quarantine and update the complete PNPM project", shortKO: "PNPM 프로젝트의 선택 패키지 전체를 검역한 뒤 업데이트합니다", longEN: pnpmProjectEnglish, longKO: pnpmProjectKorean, validate: validatePNPMProjectArgs, validateRaw: validatePNPMProjectRawArgs, flags: configurePNPMProjectFlags, flagText: pnpmProjectFlagText, examples: []string{"helox pnpm update", "helox pnpm update is-number"}},
		{id: PipPackageClient, path: []string{"pip", "install"}, parents: []packageClientParent{{"Inspect and install PyPI distributions", "PyPI 배포 파일 검사·설치"}}, route: "pip", use: "install <project>[@<version>]", shortEN: "Install a PyPI distribution into the active virtual environment", shortKO: "Python 배포 파일과 의존성을 검사한 뒤 설치·업데이트(--upgrade/-U)합니다", longEN: pypiUpgradeDescriptionEnglish, longKO: pypiUpgradeDescriptionKorean, namedSources: true, validate: validatePyPIInstallArgs, parse: artifactpypi.ParseReferenceForSource, flags: configurePipInstallFlags, upgrade: pipPackageUpgradeMode},
		{id: UVPipPackageClient, path: []string{"uv", "pip", "install"}, parents: []packageClientParent{{"Quarantine Python packages with UV", "UV로 Python 패키지를 검역합니다"}, {"Inspect and install public PyPI distributions with UV", "UV pip으로 공개 PyPI 파일을 검사·설치합니다"}}, route: "pypi", use: "install <project>[==<version>]", shortEN: "Quarantine and install a public PyPI wheel set selected by UV", shortKO: "UV가 선택한 공개 PyPI wheel과 의존성을 검역한 뒤 설치합니다", longEN: uvPipInstallEnglish, longKO: uvPipInstallKorean, validate: validateUVPipInstallArgs, validateRaw: validateUVPipRawArgs, parse: parseUVPipInstallReference, flags: configureUVPipInstallFlags, upgrade: uvPackageUpgradeMode, flagText: uvPipUpgradeFlagText},
		{id: UVPipPackageClient, path: []string{"uv", "pip", "sync"}, parents: []packageClientParent{{"Quarantine Python packages with UV", "UV로 Python 패키지를 검역합니다"}, {"Inspect and install public PyPI distributions with UV", "UV pip으로 공개 PyPI 파일을 검사·설치합니다"}}, route: "pypi", use: "sync <source-file>", shortEN: "Quarantine a complete wheel set before synchronizing a virtual environment", shortKO: "전체 wheel 목록을 검역한 뒤 가상환경을 동기화합니다", longEN: uvPipSyncEnglish, longKO: uvPipSyncKorean, validate: validateUVPipSyncArgs, validateRaw: validateUVPipSyncRawArgs, flags: configureUVPipSyncFlags, flagText: uvPipSyncFlagText, examples: []string{"helox uv pip sync requirements.txt --target /absolute/path/venv", "helox uv pip sync pylock.toml --target /absolute/path/venv", "helox uv pip sync empty.txt --target /absolute/path/venv --allow-empty-requirements"}},
	}
}

func configurePipInstallFlags(command *cobra.Command) {
	command.Flags().BoolP("upgrade", "U", false, "Inspect the complete selected closure before upgrading one distribution (only-if-needed)")
	command.Flags().Bool("adopt", false, "Enroll supported native wheel files after independent ownership verification and full inspection")
	command.Flags().String("source", "pypi", pythonSourceHelp())
	command.Flags().String("inspection-prerequisites", "", "Explicit pinned JSON for one inspection-only non-root leaf prerequisite; not installed into the target")
}

func packageClientByOperation(id PackageClientID, operation string) (packageClientDescriptor, bool) {
	for _, descriptor := range packageClientDescriptors() {
		if descriptor.id == id && descriptor.path[len(descriptor.path)-1] == operation {
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
		command.Example = strings.Join(descriptor.examples, "\n")
		if descriptor.flags != nil {
			descriptor.flags(command)
		}
		parent.AddCommand(command)
	}
}

// PackageClientInstallCommand obtains the actual registered command from the
// same descriptor used by preflight, help and binding.
func PackageClientInstallCommand(root *cobra.Command, id PackageClientID) (*cobra.Command, error) {
	return packageClientCommand(root, id, "install")
}

func packageClientCommand(root *cobra.Command, id PackageClientID, operation string) (*cobra.Command, error) {
	descriptor, ok := packageClientByOperation(id, operation)
	if root == nil || !ok {
		return nil, errors.New("unsupported package client command")
	}
	command, remaining, err := root.Find(descriptor.path)
	if err != nil || len(remaining) != 0 || relativeCommandPath(command) != strings.Join(descriptor.path, " ") {
		return nil, errors.New("package client install command is not registered")
	}
	return command, nil
}

func SelectedPackageClientOperation(root *cobra.Command) string {
	if root == nil {
		return ""
	}
	return root.Annotations[selectedPackageClientOperationAnnotation]
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
			for name, text := range descriptor.flagText {
				if flag := command.Flags().Lookup(name); flag != nil {
					flag.Usage = p.text(text.en, text.ko)
				}
			}
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
	_, err = uvPackageUpgradeMode(command, args)
	return err
}

func validateUVPipRawArgs(args []string) error {
	counts := map[string]int{}
	for _, value := range args {
		name, _, _ := strings.Cut(value, "=")
		if name == "--target" || name == "--upgrade-package" {
			counts[name]++
		}
	}
	for _, count := range counts {
		if count > 1 {
			return errors.New("UV pip requires unambiguous target and upgrade-package options")
		}
	}
	return nil
}

var uvPipUpgradeFlagText = map[string]packageClientParent{
	"upgrade":         {"Upgrade the requested root and its selected dependency closure after fresh quarantine", "요청 패키지와 선택된 의존성을 새로 검역한 뒤 업데이트합니다"},
	"upgrade-package": {"Upgrade only the requested root (one matching bare project name)", "요청한 패키지만 우선 업데이트합니다(동일한 패키지 이름 하나)"},
	"adopt":           {"Explicitly enroll supported default native UV wheel files after independent old-wheel verification", "공식 이전 wheel과 독립 대조한 뒤 지원하는 기본 UV 설치의 소유권을 등록합니다"},
}

func configureUVPipInstallFlags(command *cobra.Command) {
	command.Flags().BoolP("upgrade", "U", false, uvPipUpgradeFlagText["upgrade"].en)
	command.Flags().String("upgrade-package", "", uvPipUpgradeFlagText["upgrade-package"].en)
	command.Flags().Bool("adopt", false, uvPipUpgradeFlagText["adopt"].en)
}
func pipPackageUpgradeMode(command *cobra.Command, _ []string) (packageUpgradeMode, error) {
	upgrade, err := command.Flags().GetBool("upgrade")
	if err != nil {
		return packageNoUpgrade, err
	}
	if upgrade {
		return packageUpgradeRoot, nil
	}
	return packageNoUpgrade, nil
}
func uvPackageUpgradeMode(command *cobra.Command, args []string) (packageUpgradeMode, error) {
	upgrade, err := command.Flags().GetBool("upgrade")
	if err != nil {
		return packageNoUpgrade, err
	}
	selected, err := command.Flags().GetString("upgrade-package")
	if err != nil {
		return packageNoUpgrade, err
	}
	if command.Flags().Changed("upgrade-package") {
		if upgrade || len(args) != 1 {
			return packageNoUpgrade, errors.New("UV upgrade options are ambiguous")
		}
		name, err := artifactpypi.NormalizeProjectName(selected)
		if err != nil {
			return packageNoUpgrade, errors.New("UV upgrade-package requires the requested bare project name")
		}
		reference, err := parseUVPipInstallReference(args[0], artifactpypi.PublicPyPIProfile().Source())
		if err != nil {
			return packageNoUpgrade, err
		}
		root, err := artifactpypi.RequestedProject(reference)
		if err != nil || name != root {
			return packageNoUpgrade, errors.New("UV upgrade-package must match the requested root")
		}
		return packageUpgradeRoot, nil
	}
	if upgrade {
		return packageUpgradeClosure, nil
	}
	adopt, err := command.Flags().GetBool("adopt")
	if err != nil {
		return packageNoUpgrade, err
	}
	if adopt {
		return packageNoUpgrade, errors.New("UV --adopt requires an explicit upgrade option")
	}
	return packageNoUpgrade, nil
}

func AddUVPipInstall(root *cobra.Command, installer Installer) error {
	return AddUVPipInstallAndUpdates(root, installer, nil, nil)
}
func AddUVPipInstallAndUpdates(root *cobra.Command, installer, closureUpdater, rootUpdater Installer) error {
	if installer == nil {
		return errors.New("UV pip install requires a prepared use case")
	}
	return addPackageClientInstall(root, UVPipPackageClient, map[string]Installer{"pypi": installer}, map[packageUpgradeMode]map[string]Installer{packageUpgradeClosure: {"pypi": closureUpdater}, packageUpgradeRoot: {"pypi": rootUpdater}})
}
