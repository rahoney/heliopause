# M13-004 — 요청별 session·완료 payload 정리·공개 명령 source map

기준 `5985999`와 [검증된 client/Promotion 후보](./native-client-qualification-20261020.md).
기존 [lifecycle](./activation-lifecycle-plan-20261020.md)/[자체 업데이트 계약](../../../user-journey-cli-ia/07-self-update.md)을
재사용한다. M13-004만 IN_PROGRESS이며 아래 MISSING만 수정한다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| frozen7/source·SDK·native/metadata inspection | YES | YES, existing pipeline | YES, local component | YES, component | 새 engine 없음 |
| per-request content binding | YES | YES | YES, local concurrent | YES, component | 세 port를 실제 source/session에 전달 |
| sealed Promotion/root fixed client | YES | YES, concrete adapter | PARTIAL, local/kernel | NO, actual signed activation | 기존 client 재사용 |
| source별 session factory/완료 payload 정리 | NO | NO | NO | NO | 아래 owner 연결 |
| 공개 self-update/7언어/progress | NO | NO | NO | NO | 아래 static preflight와 실제 binding |
| producer/native first bootstrap/실제 signed lifecycle | NO | NO | NO | NO | 이후 단위, 이번 결과로 완료하지 않음 |

## 변경 위치와 순서

1. `artifact/githubrelease.HeliopauseReleaseSet`은 기존 Acquire가 만든 replay Run의
   directory/asset inode를 기록하고 명시적 `DiscardReplayRuns(ctx)`를 추가한다.
   기존 Close가 inspection Run/Evidence를 임의 삭제하도록 의미를 바꾸지 않는다.
   실제 command runtime drain이 확인된 session 종료에서만 discard를 호출한다.
   Root와 child inode/metadata 및 exact asset-only membership을 확인하고 unknown은
   보존한다. 다른 Run이나 Evidence/active installation은 삭제하지 않는다.
2. `promotion.NativeReleasePromotion`은 실제 검증한 staging payload의 소유 inode를
   기록한다. Close는 실제 drain 뒤 그 artifacts만 제거하고 final Manifest/SBOM records는
   보존한다. Unknown replacement/추가 파일은 보존하며 처음 cleanup 오류를 유지한다.
3. `promotion.NativeReleaseUpdatePreparation`은 existing RuntimeLease와 고정 target/
   source/version/no-adopt를 확인하고 UpdateOwner를 **factory/네트워크 이전**에 잡는다.
   Bootstrap이 제공하는 inert factory는 실제 frozen source+Verification+Inspection만
   구성한다. Session은 기존 InstallUpdateContentSession을 구현해 ports를 위임하고
   native Promotion을 사용한다. Close는 actual drain→독립적으로 안전한 owned payload
   정리→source FD/original download 정리→UpdateOwner 해제 순서이며 오류를 합친다.
4. `bootstrap/self_update_linux.go`는 기존 source/SDK/ReleaseSetInspector와 M6 entry/
   M4 aggregate/Evidence/Generator/LocalStaging을 연결한다. 기존 서비스 재구현은 없다.
   Managed Linux/amd64를 tools/HTTP 이전에 검사하고 non-Linux/arm64/unmanaged는 거부한다.
   `ObserverSupervisor.Close`→Executor.Close의 첫 결과를 sync.Once closure로 보존한다.
   Executor.Close는 clientHome을 먼저 비우므로 재호출 nil을 첫 실패의 해결로 해석하지 않는다.
   Borrowed health lease는 기존 AcquireUpdateOwner에서 거부한다.
5. `cli/self_update.go`와 기존 tree/presentation/catalog에 실제 `helox self-update
   --version vX.Y.Z`를 연결한다. No positional args/임의 URL/repository/target/adopt/latest,
   bounded exact semver와 single --version을 tools/HTTP 이전에 검증한다. 기존 UPDATE
   report/JSON/Evidence/checks/progress를 사용하고 self-update의 전체7 subject를 일반
   GitHub 단일 asset 범위로 설명하지 않는다. English/Korean과 기존5언어 catalog/fallback을
   지원한다. 기술 코드는 영어로 유지하며 SDK·서버/응답 불확실성·실제 전환 완료를 구분한다.
   Stable self-update failure reasons는 controller-owned typed failure에만 추가한다.
   이미 성공한 Promotion 뒤 session cleanup만 실패하면 전체 operation은 FAILED로
   보고하되 확인된 promotion_status=COMPLETED를 보존한다. 실제 전환 후 응답 손실처럼
   성공을 확인하지 못한 경우에는 성공한 promotion을 만들어 내지 않는다.
6. 실제 source가 같은 변경에 존재한 뒤 static command/help를 등록한다. Help/settings는
   기존처럼 inert하다. Root client의 PREPARED→cleanup→SH 해제→ACTIVATE 순서와 관리자
   인증 경계를 유지한다. 이름 미정 채널을 등록하거나 개발용 설치를 자동 인수하지 않는다.

## 검사와 두 번째 대조

기존 canonical Quick/security/docs/vulnerability와 영향 package Race를 먼저 사용한다.
Owned replay/stage removal, records/Evidence 보존, repeated Close/첫 cleanup 실패와 unknown
retention을 확인한다. Session delegation/negative preflight/factory-before-owner 금지,
invalid CLI input의 no runtime/network, 일곱 언어 도움말/전체-set report/JSON stdout을
검사한다. 기존 생태계 update binding은 유지하며 실제 signed native4/systemd/crash 및
native initial bootstrap은 별도 qualification이다. 완료한 binary/operator는 마지막
사용 직후 삭제하고 현재 재사용 중인 private source/cache와 shared pinned SDK는 구분한다.

두 번째 source 대조에서 lifecycle/drain을 Bootstrap business workflow로 옮기지 않고
실제 session/cleanup Adapter가 소유하도록 했다. Factory는 Policy나 root ALLOW를
제공하지 않는다. SDK 성공만으로 검역을 건너뛰거나 cleanup 실패 뒤 lease를 넘기지 않으며,
전환 요청 후 응답 손실에서 이전 버전 보존을 단정하지 않는다. Whole-CLI sudo/새 installer
선행 root 실행/추가 public release·main merge·영구 host migration은 이 변경에 포함하지 않는다.
