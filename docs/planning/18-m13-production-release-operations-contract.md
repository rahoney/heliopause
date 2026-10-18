# M13 Entry Decision — Production Release & Operations

M13은 M10에서 구현한 verified distribution chain, M11에서 완성한 richer dynamic
detection, M12에서 확장한 ecosystem support를 실제 public release로 전환하는
마지막 운영 milestone이다.

M13 이전에는 release workflow와 installer를 구현·검증할 수 있지만 실제 public
release activation은 하지 않는다.

M13 진입 조건:

```text
M12 Ecosystem Expansion qualification complete
AND
M12-02 final red-team fix gate closed
AND
no known release-blocking finding
```

---

## 1. Canonical distribution model

Heliopause의 canonical trust source는 **GitHub Verified Release**다.

GitHub Release는 다음을 소유한다.

- native `helox` binaries
- Linux privileged/network helper
- gVisor observer helper
- release manifest
- checksums
- SBOM
- runtime-image manifest
- OS-native package assets where applicable
- GitHub Artifact Attestation / protected workflow provenance

HAA-owned runtime container image가 필요한 경우 GHCR에서 **immutable digest**로
참조하며 mutable tag를 trust identity로 사용하지 않는다.

편의 설치 채널은 canonical release와 별개의 구현을 배포하지 않는다.
동일 version/manifest/digest에 bind된 bootstrap 또는 package만 배포한다.

---

## 2. First public release channels

첫 public release의 설치 채널은 다음으로 고정한다.

| Channel | Purpose | Contract |
| --- | --- | --- |
| GitHub Verified Release | canonical / strongest bootstrap path | manifest + checksum + attestation 검증 |
| npm | Node 사용자용 convenience installer | protected GitHub Actions에서 publish, canonical release version/digest에 bind |
| PyPI / pipx | Python 사용자용 convenience installer | protected GitHub Actions에서 publish, canonical release version/digest에 bind |
| `.deb` | Debian/Ubuntu 계열 native install | same protected release build, attested asset |
| `.rpm` | Fedora/RHEL/Rocky/Alma 계열 native install | same protected release build, attested asset |
| Homebrew | macOS convenience install | exact GitHub Release asset/version/SHA-256에 bind |

첫 release에서 제외:

- generic `tar.gz` installation path
- Snap
- Flatpak
- AUR/pacman
- FreeBSD/OpenBSD package
- Docker-only `helox` distribution

배포 채널이 많아져도 기능 지원 OS가 자동으로 확대되는 것은 아니다.
각 Host의 실제 security capability는 `helox doctor`와 qualification matrix가
truthfully 보고한다.

---

## 3. Bootstrap assurance tiers

### Tier A — canonical verified install

```text
GitHub Verified Release
→ release manifest/checksum
→ GitHub attestation/signer workflow
→ exact native asset/helper/runtime identity
→ atomic activation
→ helox doctor
```

가장 강한 bootstrap 경로다.

### Tier B — OS-native package

`.deb`, `.rpm`은 protected release workflow에서 동일 source commit으로 build하고
release manifest와 attestation에 포함한다.

OS package는 다음 system integration을 설치할 수 있다.

- native `helox`
- Linux helper
- systemd unit
- protected configuration path
- required directory/mode ownership

패키지 설치 성공을 HAA readiness 성공으로 간주하지 않는다.
마지막 판정은 항상 `helox doctor`다.

### Tier C — convenience channel

npm, PyPI/pipx, Homebrew는 “일단 설치해 보고 싶은 사용자”의 friction을 낮추기 위한
경로다.

원칙:

- protected GitHub Actions에서만 publish
- long-lived registry publishing secret을 가능하면 사용하지 않고 OIDC/Trusted
  Publishing을 사용
- package version은 canonical GitHub Release SemVer와 일치
- package 내부에 독립적인 unverified HAA implementation을 만들지 않음
- exact release asset URL/version/SHA-256 또는 manifest identity에 bind
- publish provenance가 불확실하면 해당 channel release를 중단

편의 경로가 canonical GitHub verified path보다 강한 trust를 제공한다고 주장하지 않는다.

---

## 4. Release activation boundary

- `release` environment는 required reviewer 승인 없이는 publish job을 실행하지 않는다.
- `v*` tag는 생성·수정·삭제 Ruleset으로 보호한다.
- tag commit은 protected `main` history에 속해야 한다.
- exact tag SHA의 canonical `Required` check가 성공해야 한다.
- tag commit은 successful `Heliopause Release Build` source commit과 일치해야 한다.
- 개발 변경은 `feature → develop → main` PR 흐름을 사용한다.
- main 직접 push, force-push와 우회 merge를 허용하지 않는다.
- main Ruleset은 `license/cla`와 `Required` status check를 필수로 한다.

---

## 5. Final publication contract

최종 publish 순서:

```text
M12 qualification complete
→ M12-02 red-team gate closed
→ protected settings active
→ protected release tag
→ exact successful candidate build
→ exact source/main/Required provenance verification
→ manifest/checksum/attestation verification
→ draft GitHub Release
→ draft exact asset name/size/digest verification
→ immutable public GitHub Release
→ post-publish release/asset verification
→ OS/convenience channels publish
→ clean Host install/doctor/smoke
→ publication evidence + incident/quarantine record
```

tag·run·manifest·asset·signer identity 중 하나라도 불일치하거나 GitHub API,
environment approval, release immutability, package provenance 또는 post-release
verification이 불확실하면 public release를 ALLOW하지 않는다.

public 이후 검증 실패는 기존 release의 overwrite/re-tag가 아니라
`PUBLISHED_BUT_QUARANTINED` incident로 처리하고 새 SemVer release로 수정한다.

---

## 6. Work breakdown

| Order | ID | Scope | Status |
| --- | --- | --- | --- |
| 1 | M13-001 | protected main/develop/tag/release environment/immutable release activation | COMPLETE |
| 2 | M13-002 | `feature → develop → main` 실제 protected PR qualification | ACCEPTED_WITH_EXCEPTION — C3 / full QUALIFIED=NO |
| 3 | M13-003 | canonical release build·manifest·attestation·OS package assets | COMPLETE — protected v0.1.0 build/attestation/package lifecycle |
| 3a | CLI-UX-001 | multilingual terminal progress and structured human results | COMPLETE — [local qualification](./evidence/m13-cli-ux/qualification-20261018.md) |
| 3b | CLI-UX-002 | OS preferred display-language auto | COMPLETE — [qualification](./evidence/cli-os-language/qualification-20261018.md); native macOS/CI328 PASS |
| 3c | SEC-UPD-001 | guarded update-command coverage | IN_PROGRESS — npm first; detailed reviewed plan |
| 3d | CLI-UX-003 | branded interactive launcher | NOT_STARTED — after SEC-UPD-001 acceptance |
| 4 | M13-004 | npm·PyPI/pipx·Homebrew convenience publication | BLOCKED — SEC-UPD-001 and owner package-name decision; accounts ready / CLI-UX-002 complete |
| 5 | M13-005 | exact publication·clean Host bootstrap·doctor·cross-ecosystem smoke·quarantine | NOT_STARTED |

### M13-001 — protected activation

- protected `main`, `develop`, `v*` tag Ruleset과 `release` environment를 실제
  repository 설정으로 활성화한다.
- required reviewer와 immutable release policy를 구성한다.
- main Ruleset은 `license/cla`와 `Required`를 필수 check로 요구한다.
- environment·Ruleset·tag 보호 상태를 read-only probe로 다시 확인한다.
- 하나라도 불확실하면 다음 단계로 진행하지 않는다.

### M13-002 — protected branch qualification

실제:

```text
feature
→ PR
develop
→ PR
main
```

흐름에서:

- Required CI
- CLA
- up-to-date requirement
- conversation resolution
- force-push block
- direct push block

이 실제 merge를 통제하는지 qualification한다.

hostile cases:

- stale branch
- missing Required
- missing CLA
- direct main push
- bypass merge
- force-push

는 모두 실패해야 한다.

#### 2026-10-17 승인된 C3 조건부 유예

실제 missing-CLA 시험은 NOT_TESTED / DEFERRED로 보존한다. 소유자가
유지보수자 본인 변경으로 진행하는 초기 릴리스 범위의 증거 공백을 명시적으로
수용하여, 이 한 항목의 M13 후속 작업·초기 릴리스 선행 차단을 유예한다.
기존 S 판정은 유지하며 C2의 준비 전용 순서 예외를 이 C3 범위로 대체한다.

M13-002는 IMPLEMENTED=YES / WIRED=YES / QUALIFIED=NO (전체 hostile matrix) /
ACCEPTANCE_CLOSED=YES (승인 예외 범위)인 ACCEPTED_WITH_EXCEPTION으로 기록한다.
실제 미서명 PR 차단이나 독립 freshness-only의 미검증 결과를 PASS로 바꾸거나,
M13 전체의 무조건적인 full qualification을 주장하지 않는다.

필수 CLA 검사·서명 의무·Strict·대화 해결·직접 push/force 금지·관리자 무우회
보호는 유지하며 실제 missing/failed CLA PR은 병합하지 않는다. 가짜 신원/기여,
status 주입이나 서명 초기화로 시험을 만들지 않는다. 최초 외부 기여를
병합하기 전에 실제 미서명 기여자의 차단 시험을 완료하고, 그 전에는 외부
기여 병합을 보류한다. 실제 차단 실패 또는 보호 설정·App 신원 불명확이
확인되면 유예를 중단하고 후속 병합·배포를 멈춘다.

M13-003..005는 Queue의 선행조건과 각 acceptance를 따라 순차 진행한다.
이 승인은 특정 PR 병합·release version/source/tag·public publish 실행을
지정하는 승인이 아니다. 구체적 후보와 검증 결과를 준비한 뒤 기존 reviewer·
배포 승인 절차를 따른다. Namespace·소유권·OIDC, 실제 build/attestation,
clean-host 설치·rollback과 publication 검증은 여전히 필수다.

유예는 최초 외부 기여 병합 전 실제 시험으로 종료한다. 초기 릴리스 이후에도
시험이 불가능하면 후속 릴리스에 이 승인을 자동 확대하지 않고 당시 증거와
보호 상태로 범위를 다시 판단한다. 종료 시 승인 예외·미검증 상태·원본 이력을
함께 보존한다. 정확한 승인 입력과 결과는
[C3 승인 기록](./evidence/m13-002-protected-path/cla-deferral-proposal-20261017.md),
기존 S/C2 이력은 [결정 기록](./evidence/m13-002-protected-path/remaining-decisions-20261017.md)이 소유한다.

### M13-003 — canonical release build and OS packages

release build는 exact protected source에서 다음을 만든다.

Core assets:

- `helox` native binaries
- gVisor observer helper
- Linux network-policy helper
- runtime-image manifest
- release manifest
- SBOM
- checksums

추가 OS assets:

- `.deb`
- `.rpm`

기존 “candidate는 항상 10개” 같은 숫자를 여러 문서에 중복 고정하지 않는다.
**canonical release manifest가 exact release asset set의 단일 source of truth**다.

release workflow는:

```text
exact source commit
→ deterministic/bounded build
→ package build
→ manifest exact asset set
→ SHA-256
→ SBOM
→ GitHub Artifact Attestation
```

을 생성한다.

`.deb`/`.rpm` 설치 script는 최소 권한으로 다음만 수행한다.

- versioned native asset install
- required Linux helper install
- systemd unit/configuration install
- correct owner/mode
- post-install `helox doctor` 안내

패키지 lifecycle script에서 network download나 arbitrary shell extension을 하지 않는다.

Linux amd64 package payload는 `/usr/lib/heliopause/releases/<version>/`에 CLI,
observer, network-policy helper와 runtime lock/image manifest를 root 소유로
설치한다. CLI·observer는 `0555`, privileged helper는 `0500`, metadata는
`0444`이며 `/usr/bin/helox`는 해당 version의 CLI를 가리킨다. 제공 systemd
unit는 기본 비활성이고, 실제 caller identity로 보호한
`/etc/heliopause/network-policy.json`이 있어야 시작할 수 있다. 추측한 UID/GID의
동작 config를 생성하거나 기존 active runsc/observer를 교체하지 않는다.
고정 post-install 안내는 controlled runtime 설정 후 `helox doctor` 실행을
요구한다. OS package 설치 성공과 verified runtime activation은 별도다.

Checksum 생성과 publish의 file set은 같은 manifest verifier를 소비한다.
Publish는 manifest attestation/source를 먼저 검증한 뒤 basename·role·platform·
size·digest·metadata·exact directory set 검증이 성공한 목록만 별도 control
file로 전달한다. Checksum 파일의 임의 경로나 artifact 자체의 file list를
publication authority로 취급하지 않는다.

### M13-004 — convenience publication

#### npm

목적:

```bash
npm install -g <official-package>
```

같은 쉬운 bootstrap UX 제공.

요구:

- official package namespace/name 확정
- protected GitHub Actions only
- Trusted Publishing/OIDC 우선
- exact HAA release version과 1:1
- canonical GitHub asset/manifest digest pin
- publish provenance 기록
- arbitrary postinstall network behavior 금지
- bootstrap에 필요한 최소 bounded logic만 허용

#### PyPI / pipx

목적:

```bash
pipx install <official-package>
```

또는 명시적으로 지원하기로 한 pip-based bootstrap UX 제공.

요구:

- protected GitHub Actions only
- PyPI Trusted Publisher/OIDC 우선
- exact HAA release version과 1:1
- canonical manifest/digest pin
- package install 시 user credential 요구 금지
- native/bootstrap contract 외 별도 HAA implementation 금지

#### Homebrew

목적:

```bash
brew install <official tap/formula>
```

요구:

- formula는 exact GitHub Release version URL과 SHA-256을 pin
- mutable `latest` URL 금지
- release update는 protected automation/manual review를 거침
- macOS에서 지원하지 않는 Linux dynamic capability를 설치 성공으로 오인시키지 않음
- 설치 후 `helox doctor`가 실제 capability를 truthful하게 보고

### M13-005 — exact publication and clean Host qualification

GitHub draft 생성 후 **public 전** canonical release manifest의 exact asset set에 대해:

```text
name
size
sha256 digest
upload state
attestation
```

을 모두 검증한다.

그 뒤에만 immutable public release로 전환한다.

post-publish:

- `gh release verify`
- all release asset verification
- registry/package publication verification
- package version/digest/provenance binding

을 수행한다.

---

## 7. Clean Host bootstrap qualification

최소 representative matrix:

| Path | Representative Host | Required result |
| --- | --- | --- |
| GitHub Verified Release | clean Ubuntu supported Host | verified bootstrap → `helox doctor` |
| `.deb` | clean Ubuntu/Debian | install → helper/systemd → doctor |
| `.rpm` | clean Fedora/Rocky/Alma representative | install → helper/systemd → doctor |
| npm convenience | clean supported Host with Node/npm | bootstrap → exact canonical version → doctor |
| PyPI/pipx convenience | clean supported Host with Python/pipx | bootstrap → exact canonical version → doctor |
| Homebrew | clean macOS | install → truthful doctor / supported CLI capability |

source checkout이나 local Go build는 public bootstrap qualification에 사용하지 않는다.

---

## 8. Cross-ecosystem smoke

Linux fully-qualified Host에서 first release가 다음 public source path를 실제로 smoke한다.

```text
helox npm ...
helox pip ...               # PyPI
helox pip ... --source ...  # official PyTorch
helox go ...
helox cargo ...
helox terraform init
helox github ...
```

각 path는 최소:

- one known-good public artifact
- exact identity/digest Evidence
- ALLOW/Promotion success
- one bounded hostile/tamper case
- cleanup success

를 확인한다.

`doctor`와 smoke 결과 중 하나라도 support matrix와 다르면 release를 중단한다.

---

## 9. Public bootstrap boundary

최종 사용자는 source checkout이나 local Go build 없이 다음 중 하나에서 시작한다.

```text
GitHub Verified Release
npm
PyPI/pipx
.deb
.rpm
Homebrew
```

canonical verified path는 GitHub Release다.

README의:

```text
go build
go run ./scripts/check ...
```

는 개발자·검증자용 경로일 뿐 public installation contract가 아니다.

---

## 10. Release evidence and incident handling

release별로 최소 다음 identity를 보존한다.

- SemVer tag
- source commit
- protected main relationship
- Required CI run
- release build run
- release publish run
- manifest SHA-256
- exact release asset set
- artifact attestations
- SBOM
- GHCR image digest if applicable
- `.deb`/`.rpm` digest
- npm/PyPI/Homebrew published version binding
- clean Host qualification result

public 이후 이상이 발견되면 기존 immutable release를 수정하지 않는다.

```text
PUBLISHED_BUT_QUARANTINED
→ 사용자 경고
→ 원인 분석
→ 새 source commit
→ 새 SemVer
→ 전체 release gate 재실행
```

---

## 11. Handoff

M10의 verified release implementation과 M11/M12 qualification evidence는 그대로
보존한다.

M13은 기능 개발 milestone이 아니다.

M13 진입 후 새로운 ecosystem이나 주요 product feature를 추가하지 않는다.
release-blocking bug가 아닌 개선 아이디어는 post-release backlog로 넘긴다.

Current status: M13-001 configuration is complete. M13-002 protected normal
PR35/36 delivery and actual main CI324 ten SUCCESS are verified; the explicitly
approved disposable nonFF method passed within its recorded scope and temporary
resources are deleted. [Post-delivery evidence](./evidence/m13-002-protected-path/post-delivery-review-20261017.json)
owns those facts. M13-002 remains BLOCKED on the actual missing-CLA case, with QUALIFIED=NO /
ACCEPTANCE_CLOSED=NO. The owner approved bounded operational Strict evidence and
the preparation-only work-order exception above. M13-003 baseline audit and
implementation/build/package preparation may proceed; publication/tag/publish
activation remains prohibited by the actual CLA gate. M13-004/005 are NOT_STARTED.


Historical pre-maintenance handoff (superseded as current execution authority):

Status: IN_PROGRESS — M13-001 configuration complete; M13-002 develop conversation
rejection and normal PR31 merge verified. The single approved PR32 request was
rejected HTTP405 for expected required checks; independent stale enforcement is
not qualified. Further mutations stopped. Remaining negative cases, develop-to-main
path and final qualification remain open; exact evidence is owned by the Queue.
Read-only rule-suite analysis now identifies the proposed update and failed Strict
required-check rule; the original independent-stale qualification remains open.
The Queue links the minimal remaining execution and approval plan. No release or
additional remote mutation is authorized by that planning update.
Concrete local candidates/action limits are now recorded through the Queue;
no remote write occurred. With no actual unsigned contributor available, the
missing-CLA hostile case is NOT_TESTED/deferred, not waived or qualified.
Other checks may proceed, while this unchanged acceptance gap requires actual
evidence or an explicit canonical disposition before M13-002 closure. Final
main merge and publication remain separately gated; M13-003..005 are NOT_STARTED.

M12/M12-02 technical acceptance and PR29/30 delivery are complete. Main
CI38328099413 is SUCCESS. The owner authorized live protection configuration on
2026-10-13. Work-item scope, acceptance and current missing work are owned by
[current work in the Queue](./02-current-work-queue.md#m13-002--protected-branch-qualification).
The no-known-release-blocking-finding condition remains subject to disposition
of guest Go patch, Race fixtures and advisory follow-ups. Protection settings
alone do not authorize publication or close that release-readiness condition.
