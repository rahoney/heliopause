package cli

import (
	"encoding/json"
	"errors"
	"fmt"

	artifact "github.com/rahoney/heliopause/internal/artifact/githubrelease"
	"github.com/rahoney/heliopause/internal/core/ports"
	"github.com/spf13/cobra"
)

const bootstrapPreparationEnglish = "Download and verify the provenance of one exact official native release for first installation on Linux/WSL2 amd64. This prepares ordinary-user files only: quarantine is NOT_PERFORMED and installation remains PENDING. Independently verify the actual root-protected helper before administrator operations, then configure the native runtime, register the installation and run doctor and a nominal quarantine as the configured ordinary user. Existing managed installations use self-update."
const bootstrapPreparationKorean = "Linux/WSL2 amd64 최초 설치용으로 정확한 공식 native 릴리스 하나를 다운로드하고 출처 서명을 검증합니다. 일반 사용자 파일만 준비하며 검역은 NOT_PERFORMED, 설치는 PENDING 상태입니다. 관리자 작업 전 실제 root 보호 도우미 사본을 독립 검증하고, native 실행 환경 설정·설치 등록 뒤 설정된 일반 사용자로 doctor와 기본 검역을 확인해야 합니다. 기존 관리 설치는 self-update를 사용하세요."

func newBootstrapCommand() *cobra.Command {
	group := &cobra.Command{Use: "bootstrap", Short: "Prepare first-install native release files"}
	command := &cobra.Command{Use: "prepare", Short: "Download and verify one exact native release for first installation", Long: bootstrapPreparationEnglish, Args: validateBootstrapPreparationArgs, RunE: func(*cobra.Command, []string) error { return errors.New("bootstrap preparation is not configured") }}
	command.Flags().String("version", "", "Exact official Heliopause version (vX.Y.Z)")
	group.AddCommand(command)
	return group
}
func validateBootstrapPreparationArgs(command *cobra.Command, args []string) error {
	if err := cobra.NoArgs(command, args); err != nil {
		return err
	}
	version, err := command.Flags().GetString("version")
	if err != nil {
		return err
	}
	return artifact.ValidateHeliopauseVersion(version)
}
func validateBootstrapPreparationIntent(root *cobra.Command, args []string) error {
	return validateVersionIntent(root, args, "helox bootstrap prepare", "bootstrap prepare")
}

func AddBootstrapPreparation(root *cobra.Command, service ports.BootstrapPreparation) error {
	if root == nil || service == nil {
		return errors.New("bootstrap preparation service is required")
	}
	command, _, err := root.Find([]string{"bootstrap", "prepare"})
	if err != nil || command.CommandPath() != "helox bootstrap prepare" {
		return errors.New("bootstrap preparation is not registered")
	}
	command.RunE = func(command *cobra.Command, args []string) error {
		if err := validateBootstrapPreparationArgs(command, args); err != nil {
			return err
		}
		version, _ := command.Flags().GetString("version")
		p := presentationFor(command.OutOrStdout())
		ctx, stop := beginProgress(contextOrBackground(command.Context()), command.OutOrStdout(), machineOutput(command))
		result, prepareErr := service.Prepare(ctx, version)
		progressErr := stop()
		combined := errors.Join(prepareErr, progressErr)
		if combined != nil {
			code := ports.BootstrapExportFailed
			var failure *ports.BootstrapPreparationFailure
			if errors.As(prepareErr, &failure) {
				code = failure.Code()
			}
			p.bootstrapPreparationReport(result, string(code), false)
			p.reportedError = true
			return errors.Join(ExitError{Code: 1}, combined, p.writeErr)
		}
		p.bootstrapPreparationReport(result, "", true)
		return p.writeErr
	}
	return nil
}

func (p *presentation) bootstrapPreparationReport(result ports.BootstrapPreparationResult, code string, success bool) {
	status := "FAILED"
	if success {
		status = "PREPARED"
	}
	if !p.human() {
		value := struct {
			Schema       string                           `json:"schema_version"`
			Status       string                           `json:"status"`
			Code         string                           `json:"code,omitempty"`
			Quarantine   string                           `json:"quarantine"`
			Installation string                           `json:"installation"`
			Provenance   string                           `json:"provenance"`
			Preparation  ports.BootstrapPreparationResult `json:"preparation"`
		}{Schema: "helox.bootstrap-preparation/v1", Status: status, Code: code, Quarantine: "NOT_PERFORMED", Installation: "PENDING", Provenance: "UNCONFIRMED", Preparation: result}
		if success {
			value.Provenance = "VERIFIED"
		}
		p.writeErr = json.NewEncoder(p.out).Encode(value)
		return
	}
	p.line("%s  %s", p.text("Bootstrap preparation", "최초 설치 파일 준비"), status)
	if result.Version != "" {
		p.line("%s  %s", p.text("Version", "버전"), safeTerminal(result.Version))
	}
	if success {
		p.line("%s  VERIFIED", p.text("Provenance", "출처 서명"))
		p.line("%s  %d", p.text("Prepared files", "준비한 파일"), len(result.Files))
	} else {
		p.line("%s  %s", p.text("Error code", "오류 코드"), code)
	}
	p.line("%s  NOT_PERFORMED", p.text("Quarantine", "검역"))
	p.line("%s  PENDING", p.text("Installation", "설치"))
	if result.Directory != "" {
		p.line("%s  %s", p.text("Directory", "디렉터리"), fmt.Sprintf("%q", result.Directory))
	}
	if success {
		p.line("%s", p.text("Independently verify the root-protected helper before administrator operations. Registration, doctor and nominal quarantine remain required.", "관리자 작업 전 root 보호 도우미 사본을 독립 검증하세요. 설치 등록, doctor와 기본 검역 확인이 남아 있습니다."))
	} else {
		p.line("%s", p.text("No installation was performed. Check the stable error code; a retained directory may require manual cleanup.", "설치는 수행하지 않았습니다. 오류 코드를 확인하세요. 남은 디렉터리는 별도 정리가 필요할 수 있습니다."))
	}
}
