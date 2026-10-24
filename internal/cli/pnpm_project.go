package cli

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	artifactnpm "github.com/rahoney/heliopause/internal/artifact/npm"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/spf13/cobra"
)

type PNPMProjectRequest struct {
	Intent   artifactnpm.PNPMSelectionIntent
	Selector string
	Install  domain.InstallContext
	Adopt    bool
}
type PNPMProjectOutcome struct {
	Verified                   domain.ProjectVerifiedSet
	ManifestID                 domain.ContentDigest
	ManifestHandle, SBOMHandle string
}
type PNPMProjectApplier interface {
	ApplyPNPM(context.Context, PNPMProjectRequest) (PNPMProjectOutcome, error)
}

const pnpmClientVersionAnnotation = "helox.pnpm-client-version"

// Static composition supplies the reviewed identity without importing runtime
// implementation packages or preparing any native tool from CLI preflight.
func SetPNPMClientVersion(root *cobra.Command, version string) error {
	if root == nil || version == "" {
		return errors.New("pnpm requires a reviewed client identity")
	}
	if root.Annotations == nil {
		root.Annotations = map[string]string{}
	}
	root.Annotations[pnpmClientVersionAnnotation] = version
	return nil
}

const pnpmProjectEnglish = "Quarantine the complete public npm package set selected by PNPM before installing it into one existing project. Use the current directory or --target. Native PNPM selects package.json and pnpm-lock.yaml; every selected package is inspected again, including unchanged versions. Root project scripts are preserved and do not run.\n\nAdd accepts one public package name or version/range spec. Install uses the existing manifest and optional lock; an existing lock must match the manifest. Update optionally selects one existing root package by name. Peer instances and approved internal links are preserved. Supports Linux amd64 with the pinned runtime. --adopt explicitly registers supported edited controls in an HAA-managed project after checking the existing payload. Existing unmanaged PNPM payloads require separate ownership support."
const pnpmProjectKorean = "PNPM이 선택한 공개 npm 패키지 전체를 검역한 뒤 기존 프로젝트 하나에 설치합니다. 현재 디렉터리 또는 --target을 사용하세요. Native PNPM이 package.json과 pnpm-lock.yaml을 선택하며 같은 버전을 포함한 선택 패키지 전체를 새로 검사합니다. 프로젝트 자체의 script는 보존하고 실행하지 않습니다.\n\nAdd는 공개 패키지 이름 또는 버전·범위 지정 하나를 받습니다. Install은 기존 manifest와 선택적인 lock을 사용하며 기존 lock은 manifest와 일치해야 합니다. Update에는 기존 직접 의존성 이름 하나를 지정할 수 있습니다. Peer instance와 승인된 내부 link를 보존하며 고정 실행 환경의 Linux amd64를 지원합니다. --adopt는 기존 파일을 검증한 뒤 HAA 관리 프로젝트의 지원하는 설정 변경을 명시적으로 등록합니다. 일반 PNPM 설치 파일의 인계는 별도 소유권 지원이 필요합니다."

var pnpmProjectFlagText = map[string]packageClientParent{
	"target": {"Existing project directory (defaults to the current directory)", "기존 프로젝트 디렉터리(기본값: 현재 디렉터리)"},
	"adopt":  {"Explicitly register supported edited controls in an HAA-managed project", "HAA 관리 프로젝트의 지원하는 설정 변경을 명시적으로 등록합니다"},
}

func configurePNPMProjectFlags(command *cobra.Command) {
	command.Flags().Bool("adopt", false, pnpmProjectFlagText["adopt"].en)
	command.Flags().Lookup("target").Usage = pnpmProjectFlagText["target"].en
}
func validatePNPMProjectRawArgs(args []string) error {
	counts := map[string]int{}
	for _, arg := range args {
		name, _, _ := strings.Cut(arg, "=")
		if name == "--target" || name == "--adopt" {
			counts[name]++
			if counts[name] > 1 {
				return errors.New("pnpm project options must not repeat")
			}
		}
	}
	return nil
}
func validatePNPMProjectArgs(command *cobra.Command, args []string) error {
	_, err := readPNPMProjectRequest(command, args)
	return err
}

func readPNPMPreflightControl(root *os.Root, name string) (control domain.ProjectControlFile, resultErr error) {
	before, err := root.Lstat(name)
	if name == "pnpm-lock.yaml" && errors.Is(err, os.ErrNotExist) {
		return domain.NewProjectControlFile(name, nil, false)
	}
	regular := func(info os.FileInfo) bool {
		if info == nil || !info.Mode().IsRegular() || info.Mode()&^os.ModePerm != 0 || info.Size() < 0 || info.Size() > artifactnpm.ProjectControlLimit {
			return false
		}
		s, ok := info.Sys().(*syscall.Stat_t)
		return ok && s.Nlink == 1
	}
	if err != nil || !regular(before) {
		return control, errors.Join(errors.New("pnpm control must be bounded single-link regular content"), err)
	}
	f, err := root.OpenFile(name, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return control, err
	}
	defer func() {
		resultErr = errors.Join(resultErr, f.Close())
		if resultErr != nil {
			control = domain.ProjectControlFile{}
		}
	}()
	opened, err := f.Stat()
	if err != nil || !regular(opened) || !os.SameFile(before, opened) {
		return control, errors.New("pnpm control changed while opening")
	}
	body, readErr := io.ReadAll(io.LimitReader(f, artifactnpm.ProjectControlLimit+1))
	after, statErr := f.Stat()
	current, currentErr := root.Lstat(name)
	if readErr != nil || statErr != nil || currentErr != nil || !regular(after) || !regular(current) || !os.SameFile(before, after) || !os.SameFile(before, current) || after.Mode() != before.Mode() || current.Mode() != before.Mode() || int64(len(body)) != before.Size() || after.Size() != before.Size() || current.Size() != before.Size() || !after.ModTime().Equal(before.ModTime()) || !current.ModTime().Equal(before.ModTime()) {
		return control, errors.Join(errors.New("pnpm control changed during preflight"), readErr, statErr, currentErr)
	}
	return domain.NewProjectControlFile(name, body, true)
}

// This is syntax preflight. The application subsequently acquires its shared
// target lock and independently freezes all original controls and payload.
func readPNPMProjectRequest(command *cobra.Command, args []string) (request PNPMProjectRequest, resultErr error) {
	intent := artifactnpm.PNPMSelectionIntent(command.Name())
	switch intent {
	case artifactnpm.PNPMAdd:
		if err := cobra.ExactArgs(1)(command, args); err != nil {
			return request, err
		}
	case artifactnpm.PNPMInstall:
		if err := cobra.NoArgs(command, args); err != nil {
			return request, err
		}
	case artifactnpm.PNPMUpdate:
		if err := cobra.MaximumNArgs(1)(command, args); err != nil {
			return request, err
		}
	default:
		return request, errors.New("unsupported pnpm project operation")
	}
	selector := ""
	if len(args) == 1 {
		selector = args[0]
		if selector == "" {
			return request, errors.New("pnpm selector must not be empty")
		}
	}
	target, err := command.Flags().GetString("target")
	if err != nil {
		return request, err
	}
	if command.Flags().Changed("target") && target == "" {
		return request, errors.New("pnpm --target must not be empty")
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
		return request, errors.New("pnpm requires an existing project without symlinked ancestry or mount delimiters")
	}
	before, err := os.Lstat(target)
	if err != nil || !before.IsDir() {
		return request, errors.New("pnpm target must be an existing project directory")
	}
	root, err := os.OpenRoot(target)
	if err != nil {
		return request, err
	}
	defer func() {
		resultErr = errors.Join(resultErr, root.Close())
		if resultErr != nil {
			request = PNPMProjectRequest{}
		}
	}()
	controls := []domain.ProjectControlFile{}
	for _, name := range []string{"package.json", "pnpm-lock.yaml"} {
		control, err := readPNPMPreflightControl(root, name)
		if err != nil {
			return request, err
		}
		controls = append(controls, control)
	}
	version := command.Root().Annotations[pnpmClientVersionAnnotation]
	if version == "" {
		return request, errors.New("pnpm reviewed client identity was not bound by composition")
	}
	if _, err := artifactnpm.NewPNPMSelectionInput(intent, selector, controls, version); err != nil {
		return request, err
	}
	current, err := os.Lstat(target)
	opened, openErr := root.Stat(".")
	if err != nil || openErr != nil || !os.SameFile(before, current) || !os.SameFile(before, opened) || current.Mode() != before.Mode() {
		return request, errors.New("pnpm project changed during preflight")
	}
	destination, err := domain.NewInstallTarget(target)
	if err != nil {
		return request, err
	}
	install, err := domain.NewNPMProjectInstallContext(destination)
	if err != nil {
		return request, err
	}
	adopt, err := command.Flags().GetBool("adopt")
	if err != nil {
		return request, err
	}
	return PNPMProjectRequest{intent, selector, install, adopt}, nil
}

func AddPNPMProject(root *cobra.Command, service PNPMProjectApplier) error {
	if root == nil || service == nil {
		return errors.New("pnpm requires guarded project application")
	}
	for _, operation := range []string{"add", "install", "update"} {
		command, err := packageClientCommand(root, PNPMPackageClient, operation)
		if err != nil {
			return err
		}
		command.RunE = func(command *cobra.Command, args []string) error {
			request, err := readPNPMProjectRequest(command, args)
			if err != nil {
				return err
			}
			ctx, finish := beginProgress(contextOrBackground(command.Context()), command.OutOrStdout(), machineOutput(command))
			defer func() { _ = finish() }()
			outcome, cause := service.ApplyPNPM(ctx, request)
			prefix := strings.TrimSuffix(outcome.ManifestHandle, ":manifest.json")
			if cause == nil && (!outcome.Verified.Valid() || outcome.Verified.Inspected().Snapshot().Context() != request.Install || outcome.Verified.Inspected().Snapshot().Source().String() != "npm" || outcome.ManifestID.String() == "" || prefix == outcome.ManifestHandle || !strings.HasPrefix(prefix, "project-cache:") || outcome.SBOMHandle != prefix+":sbom.cdx.json") {
				cause = errors.New("pnpm returned no exact completed publication records")
			}
			progressErr := finish()
			var records *machineVerifiedSet
			if cause == nil {
				records = &machineVerifiedSet{ManifestID: outcome.ManifestID.String(), SBOMHandle: outcome.SBOMHandle, StagingHandle: prefix, EntryCount: len(outcome.Verified.Inspected().Inspections())}
			}
			reportErr := writeProjectMutationReportWithRecords(command.OutOrStdout(), outcome.Verified, cause, request.Selector, "npm", records, request.Intent != artifactnpm.PNPMUpdate)
			return errors.Join(cause, progressErr, reportErr)
		}
	}
	return nil
}
