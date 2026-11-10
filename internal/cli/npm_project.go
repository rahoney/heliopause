package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/spf13/cobra"
)

const npmClientVersionAnnotation = "helox.npm-client-version"

func SetNPMClientVersion(root *cobra.Command, version string) error {
	if root == nil || version == "" {
		return errors.New("npm requires a reviewed client identity")
	}
	if root.Annotations == nil {
		root.Annotations = map[string]string{}
	}
	root.Annotations[npmClientVersionAnnotation] = version
	return nil
}

const npmProjectEnglish = "Quarantine every public package selected by the pinned npm before changing an existing project. Install without a package uses package.json and reconciles an optional package-lock.json. Ci requires a matching lock and preserves its bytes. Use the current directory or --target. Existing payload requires independent Heliopause ownership; arbitrary native installs are refused. Explicit package installation keeps its existing workflow."
const npmProjectKorean = "고정 npm이 선택한 공개 패키지 전체를 검역한 뒤 기존 프로젝트를 변경합니다. 패키지 인자 없는 install은 package.json과 선택적인 package-lock.json을 조정합니다. Ci는 일치하는 lock이 필수이며 원문을 보존합니다. 현재 디렉터리 또는 --target을 사용하세요. 기존 설치는 Heliopause의 독립 소유 기록이 필요하며 임의의 기존 설치는 거부합니다. 패키지를 명시하는 설치는 기존 흐름을 사용합니다."

func configureNPMProjectFlags(command *cobra.Command) {
	command.Flags().Lookup("target").Usage = "Project directory (defaults to the current directory); explicit package installation retains its existing target contract"
}

func validateNPMProjectRawArgs(args []string) error {
	count := 0
	for _, arg := range args {
		if name, _, _ := strings.Cut(arg, "="); name == "--target" {
			count++
		}
	}
	if count > 1 {
		return errors.New("npm --target must not repeat")
	}
	return nil
}

func validateNPMProjectArgs(command *cobra.Command, args []string) error {
	if command.Name() == "install" && len(args) == 1 {
		if _, err := artifactnpm.ParseReference(args[0]); err != nil {
			return err
		}
		target, err := command.Flags().GetString("target")
		if err != nil {
			return err
		}
		if target != "" {
			_, err = domain.NewInstallTarget(target)
		}
		return err
	}
	_, err := readNPMProjectRequest(command, args)
	return err
}

type NPMProjectRequest struct {
	Intent  artifactnpm.NPMProjectIntent
	Install domain.InstallContext
}
type NPMProjectOutcome struct {
	RootChecked, RootRequired  bool
	RootInspection             domain.OperationResult
	Verified                   domain.ProjectVerifiedSet
	ManifestID                 domain.ContentDigest
	ManifestHandle, SBOMHandle string
}
type NPMProjectApplier interface {
	ApplyNPM(context.Context, NPMProjectRequest) (NPMProjectOutcome, error)
}

func readNPMProjectRequest(command *cobra.Command, args []string) (request NPMProjectRequest, resultErr error) {
	if err := cobra.NoArgs(command, args); err != nil {
		return request, err
	}
	intent := artifactnpm.NPMProjectIntent(command.Name())
	if intent != artifactnpm.NPMProjectInstall && intent != artifactnpm.NPMProjectCI {
		return request, errors.New("unsupported npm whole-project intent")
	}
	target, err := command.Flags().GetString("target")
	if err != nil {
		return request, err
	}
	if command.Flags().Changed("target") && target == "" {
		return request, errors.New("npm --target must not be empty")
	}
	if target == "" {
		target, err = os.Getwd()
		if err != nil {
			return request, err
		}
	}
	target, err = filepath.Abs(target)
	if err != nil {
		return request, err
	}
	resolved, err := filepath.EvalSymlinks(target)
	if err != nil || resolved != target || strings.Contains(target, ",") {
		return request, errors.New("npm requires an existing project without symlinked ancestry or mount delimiters")
	}
	before, err := os.Lstat(target)
	if err != nil || !before.IsDir() {
		return request, errors.New("npm target must be an existing project directory")
	}
	root, err := os.OpenRoot(target)
	if err != nil {
		return request, err
	}
	defer func() { resultErr = errors.Join(resultErr, root.Close()) }()
	var controls []domain.ProjectControlFile
	for _, name := range []string{"package.json", "package-lock.json"} {
		control, err := readNPMProjectPreflightControl(root, name, "package-lock.json", "npm")
		if err != nil {
			return request, err
		}
		controls = append(controls, control)
	}
	version := command.Root().Annotations[npmClientVersionAnnotation]
	if version == "" {
		return request, errors.New("npm reviewed client identity was not bound by composition")
	}
	if _, err := artifactnpm.NewNPMProjectInput(intent, controls, version); err != nil {
		return request, err
	}
	current, err := os.Lstat(target)
	opened, openErr := root.Stat(".")
	if err != nil || openErr != nil || !os.SameFile(before, current) || !os.SameFile(before, opened) || before.Mode() != current.Mode() {
		return request, errors.New("npm project changed during preflight")
	}
	destination, err := domain.NewInstallTarget(target)
	if err != nil {
		return request, err
	}
	install, err := domain.NewNPMProjectInstallContext(destination)
	if err != nil {
		return request, err
	}
	return NPMProjectRequest{intent, install}, nil
}

// Selection comes from the same validated command descriptor and parsed args.
func NPMProjectRequested(root *cobra.Command) bool {
	if SelectedPackageClient(root) != NPMPackageClient {
		return false
	}
	operation := SelectedPackageClientOperation(root)
	command := findLeaf(root, "npm", operation)
	return command != nil && (operation == "ci" || operation == "install" && command.Flags().NArg() == 0)
}

func AddNPMProject(root *cobra.Command, service NPMProjectApplier) error {
	if root == nil || service == nil {
		return errors.New("npm requires guarded whole-project application")
	}
	for _, operation := range []string{"install", "ci"} {
		command, err := packageClientCommand(root, NPMPackageClient, operation)
		if err != nil {
			return err
		}
		explicit := command.RunE
		command.RunE = func(command *cobra.Command, args []string) error {
			if command.Name() == "install" && len(args) != 0 {
				return explicit(command, args)
			}
			request, err := readNPMProjectRequest(command, args)
			if err != nil {
				return err
			}
			ctx, finish := beginProgress(contextOrBackground(command.Context()), command.OutOrStdout(), machineOutput(command))
			defer func() { _ = finish() }()
			outcome, cause := service.ApplyNPM(ctx, request)
			prefix := strings.TrimSuffix(outcome.ManifestHandle, ":manifest.json")
			if cause == nil && (!outcome.Verified.Valid() || outcome.Verified.Inspected().Snapshot().Context() != request.Install || outcome.Verified.Inspected().Snapshot().Source().String() != "npm" || outcome.ManifestID.String() == "" || prefix == outcome.ManifestHandle || !strings.HasPrefix(prefix, "project-cache:") || outcome.SBOMHandle != prefix+":sbom.cdx.json") {
				cause = errors.New("npm returned no exact completed publication records")
			}
			progressErr := finish()
			var records *machineVerifiedSet
			if cause == nil {
				records = &machineVerifiedSet{ManifestID: outcome.ManifestID.String(), SBOMHandle: outcome.SBOMHandle, StagingHandle: prefix, EntryCount: len(outcome.Verified.Inspected().Inspections())}
			}
			return errors.Join(cause, progressErr, writeProjectMutationReportWithRecords(command.OutOrStdout(), outcome.Verified, cause, "", "npm", records, true, projectLifecycleReport{outcome.RootChecked, outcome.RootRequired, outcome.RootInspection}))
		}
	}
	return nil
}
