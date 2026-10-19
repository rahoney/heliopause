package cli

import (
	"strings"

	"github.com/spf13/cobra"
)

func localizeHelp(root *cobra.Command, p *presentation) {
	descriptions := map[string][2]string{
		"helox help":                  {"Help about any command", "명령별 사용법을 표시합니다"},
		"helox completion":            {"Generate shell completion", "셸 자동 완성 스크립트를 생성합니다"},
		"helox completion bash":       {"Generate bash completion", "bash 자동 완성 스크립트를 생성합니다"},
		"helox completion zsh":        {"Generate zsh completion", "zsh 자동 완성 스크립트를 생성합니다"},
		"helox completion fish":       {"Generate fish completion", "fish 자동 완성 스크립트를 생성합니다"},
		"helox completion powershell": {"Generate PowerShell completion", "PowerShell 자동 완성 스크립트를 생성합니다"},
		"helox":                       {"Inspect software artifacts before trusted promotion", "외부 소프트웨어를 검사한 뒤 허용된 파일만 설치합니다"},
		"helox npm":                   {"Inspect and install npm registry packages", "npm 패키지 검사·설치"},
		"helox npm inspect":           {"Inspect an npm package", "프로젝트에 설치하지 않고 npm 패키지를 검사합니다"},
		"helox npm install":           {"Inspect and install an npm package", "npm 패키지와 의존성을 검사하고 허용된 파일만 설치합니다"},
		"helox npm update":            {"Update an npm project after complete quarantine checks", "새 npm 버전과 전체 의존성을 검사한 뒤 프로젝트를 업데이트합니다"},
		"helox pypi":                  {"Inspect and install PyPI distributions", "PyPI 배포 파일 검사·설치"},
		"helox pypi inspect":          {"Inspect a PyPI distribution", "프로젝트에 설치하지 않고 PyPI 배포 파일을 검사합니다"},
		"helox pip":                   {"Inspect and install PyPI distributions", "PyPI 배포 파일 검사·설치"},
		"helox pip install":           {"Inspect and install a PyPI distribution", "Python 배포 파일과 의존성을 검사한 뒤 설치합니다"},
		"helox github":                {"Inspect and install public GitHub Release assets", "GitHub Release 파일 검사·설치"},
		"helox github inspect":        {"Inspect a GitHub Release asset", "GitHub Release 파일을 검사합니다"},
		"helox github install":        {"Inspect and install a GitHub Release asset", "GitHub Release 파일을 검사한 뒤 설치합니다"},
		"helox go":                    {"Inspect and build public Go Modules", "공개 Go 모듈 검사·다운로드·빌드"},
		"helox go get":                {"Inspect and add Go dependencies", "Go 모듈과 의존성을 확인하고 프로젝트를 갱신합니다"},
		"helox go mod":                {"Manage project Go dependencies", "현재 Go 프로젝트의 모듈 작업"},
		"helox go mod download":       {"Inspect and cache project dependencies", "현재 프로젝트 의존성을 검사하고 검증된 캐시에 준비합니다"},
		"helox go build":              {"Build the approved Go project", "허용된 Go 프로젝트를 격리 환경에서 빌드합니다"},
		"helox cargo":                 {"Inspect and build public crates.io packages", "공개 crates.io 패키지 검사·빌드"},
		"helox cargo add":             {"Inspect and add Rust dependencies", "Rust 의존성을 검사한 뒤 프로젝트에 추가합니다"},
		"helox cargo update":          {"Inspect and update the complete Rust lock graph", "전체 Rust 의존성을 검사한 뒤 잠금파일을 업데이트합니다"},
		"helox cargo build":           {"Build the approved Rust project", "허용된 Rust 프로젝트를 격리 환경에서 빌드합니다"},
		"helox terraform":             {"Install public Terraform Providers", "공개 Terraform Provider 설치"},
		"helox terraform init":        {"Inspect and install Terraform Providers", "Terraform Provider를 검사한 뒤 설치합니다"},
		"helox doctor":                {"Check installed release and trusted Host runtime readiness", "설치와 격리 실행 환경의 준비 상태를 확인합니다"},
		"helox config":                {"Manage CLI display settings", "CLI 표시 설정"},
		"helox config language":       {"Show or save CLI language", "CLI 표시 언어 조회·저장"},
	}
	var visit func(*cobra.Command)
	visit = func(command *cobra.Command) {
		if command.CommandPath() == "helox npm update" {
			command.Long = p.text(npmUpdateDescriptionEnglish, npmUpdateDescriptionKorean)
		}
		if command.CommandPath() == "helox cargo update" {
			command.Long = p.text(cargoUpdateDescriptionEnglish, cargoUpdateDescriptionKorean)
		}
		if command.CommandPath() == "helox go get" {
			command.Long = p.text(goUpgradeDescriptionEnglish, goUpgradeDescriptionKorean)
		}
		if command.CommandPath() == "helox terraform init" {
			command.Long = p.text(terraformUpgradeDescriptionEnglish, terraformUpgradeDescriptionKorean)
		}
		if command.CommandPath() == "helox github install" {
			command.Long = p.text(githubReplacementDescriptionEnglish, githubReplacementDescriptionKorean)
		}
		if command.CommandPath() == "helox pip install" {
			command.Long = p.text(pypiUpgradeDescriptionEnglish, pypiUpgradeDescriptionKorean)
		}
		if text, ok := descriptions[command.CommandPath()]; ok {
			command.Short = p.text(text[0], text[1])
		}
		command.InitDefaultHelpFlag()
		for key, text := range map[string][2]string{
			"help":                     {"Show help for this command", "이 명령의 사용법을 표시합니다"},
			"output":                   {"Output: auto, human or json", "출력 형식: auto(자동), human(사람용), json(자동화용)"},
			"lang":                     {"CLI language: auto, en, ko, de, fr, es, pt-BR or ja", "표시 언어: auto, en, ko, de, fr, es, pt-BR, ja"},
			"color":                    {"Color: auto, always or never", "색상: auto(자동), always(사용), never(미사용)"},
			"progress":                 {"Progress: auto, always or never", "진행 표시: auto(자동), always(사용), never(미사용)"},
			"verbose":                  {"Show bounded runtime diagnostics", "제한된 진단 정보와 검사 기록 식별자를 표시합니다"},
			"target":                   {"Installation target path", "검증한 파일을 설치할 대상 경로"},
			"replace":                  {"Replace only a proven HAA-managed asset after fresh inspection; requires --target", "독립 HAA 소유 기록이 있는 파일만 새 검사 후 교체합니다. --target을 명시해야 합니다"},
			"package":                  {"Select one existing public crate by name or name@locked-version", "기존 공개 crate 하나를 이름 또는 이름@기존잠금버전으로 선택합니다"},
			"adopt":                    {"Explicitly replace supported native project dependencies after full inspection", "지원하는 기존 프로젝트를 명시적으로 등록하고 전체 검사 후 의존성을 교체합니다"},
			"upgrade":                  {"Inspect the complete selected closure before upgrading (only-if-needed)", "선택된 전체 의존성을 검사한 뒤 only-if-needed 전략으로 업데이트합니다"},
			"source":                   {"Explicit package source", "검사에 사용할 명시적 패키지 출처"},
			"inspection-prerequisites": {"Explicit inspection-only pinned input; not installed into the target", "검사용으로만 사용할 고정된 추가 입력; 설치 대상에는 반입하지 않습니다"},
		} {
			if flag := command.Flags().Lookup(key); flag != nil {
				flag.Usage = p.text(text[0], text[1])
			}
			if flag := command.PersistentFlags().Lookup(key); flag != nil {
				flag.Usage = p.text(text[0], text[1])
			}
		}
		if command.CommandPath() == "helox pip install" {
			command.Flags().Lookup("adopt").Usage = p.text("Enroll supported native wheel files after independent ownership verification and full inspection", "공식 wheel과 기존 파일의 소유권을 대조하고 전체 검사 후 지원하는 일반 venv를 등록합니다")
		}
		if command.CommandPath() == "helox cargo update" {
			command.Flags().Lookup("adopt").Usage = p.text("Enroll supported native or changed controls after complete inspection", "지원하는 일반 프로젝트나 변경한 설정을 전체 검사 후 명시적으로 등록합니다")
		}
		if command.CommandPath() == "helox go get" {
			command.Flags().Lookup("upgrade").Usage = p.text("Inspect every selected module before upgrading one existing requirement and its imported dependencies", "선택된 모든 모듈을 검사한 뒤 기존 의존성 하나와 그 의존성을 업데이트합니다")
			command.Flags().Lookup("adopt").Usage = p.text("Enroll supported native or changed controls with -u after complete inspection", "-u와 함께 지원하는 일반 프로젝트나 변경한 설정을 전체 검사 후 등록합니다")
		}
		if command.CommandPath() == "helox terraform init" {
			command.Flags().Lookup("upgrade").Usage = p.text("Inspect and upgrade every required provider within the current version constraints", "모든 필수 Provider를 검사하고 현재 버전 제약 안에서 최신 안정 버전으로 업데이트합니다")
			command.Flags().Lookup("adopt").Usage = p.text("Enroll providerless native or changed managed controls with -upgrade", "-upgrade와 함께 Provider가 없는 일반 설정이나 변경한 관리 설정을 명시적으로 등록합니다")
		}
		for _, child := range command.Commands() {
			visit(child)
		}
	}
	visit(root)
	template := root.HelpTemplate()
	for _, pair := range [][2]string{
		{"Global Flags:", p.text("Global Flags:", "공통 옵션:")}, {"Available Commands:", p.text("Available Commands:", "사용 가능한 명령:")},
		{"Usage:", p.text("Usage:", "사용법:")}, {"Flags:", p.text("Flags:", "옵션:")},
		{"Additional help topics:", p.text("Additional help topics:", "추가 도움말:")},
		{"Use \"{{.CommandPath}} [command] --help\" for more information about a command.", p.text("Use \"{{.CommandPath}} [command] --help\" for more information about a command.", "자세한 사용법: {{.CommandPath}} [command] --help")},
	} {
		template = strings.ReplaceAll(template, pair[0], pair[1])
	}
	root.SetHelpTemplate(template)
}

func (p *presentation) localizeHelpText(text string) string {
	for _, pair := range [][2]string{
		{"Global Flags:", p.text("Global Flags:", "공통 옵션:")},
		{"Available Commands:", p.text("Available Commands:", "사용 가능한 명령:")},
		{"Usage:", p.text("Usage:", "사용법:")},
		{"Flags:", p.text("Flags:", "옵션:")},
		{"(default ", p.text("(default ", "(기본값 ")},
		{"Use \"", p.text("Use \"", "자세한 사용법: \"")},
		{"for more information about a command.", p.text("for more information about a command.", "")},
	} {
		text = strings.ReplaceAll(text, pair[0], pair[1])
	}
	return text
}

func (p *presentation) renderDoctor(report DoctorReport) error {
	return p.renderDoctorNotice(report, true)
}

func (p *presentation) renderDoctorNotice(report DoctorReport, exitNotice bool) error {
	p.reported = true
	title, color := p.text("Ready", "준비 완료"), "32"
	if !report.Healthy {
		title, color = p.text("Not fully ready", "준비 미완료"), "33"
	}
	p.line("\n╭─ Heliopause · %s", p.paint(p.text("Environment: ", "실행 환경: ")+title, color))
	for _, check := range report.Checks {
		label, explanation := p.doctorDescription(check.Name)
		marker, status := "!", p.text("UNAVAILABLE", "준비되지 않음")
		if check.Healthy {
			marker, status = "✓", p.text("OK", "정상")
		}
		p.line("│ %s %s: %s", marker, label, status)
		if !check.Healthy {
			p.line("│   %s", explanation)
			p.line("│   %s: %s", p.text("Code", "코드"), safeTerminal(check.Detail))
		}
	}
	p.line("╰────────────────────────────────────────")
	if !report.Healthy && exitNotice {
		p.line("%s", p.text("Some requirements are unavailable. Exit code 1 is intentional; this is not a fully healthy installation.", "준비되지 않은 항목이 있어 종료 코드는 1입니다. 전체 설치 정상으로 판단할 수 없습니다."))
	}
	return p.writeErr
}
func (p *presentation) doctorDescription(name string) (string, string) {
	switch name {
	case "release-installation":
		return p.text("Verified release installation", "검증된 릴리스 설치"), p.text("The verified version/manifest installation is unavailable. This is separate from the controlled runtime checks below.", "검증된 버전·설치 기록을 확인할 수 없습니다. 아래의 실행 환경 준비 항목과 별도입니다.")
	case "trusted-host-runtime":
		return p.text("Trusted runtime", "신뢰된 실행 환경"), p.text("Check the protected installed tools and runtime identities.", "보호된 설치 도구와 런타임의 준비 상태를 확인하세요.")
	case "observer-helper":
		return p.text("Isolated behavior observer", "격리 동작 관찰기"), p.text("The required installed observer helper is unavailable.", "격리 동작 관찰에 필요한 helper를 사용할 수 없습니다.")
	case "network-policy-service":
		return p.text("Network policy service", "네트워크 정책 서비스"), p.text("The configured network-policy helper is not ready. Start the controlled service before inspecting dependencies.", "네트워크 정책 helper가 준비되지 않았습니다. 의존성 검사 전에 설정된 서비스를 실행하세요.")
	case "runtime-images":
		return p.text("Pinned execution images", "고정된 실행 이미지"), p.text("The required pinned Node/Python execution images are unavailable.", "필요한 고정 Node/Python 실행 이미지를 사용할 수 없습니다.")
	default:
		return safeTerminal(name), p.text("This required capability is unavailable.", "이 필수 기능을 사용할 수 없습니다.")
	}
}
