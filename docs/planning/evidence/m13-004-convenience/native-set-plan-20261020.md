# M13-004 — release metadata와 native set 연결 계획

기준 source `3d20ea7`이며 전체 active item은 M13-004다.
[획득 연결부](./acquisition-qualification-20261020.md)를 재구현하지 않는다.
사용자가 승인한 별도 설치 역할과 Linux/WSL2 amd64/same-runtime 범위를 따른다.
이 계획은 해당 범위 안의 다음 source 변경에 앞선 감사와 두 번째 대조다.

## 실제 baseline

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | 위치·실제 gap |
| --- | --- | --- | --- | --- | --- |
| strict release metadata 파싱·완전 bundle 확인 | YES | YES, release build/publish | YES, M13-003 범위 | YES, 기존 범위 | scripts/release-manifest/bundle.go:manifestRecords. Product reader로는 공유되지 않음 |
| user installer manifest 파싱 | YES | YES | YES, 기존 범위 | YES, 기존 범위 | releaseinstall.readManifest/selectFiles; 기존 versioned user install은 유지 |
| 서명 검사 결과를 Verification Port/Evidence에 전달 | NO | NO | NO | NO | SDK primitive만 있음; 기존 GitHub integrity verifier와 연결이 gap |
| native 전체 set을 기존 Install workflow로 검사 | 부분, 일반 graph owner 있음 | NO, self-update caller | NO | NO | InstallInspectService/InstallService, M6 entry/M4 set, evidence.Generator, promotion.LocalStaging 재사용 |
| 설치 전용 executable·protected lifecycle | NO | NO | NO | NO | 별도 구체 IPC/journal/동시 실행 계획 뒤 구현 |

M4의 set evaluator는 이미 Go/Cargo/Terraform에서도 사용하는 source-neutral
aggregate다. M6의 set evaluator는 single node만 받으므로 release 전체 set에
그 evaluator를 재사용하면 안 된다. M6 entry와 M4 aggregate를 그대로 연결한다.
기존 Staging은 GitHub intake asset/새 Run/digest copy를 이미 지원한다.
새 검사 엔진·Policy·Manifest/SBOM writer·staging 구현을 만들지 않는다.

## 수정 위치와 순서

1. `releaseinstall/installer.go:readManifest`의 기존 byte parsing/공통 validation을
   같은 package의 작은 helper로 추출한다. 기존 installer의 허용 범위·activation을
   대체하지 않는다. 새 `releaseinstall` strict product reader는 bounded1MiB JSON,
   unknown/duplicate field·trailing value·잘못된 UTF-8·중복 asset/record·size/digest/
   role/platform와 exact source/run/version을 확인한다. 반환 view는 private fields와
   defensive copies를 사용한다. 파싱 성공은 signature/Policy/설치 권한이 아니다.
2. 일반 release reader는 실제 현재8종을 읽는다. self-update selection은 Linux amd64의
   CLI/observer/network helper에 승인된 별도 `haa-release-installer-linux-amd64`
   (`release-installer`)가 함께 있는 release만 허용한다. old v0.1.0을 업데이트 가능한
   새 release로 오인하지 않는다. 새 helper의 실제 executable/build/package/manifest
   producer 연결은 해당 구현과 함께 변경하며 기존 v0.1.0 evidence는 유지한다.
3. `verification/releaseprovenance`에 실제 Verification Port adapter를 추가한다.
   기존 GitHub IntegrityVerifier로 observed/declared SHA를 확인하고 fixed SDK로
   exact subject/version/source/run을 확인한다. 두 사실을 단일 명시적 required
   authenticity check와 그 check에 bound된 Evidence로 보고한다. invalid signature는
   MISMATCH/Finding, 취소/읽기 오류는 operational failure로 남긴다. 정적 검사에
   signature 책임을 옮기거나 합법적인 digest만으로 provenance 성공을 출력하지 않는다.
4. 이후 bounded release-set resolver는 단 한 snapshot에서 metadata와 해당 플랫폼
   native files를 고정한다. metadata는 untrusted passive JSON으로만 선행 파싱하며
   그 파싱이 activation authority가 되지 않는다. 각 graph member의 exact digest/
   size/role와 서명·검사·Evidence/entry Policy 및 aggregate ALLOW가 모두 필요하다.
   원본 FD replay를 재사용한다. manifest/SBOM/runtime-images는 각 고정 schema의
   passive metadata inspection만 받으며 native ELF의 정적·동적 required 검사를
   RESOURCE 판정으로 우회하지 않는다.
5. 이 연결의 실제 caller는 기존 Application Install workflow를 재사용한다. 새 본체
   명령·성공 출력은 protected promoter가 구현되기 전에 등록하지 않는다. ordinary
   dependency install/update는 이름이 같아도 self-update로 가로채지 않는다.

## 검사 계획과 두 번째 검토

- committed v0.1.0 manifest의 기존8종 파싱/identity와 기존 installer 회귀를 보존한다.
  self-update helper 부재는 명시 거부한다. malformed/duplicate/unknown/trailing JSON,
  중복·누락·잘못된 role/platform/size/digest, path 이름과 잘못된 run/version을 거부한다.
- 실제 committed signed bundle을 Verification Port→기존 InspectService/Evidence/Policy에
  전달한다. 정상 ALLOW, checksum/signature/source/tag/name/digest mismatch의 BLOCK,
  canceled cause 보존을 확인한다. 검증되지 않은 receipt를 caller가 만들 수 없다.
- full-set wiring 때 어느 하나의 native required check가 실패/미완료면 Staging과
  Promotion을 호출하지 않는 실제 workflow 회귀를 추가한다. metadata-only 성공을
  native 전체 검역 성공으로 보고하지 않는다.
- canonical format/Quick/security/docs와 관련 Race, 새 import 범위 vulnerability를
  실행한다. Darwin compile은 native 실행과 구분한다. 종료한 전용 binaries/payload/
  operators는 삭제하고 현재 바로 소비하는 source/cache만 재사용한다.

두 번째 source 대조에서 metadata를 별도 application workflow로 중복 구현하는
방안을 제외했다. 이미 있는 complete graph/entry Policy/set Policy/staging이 해당
책임을 소유한다. signature는 Verification provider, schema는 Inspection provider,
authority는 기존 Policy/installer boundary에 둔다. 기존 generic JSON 지원 범위를
넓히거나 새로운 Core framework를 도입하지 않는다. Protected activation의 concrete
authority·journal·crash/동시 실행 계약은 여전히 다음 privilege source 변경 전 필수다.

## 다음 source 연결의 정확한 map

앞선 reader/Verification Port의 [검증 범위](./metadata-qualification-20261020.md)는
닫혔으며 다음은 위 audit에서 아직 NO인 전체 set 연결이다.

- `artifact/githubrelease/heliopause_set_unix.go`: `HeliopauseReleaseSet`이 기존
  production Client의 한 snapshot에서 exact official tag/manifest를 고정한다.
  Strict view의 version/runtime lock/gVisor identity와 새 installer를 포함한4 native
  roles를 요구하고 manifest/SBOM/runtime-images까지7개 graph member를 만든다.
  API size/digest와 각 manifest record가 다르면 거부하며 전체 download는1GiB 이하다.
  Native safety/Policy를 source에서 판정하지 않는다. 기존 FrozenIntake를 Artifact로
  위임하고 검증 provider에 immutable identity/proof만 제공한다. 원래 acquisition
  Run은 소유 inode를 기록해 Close에서 정리하며 Evidence/검사 Run은 임의 삭제하지 않는다.
- `releaseinstall/metadata.go`: 현재 producer의 fixed SBOM/runtime-images wire
  schema만 bounded passive JSON으로 읽는다. 기존 strict duplicate/key reader를
  공유하고 executable/URL 실행이나 새로운 JSON artifact 일반 지원을 추가하지 않는다.
- `inspection/githubrelease/release_set.go`: 기존 native CompositeInspector에
  위임하되 세 고정 metadata에는 해당 schema 검사를 적용한다. exact manifest의
  member/size/digest/version/source와 일치하지 않는 파일은 거부한다. Runtime image
  record는 기존 compiled runtime identity와 대조하며 artifact의 설명 문구를 authority로
  사용하지 않는다. 일반 GitHub inspector의 지원 범위는 유지한다.
- `verification/releaseprovenance/artifact.go`: 실제 resolved set의 immutable proof를
  공급하는 좁은 callback constructor로 기존 ArtifactVerifier를 연결한다. Core Port를
  확대하거나 source→verification import/새 dependency framework를 만들지 않는다.
- `scripts/check/architecture.go`: 위 실제 provider 간에 필요한 import만 명시한다.
  Domain/Application은 concrete release/SDK/OS 구현을 참조하지 않는다.

두 번째 대조: metadata를 별도 application pipeline으로 중복 구현하지 않고
manifest를 primary로 삼아 그 선언 파일을 연결하는 기존 graph를 사용한다. M6 entry,
M4 set, Evidence/Manifest/SBOM writer와 staging은 그대로다. Default tests는 fake
production-origin transport와 committed portable 자료만 사용한다. Whole-set 정상과
부분 실패의 workflow stop, drift·record mismatch·변조·취소·동시 Close·owned cleanup을
검사한다. Fake proof/source 단독 성공을 실제 새 release의 서명/동적/activation
qualification으로 전용하지 않는다. CLI와 root installer는 아직 등록하지 않는다.
