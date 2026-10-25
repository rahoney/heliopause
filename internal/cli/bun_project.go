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

type BunProjectRequest struct {
	Intent   artifactnpm.BunSelectionIntent
	Selector string
	Install  domain.InstallContext
}
type BunProjectOutcome struct {
	Verified                   domain.ProjectVerifiedSet
	ManifestID                 domain.ContentDigest
	ManifestHandle, SBOMHandle string
}
type BunProjectApplier interface {
	ApplyBun(context.Context, BunProjectRequest) (BunProjectOutcome, error)
}

const bunClientVersionAnnotation = "helox.bun-client-version"

// Static composition supplies the reviewed identity without importing runtime
// implementation packages or preparing any native tool from CLI preflight.
func SetBunClientVersion(root *cobra.Command, version string) error {
	if root == nil || version == "" {
		return errors.New("bun requires a reviewed client identity")
	}
	if root.Annotations == nil {
		root.Annotations = map[string]string{}
	}
	root.Annotations[bunClientVersionAnnotation] = version
	return nil
}

const bunProjectEnglish = "Quarantine the complete public npm package set selected by Bun before installing it into one existing project. Use the current directory or --target. Add accepts one public package name or version/range; update optionally selects one existing root package. Every selected package is inspected again, including unchanged versions. Root project scripts are preserved and do not run.\n\nSupports the pinned Bun hoisted layout on Linux amd64 and Node-compatible package lifecycle inspection. Empty projects preserve an absent bun.lock. Updates require the complete HAA ownership record for the existing installed payload."
const bunProjectKorean = "Bun이 선택한 공개 npm 패키지와 전체 의존성을 검역한 뒤 기존 프로젝트 하나에 설치합니다. 현재 디렉터리 또는 --target을 사용하세요. Add에는 공개 패키지 이름이나 버전·범위를 지정하고 update에는 기존 직접 의존성 이름 하나를 지정할 수 있습니다. 같은 버전도 선택한 패키지 전체를 새로 검사하며 프로젝트 자체의 script는 보존하고 실행하지 않습니다.\n\n고정 Bun의 hoisted 형식과 Linux amd64, Node 호환 package lifecycle 검사를 지원합니다. 빈 프로젝트에서는 bun.lock의 부재를 보존합니다. 기존 설치 업데이트에는 전체 HAA 소유 기록이 필요합니다."

var bunProjectFlagText = map[string]packageClientParent{
	"target": {"Existing project directory (defaults to the current directory)", "기존 프로젝트 디렉터리(기본값: 현재 디렉터리)"},
}

func configureBunProjectFlags(command *cobra.Command) {
	command.Flags().Lookup("target").Usage = bunProjectFlagText["target"].en
}
func validateBunProjectRawArgs(args []string) error {
	counts := map[string]int{}
	for _, arg := range args {
		name, _, _ := strings.Cut(arg, "=")
		if name == "--target" {
			counts[name]++
			if counts[name] > 1 {
				return errors.New("bun project options must not repeat")
			}
		}
	}
	return nil
}
func validateBunProjectArgs(command *cobra.Command, args []string) error {
	_, err := readBunProjectRequest(command, args)
	return err
}

// This is syntax preflight. The application subsequently acquires its shared
// target lock and independently freezes all original controls and payload.
func readBunProjectRequest(command *cobra.Command, args []string) (request BunProjectRequest, resultErr error) {
	intent := artifactnpm.BunSelectionIntent(command.Name())
	switch intent {
	case artifactnpm.BunAdd:
		if err := cobra.ExactArgs(1)(command, args); err != nil {
			return request, err
		}
	case artifactnpm.BunInstall:
		if err := cobra.NoArgs(command, args); err != nil {
			return request, err
		}
	case artifactnpm.BunUpdate:
		if err := cobra.MaximumNArgs(1)(command, args); err != nil {
			return request, err
		}
	default:
		return request, errors.New("unsupported bun project operation")
	}
	selector := ""
	if len(args) == 1 {
		selector = args[0]
		if selector == "" {
			return request, errors.New("bun selector must not be empty")
		}
	}
	target, err := command.Flags().GetString("target")
	if err != nil {
		return request, err
	}
	if command.Flags().Changed("target") && target == "" {
		return request, errors.New("bun --target must not be empty")
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
		return request, errors.New("bun requires an existing project without symlinked ancestry or mount delimiters")
	}
	before, err := os.Lstat(target)
	if err != nil || !before.IsDir() {
		return request, errors.New("bun target must be an existing project directory")
	}
	root, err := os.OpenRoot(target)
	if err != nil {
		return request, err
	}
	defer func() {
		resultErr = errors.Join(resultErr, root.Close())
		if resultErr != nil {
			request = BunProjectRequest{}
		}
	}()
	controls := []domain.ProjectControlFile{}
	for _, name := range []string{"package.json", "bun.lock"} {
		control, err := readNPMProjectPreflightControl(root, name, "bun.lock", "bun")
		if err != nil {
			return request, err
		}
		controls = append(controls, control)
	}
	version := command.Root().Annotations[bunClientVersionAnnotation]
	if version == "" {
		return request, errors.New("bun reviewed client identity was not bound by composition")
	}
	if _, err := artifactnpm.NewBunSelectionInput(intent, selector, controls, version); err != nil {
		return request, err
	}
	current, err := os.Lstat(target)
	opened, openErr := root.Stat(".")
	if err != nil || openErr != nil || !os.SameFile(before, current) || !os.SameFile(before, opened) || current.Mode() != before.Mode() {
		return request, errors.New("bun project changed during preflight")
	}
	destination, err := domain.NewInstallTarget(target)
	if err != nil {
		return request, err
	}
	install, err := domain.NewNPMProjectInstallContext(destination)
	if err != nil {
		return request, err
	}
	return BunProjectRequest{intent, selector, install}, nil
}

func AddBunProject(root *cobra.Command, service BunProjectApplier) error {
	if root == nil || service == nil {
		return errors.New("bun requires guarded project application")
	}
	for _, operation := range []string{"add", "install", "update"} {
		command, err := packageClientCommand(root, BunPackageClient, operation)
		if err != nil {
			return err
		}
		command.RunE = func(command *cobra.Command, args []string) error {
			request, err := readBunProjectRequest(command, args)
			if err != nil {
				return err
			}
			ctx, finish := beginProgress(contextOrBackground(command.Context()), command.OutOrStdout(), machineOutput(command))
			defer func() { _ = finish() }()
			outcome, cause := service.ApplyBun(ctx, request)
			prefix := strings.TrimSuffix(outcome.ManifestHandle, ":manifest.json")
			if cause == nil && (!outcome.Verified.Valid() || outcome.Verified.Inspected().Snapshot().Context() != request.Install || outcome.Verified.Inspected().Snapshot().Source().String() != "npm" || outcome.ManifestID.String() == "" || prefix == outcome.ManifestHandle || !strings.HasPrefix(prefix, "project-cache:") || outcome.SBOMHandle != prefix+":sbom.cdx.json") {
				cause = errors.New("bun returned no exact completed publication records")
			}
			progressErr := finish()
			var records *machineVerifiedSet
			if cause == nil {
				records = &machineVerifiedSet{ManifestID: outcome.ManifestID.String(), SBOMHandle: outcome.SBOMHandle, StagingHandle: prefix, EntryCount: len(outcome.Verified.Inspected().Inspections())}
			}
			reportErr := writeProjectMutationReportWithRecords(command.OutOrStdout(), outcome.Verified, cause, request.Selector, "npm", records, request.Intent != artifactnpm.BunUpdate)
			return errors.Join(cause, progressErr, reportErr)
		}
	}
	return nil
}
