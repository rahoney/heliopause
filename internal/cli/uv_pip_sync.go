package cli

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	artifactpypi "github.com/rahoney/heliopause/internal/artifact/pypi"
	"github.com/rahoney/heliopause/internal/core/domain"
	"github.com/spf13/cobra"
)

const uvPipSyncEnglish = "Quarantine every exact public PyPI wheel in one complete requirements file or strict pylock.toml before synchronizing an existing virtual environment. Packages excluded from the file are removed only after original ownership verification and complete-set approval.\n\nRequirements must list the full dependency closure using exact final versions (name==version). Includes, version ranges, editable/local/VCS/direct sources and arbitrary native options are unsupported. A blank requirements file is skipped by default; --allow-empty-requirements explicitly requests an empty environment. An explicit empty pylock also follows the full approval path. Supports Linux amd64 with the pinned Python runtime. Use --adopt for supported native UV installations. User files are preserved; all selected wheels are inspected again, including unchanged versions."
const uvPipSyncKorean = "완전한 requirements 파일 또는 엄격한 pylock.toml의 공개 PyPI wheel 전체를 검역한 뒤 기존 가상환경을 동기화합니다. 목록에서 제외된 패키지는 기존 파일의 소유권을 검증하고 전체 목록이 허용된 뒤 제거합니다.\n\nRequirements에는 모든 의존성을 최종 버전으로 고정(name==version)해 나열해야 합니다. Include·버전 범위·editable/local/VCS/direct 출처와 임의 native 옵션은 지원하지 않습니다. 빈 requirements는 기본적으로 건너뛰며 --allow-empty-requirements로 빈 환경을 명시할 수 있습니다. 명시적 빈 pylock도 전체 승인 경로를 거칩니다. 고정된 Python 실행 환경의 Linux amd64를 지원합니다. 지원하는 일반 UV 설치를 인계받으려면 --adopt를 사용하세요. 사용자 파일은 보존하며 동일 버전을 포함한 선택된 wheel 전체를 새로 검사합니다."

type UVPipSyncRequest struct {
	SourceFile        string
	Format            artifactpypi.UVSyncSourceFormat
	AllowEmpty, Adopt bool
	Install           domain.InstallContext
}
type UVPipSyncOutcome struct {
	Verified                   domain.ProjectVerifiedSet
	ManifestID                 domain.ContentDigest
	ManifestHandle, SBOMHandle string
}
type UVPipSyncer interface {
	Sync(context.Context, UVPipSyncRequest) (UVPipSyncOutcome, error)
}

var uvPipSyncFlagText = map[string]packageClientParent{
	"allow-empty-requirements": {"Explicitly approve synchronization to an empty requirements set", "빈 requirements 목록으로 동기화하도록 명시합니다"},
	"adopt":                    uvPipUpgradeFlagText["adopt"],
}

func configureUVPipSyncFlags(command *cobra.Command) {
	command.Flags().Bool("adopt", false, uvPipSyncFlagText["adopt"].en)
	command.Flags().Bool("allow-empty-requirements", false, uvPipSyncFlagText["allow-empty-requirements"].en)
}
func validateUVPipSyncRawArgs(args []string) error {
	counts := map[string]int{}
	for _, arg := range args {
		name, _, _ := strings.Cut(arg, "=")
		if name == "--target" || name == "--adopt" || name == "--allow-empty-requirements" {
			counts[name]++
			if counts[name] > 1 {
				return errors.New("UV pip sync options must not repeat")
			}
		}
	}
	return nil
}
func validateUVPipSyncArgs(command *cobra.Command, args []string) error {
	_, _, err := readUVPipSyncRequest(command, args)
	return err
}

// This is bounded syntax preflight, never ownership or selection authority.
// The actual transaction independently freezes and verifies source and target.
func readUVPipSyncRequest(command *cobra.Command, args []string) (UVPipSyncRequest, bool, error) {
	var request UVPipSyncRequest
	if err := cobra.ExactArgs(1)(command, args); err != nil {
		return request, false, err
	}
	if args[0] == "" || args[0] == "-" {
		return request, false, errors.New("UV pip sync requires one regular source file")
	}
	source, err := filepath.Abs(args[0])
	if err != nil {
		return request, false, err
	}
	resolved, err := filepath.EvalSymlinks(source)
	if err != nil || resolved != source {
		return request, false, errors.New("UV pip sync source or ancestry is unavailable or symlinked")
	}
	info, err := os.Lstat(source)
	if err != nil || !info.Mode().IsRegular() || info.Size() < 0 || info.Size() > artifactpypi.UVSyncSourceByteLimit {
		return request, false, errors.New("UV pip sync source is not bounded regular content")
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Nlink != 1 {
		return request, false, errors.New("UV pip sync source must have one file link")
	}
	file, err := os.OpenFile(source, os.O_RDONLY|syscall.O_NONBLOCK|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return request, false, err
	}
	opened, statErr := file.Stat()
	if statErr != nil || !os.SameFile(info, opened) || !opened.Mode().IsRegular() {
		return request, false, errors.Join(errors.New("UV pip sync source changed while opening"), file.Close())
	}
	body, readErr := io.ReadAll(io.LimitReader(file, artifactpypi.UVSyncSourceByteLimit+1))
	after, afterErr := file.Stat()
	closeErr := file.Close()
	current, currentErr := os.Lstat(source)
	if readErr != nil || afterErr != nil || closeErr != nil || currentErr != nil || len(body) > artifactpypi.UVSyncSourceByteLimit || int64(len(body)) != info.Size() || !os.SameFile(info, after) || !os.SameFile(info, current) || after.Mode() != info.Mode() || after.Size() != info.Size() || !after.ModTime().Equal(info.ModTime()) {
		return request, false, errors.New("UV pip sync source changed during preflight")
	}
	format := artifactpypi.UVSyncRequirements
	name := filepath.Base(source)
	if name == "pylock.toml" || strings.HasPrefix(name, "pylock.") && strings.HasSuffix(name, ".toml") {
		format = artifactpypi.UVSyncPylock
	}
	allow, _ := command.Flags().GetBool("allow-empty-requirements")
	adopt, _ := command.Flags().GetBool("adopt")
	if format == artifactpypi.UVSyncPylock && allow {
		return request, false, errors.New("--allow-empty-requirements applies only to requirements files")
	}
	input, err := artifactpypi.ParsePinnedUVSyncInput(format, body, allow)
	if err != nil {
		return request, false, err
	}
	target, _ := command.Flags().GetString("target")
	if command.Flags().Changed("target") && target == "" {
		return request, false, errors.New("UV pip sync --target must not be empty")
	}
	installTarget, err := activePythonVenvForCommand(target, "uv pip sync")
	if err != nil {
		return request, false, err
	}
	if strings.Contains(installTarget.String(), ",") {
		return request, false, errors.New("UV pip sync target is unsupported by the mount boundary")
	}
	install, err := domain.NewPythonVenvInstallContext(installTarget)
	if err != nil {
		return request, false, err
	}
	rel, err := filepath.Rel(installTarget.String(), source)
	if err != nil || rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return request, false, errors.New("UV pip sync source must be outside its target")
	}
	return UVPipSyncRequest{source, format, allow, adopt, install}, input.ShouldSkip(), nil
}

func AddUVPipSync(root *cobra.Command, service UVPipSyncer) error {
	if root == nil || service == nil {
		return errors.New("UV pip sync requires a command and guarded use case")
	}
	command, err := packageClientCommand(root, UVPipPackageClient, "sync")
	if err != nil {
		return err
	}
	command.RunE = func(command *cobra.Command, args []string) error {
		request, skip, err := readUVPipSyncRequest(command, args)
		if err != nil {
			return err
		}
		if skip {
			return errors.New("UV pip sync source became blank after runtime preflight")
		}
		ctx, finish := beginProgress(contextOrBackground(command.Context()), command.OutOrStdout(), machineOutput(command))
		defer func() { _ = finish() }()
		outcome, cause := service.Sync(ctx, request)
		if cause == nil && (!outcome.Verified.Valid() || outcome.Verified.Inspected().Snapshot().Context() != request.Install || outcome.Verified.Inspected().Snapshot().Source() != artifactpypi.PublicPyPIProfile().Source() || outcome.ManifestID.String() == "" || outcome.SBOMHandle == "" || outcome.ManifestHandle == "") {
			cause = errors.New("UV pip sync returned no completed approved publication records")
		}
		progressErr := finish()
		var records *machineVerifiedSet
		if cause == nil {
			records = &machineVerifiedSet{ManifestID: outcome.ManifestID.String(), SBOMHandle: outcome.SBOMHandle, StagingHandle: strings.TrimSuffix(outcome.ManifestHandle, ":manifest.json"), EntryCount: len(outcome.Verified.Inspected().Inspections())}
		}
		reportErr := writeProjectUpdateReportWithRecords(command.OutOrStdout(), outcome.Verified, cause, "", "pypi", records)
		return errors.Join(cause, progressErr, reportErr)
	}
	return nil
}

func prepareUVSyncSkip(command *cobra.Command, args []string) (bool, error) {
	_, skip, err := readUVPipSyncRequest(command, args)
	if err != nil || !skip {
		return false, err
	}
	command.RunE = func(command *cobra.Command, args []string) error {
		_, skip, err := readUVPipSyncRequest(command, args)
		if err != nil {
			return err
		}
		if !skip {
			return errors.New("UV pip sync source changed after skip preflight")
		}
		p := presentationFor(command.OutOrStdout())
		if !p.human() {
			return json.NewEncoder(command.OutOrStdout()).Encode(cargoUpdateReport{Schema: "helox.project-update-result/v1", Operation: "UPDATE", Status: "SKIPPED", Promotion: "NOT_PERFORMED", Source: "pypi", Entries: []cargoUpdateEntry{}})
		}
		p.reportHeader("UPDATE", "SKIPPED", "", "")
		p.line("%s", p.text("Blank requirements: synchronization skipped; the environment was preserved.", "빈 requirements이므로 동기화를 건너뛰었습니다. 기존 환경을 보존했습니다."))
		p.line("%s", p.text("To explicitly synchronize to an empty set, use --allow-empty-requirements.", "빈 목록으로 동기화하려면 --allow-empty-requirements를 명시하세요."))
		return p.writeErr
	}
	return true, nil
}
