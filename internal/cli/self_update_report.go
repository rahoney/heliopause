package cli

func (p *presentation) selfUpdateErrorExplanation(code string) string {
	switch code {
	case "SELF_UPDATE_MANAGED_INSTALLATION_REQUIRED":
		return p.text("A verified managed native installation is required. Development and OS-package installations need a separately reviewed bootstrap or migration.", "검증된 native 관리 설치가 필요합니다. 개발용·OS 패키지 설치는 별도로 검토한 최초 설치 또는 전환 과정이 필요합니다.")
	case "SELF_UPDATE_PLATFORM_UNSUPPORTED":
		return p.text("Quarantine self-update requires Linux/WSL2 amd64 and a healthy trusted native runtime with the same runtime lock.", "검역형 본체 업데이트는 Linux/WSL2 amd64와 동일한 runtime lock의 정상 native 실행 환경이 필요합니다.")
	case "SELF_UPDATE_PROTECTED_CONTROLLER_REQUIRED":
		return p.text("Run the protected installed CLI as its configured ordinary user. Do not run the whole CLI with sudo or use a development binary to activate updates.", "설정된 일반 사용자로 보호된 설치 CLI를 실행하세요. CLI 전체를 sudo로 실행하거나 개발용 실행 파일로 본체를 교체할 수 없습니다.")
	case "SELF_UPDATE_BUSY":
		return p.text("Another update owns this installation. Wait for its completion and cleanup; do not remove lock files.", "다른 업데이트가 설치 잠금을 소유하고 있습니다. 완료·정리를 기다리세요. 잠금 파일을 삭제하지 마세요.")
	case "SELF_UPDATE_TRANSITION_OR_RECOVERY_REQUIRED":
		return p.text("The active version is not confirmed and may already have changed. An administrator must resolve the protected installer recovery state before retrying.", "현재 활성 버전을 확정하지 못했으며 이미 교체됐을 수도 있습니다. 관리자가 보호된 설치 도우미의 복구 상태를 해결한 뒤 다시 실행해야 합니다.")
	case "SELF_UPDATE_PROTECTED_INSTALLER_FAILED":
		return p.text("The installed protected helper did not complete successfully. Prepare administrator authentication with sudo -v, check helox doctor and resolve any retained recovery state before retrying.", "설치된 관리자 도우미의 정상 완료를 확인하지 못했습니다. sudo -v로 관리자 인증을 준비하고 helox doctor 및 남아 있는 복구 상태를 확인한 뒤 다시 실행하세요.")
	case "SELF_UPDATE_RUNTIME_CLEANUP_FAILED", "SELF_UPDATE_OWNED_STAGE_CLEANUP_UNCERTAIN":
		return p.text("Execution or owned-file cleanup is uncertain. Retained files were not blindly deleted. Check the active version and finish controlled cleanup before retrying.", "실행 종료 또는 소유 파일 정리가 불확실합니다. 남은 파일을 임의로 삭제하지 않았습니다. 활성 버전과 실제 종료·정리 상태를 확인한 뒤 다시 실행하세요.")
	case "SELF_UPDATE_COMPATIBILITY_HEALTH_FAILED":
		return p.text("The required compatibility health check failed. Update completion is not confirmed; inspect the protected runtime and recovery state.", "필수 호환성·정상 동작 검사가 실패했습니다. 업데이트 완료로 판단할 수 없습니다. 보호된 실행 환경과 복구 상태를 확인하세요.")
	case "SELF_UPDATE_INSTALLATION_IDENTITY_INVALID", "SELF_UPDATE_PAYLOAD_PROTOCOL_INVALID":
		return p.text("The installed identity or exact official release transfer could not be verified. Keep the retained recovery data and check the protected installation before retrying.", "설치 신원 또는 정확한 공식 릴리스 전달을 검증하지 못했습니다. 복구 자료를 보존하고 보호된 설치 상태를 확인한 뒤 다시 실행하세요.")
	default:
		return p.text("Self-update did not complete. Review the reported checks and helox doctor; do not assume the previous version is still active.", "본체 업데이트가 완료되지 않았습니다. 위 검사 결과와 helox doctor를 확인하세요. 이전 버전이 여전히 활성 상태라고 단정할 수 없습니다.")
	}
}
