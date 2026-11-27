# P2c5-B — PNPM frozen 검사 문맥의 production 연결

2026-10-24, baseline54362e01d41a2e1740e0b48d00232b9d069b6fb7.
[A의 실제 prerequisite](./p2c5-pnpm-offline-context-plan-20261024.md#8-a3-peer-fixture--착수-감사수정-지도두-번째-검토)를
통과한 같은 M13-004/P2 작업이다. M13-004 하나만 IN_PROGRESS로 유지한다.

Read: P2c3/P2c4/A 전체 지도, Architecture02/03, Domain08/12/13,
artifact/npm의 pnpm_selection/project_snapshot/lock, application/project_inspect와
dependency_batch_inspect, inspection/npm의 static/composite/dynamic,
sandbox/backend/pnpm_resolver/pnpm_project_transport 및 PyPI의 기존 static-all 소비자.

## 1. Source 작성 전 baseline audit

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| PNPM full lock/독립 exact source/Domain snapshot | YES | YES | NO | NO | 로컬+선택 actual은 검증; 전체 P2 qualification은 미완료 |
| acquire/verify 전체→adapter batch→record/Policy 공용 owner | YES | YES | YES | YES | 기존 projectAwareInspection 재사용; Application에 PNPM 해석 추가 안 함 |
| npm static 및 기존 단일 npm dynamic | YES | YES | YES | YES | 기존 경로 유지; PNPM dependency/peer context는 없음 |
| native store/metadata/offline layout 의미 | YES | YES(test) | NO(full) | NO(full) | transitive2/peer4 primitive 실제 PASS; production owner와 다중 instance 동적 coverage 미구현 |
| frozen lock+controls+runtime+acquired/instances의 typed 연결 | NO | NO | NO | NO | B1 아래의 immutable 값과 producer 연결 |
| PNPM static-all/per-instance dynamic/검증 layout 이송 | NO | NO | NO | NO | B2 이후; A 성공이나 B1 값 검증으로 완료 처리 금지 |
| Host staging/records/transaction/CLI | NO | NO | NO | NO | 검역 연결 뒤 순차 수행; P2 완료 전 광고 안 함 |

기존 single Backend.Execute는 그대로 둔다. 새 batch capability가 없을 때 단일 검사로
조용히 fallback하여 transitive 검사를 완료했다고 처리하지 않는다. 새 optional capability는
현재 PNPM 작업에서 필요한 소비자 경계이며 Domain/Policy에 native lock schema를 넣지 않는다.

## 2. B1의 정확한 수정 위치와 방법

1. artifact/npm/pnpm_inspection_context.go: source-bound PNPMSelection의
   InspectionContext(install, controls, runtimeIdentity)가 기존 ProjectSnapshot을 **재사용**하여
   PNPMInspectionContext를 만든다. 원래 controls와 source-bound lock·runtime·snapshot을
   private 필드로 보존한다. Zero/unbound/변조 controls·빈 runtime은 기존 검증에서 거부된다.
2. MatchesSnapshot은 graphDigest뿐 아니라 source/target/mode/dependency-free,
   정렬된 control digests와 정확한 ResolvedArtifact(URL/SRI 포함) 목록도 비교한다. 같은
   digest를 복사해 다른 Domain snapshot을 만든 입력을 승인하지 않는다.
3. BindAcquired(snapshot, map[node]AcquiredArtifact)는 위 일치와 전체 원소 수/정확한 node,
   identity/SRI/observed SRI/비어 있지 않은 SHA/size와 canonical intake:RunID:tarball handle,
   고유 run을 요구한다. 반환 PNPMInspectionInput은 immutable 사본이며 verification/Policy
   권한이 아니다. 실제 파일의 byte digest는 introduction owner가 다시 확인해야 한다.
4. Input.InstancesFor(exact acquired subject)는 lock의 package metadata와 모든 matching
   peer-qualified snapshots를 돌려준다. Unique archive와 복수 instance를 합치지 않는다.
   Getter는 slice/map 내부 저장소를 노출하지 않고 zero/foreign subject는 거부한다.
5. sandbox/pnpm_resolver.go: BindInspectionContext가 기존 source binding을 한 번 수행하고
   generated runtime identity와 자기 controls를 위 constructor로 전달한다. 기존
   BindProjectSnapshot은 이 값을 재사용하는 wrapper가 되어 caller 계약을 유지한다.
   같은 source를 두 번 resolve하거나 artifact adapter→runtimeidentity 의존을 만들지 않는다.

B1은 값 결합과 실제 producer 연결만 검증한다. Runner/관찰기·네트워크·permission·Policy,
기존 npm composite의 동작과 CLI는 바꾸지 않는다. B2/실제 backend가 완성될 때까지
현재 public quarantine fixture의 transitive INCOMPLETE는 그대로 남는다.

## 3. B1 검증과 두 번째 검토

기존 snapshot fixture·정상 producer tests를 재사용한다. 정상/explicit empty/peer4와
synthetic six archives/seven instances를 구분한다. 후자는 source-binding test double의
구조 검사이며 public registry 또는 actual multiple-peer dynamic 증거가 아니다.

직접 부정 입력은 same-digest 다른 target/mode/controls/URL/SRI/dependencies,
missing/extra/wrong node, 다른 identity/variant, SRI/observed drift·0size·invalid/공유 handle,
zero context와 foreign exact subject다. getter copy·호출자 map 변경 뒤 값 불변을 확인한다.
Formatter/Quick/security/docs와 npm/sandbox 관련 tests/Race를 적용하고 실제 public primitive를
변경하지 않았으면 A3 actual을 이유 없이 반복하지 않는다. 새 runner가 연결될 때 C4의
single/transitive/peer/selected-update 실제 소비자를 다시 검증한다.

두 번째 검토에서 opaque graph hash 하나만 비교하면 Domain constructor로 위조한 다른
dependency 목록이 통과할 수 있음을 확인했다. 전체 normalized 값 비교도 포함한다.
Acquired 메타데이터의 일치는 byte verification을 대신하지 않으며 `Input.Valid`를 ALLOW나
cache/staging 권한으로 쓰지 않는다. 기존 단일 backend의 교체·재작성은 필요하지 않다.

## 4. B2 이후의 순서와 필수 수명/coverage 경계

- Inspection의 PNPM 전용 project adapter가 기존 ProjectInspectService의 batch capability를
  소비한다. 완전한 acquired set 결합 뒤 static-all을 수행한다. 하나라도 finding/incomplete이면
  전체 dynamic을 시작하지 않고 모든 entry에 required dynamic 미실행 상태를 명시한다.
  M3가 단순히 전달된 required checks를 순회하므로 dynamic check 누락으로 clean entry를
  완료로 보이는 방법은 금지한다. Static Finding/Evidence는 보존한다.
- Sandbox의 production prepare owner는 qualified PNPM/Node/project volumes·exact command·
  early store config·typed policy를 재사용한다. Native opaque store를 직접 생성하지 않는다.
  Metadata 단계는 scripts0이며 artifact 실행 전 network detach/endpoint0을 증명한다.
  Typed owner/sole attachment/complete observation/cleanup 실패면 결과를 폐기한다.
- Artifact 실행은 별도 fresh ephemeral session에서 수행하고 metadata control profile을
  artifact lifecycle 권한으로 바꾸지 않는다. Native output의 layout은 opaque 성공만으로
  신뢰하지 않고 exact archive identity/bytes·instance/edge·symlink/bin/허용 경로를 검증해야 한다.
  이송 방법·pinned npm lifecycle command 의미는 구현 전에 실제 source와 추가 대조한다.
- Unique archive가 여러 peer instance를 가지면 **모든** expected instance 관찰이 필요하다.
  각 session의 실제 ID·원래 오류/관찰을 보존하고 가짜 aggregate SandboxSession을 만들지
  않는다. Missing/duplicate/foreign/partial instance를 단일 성공 결과로 축약하지 않는다.
  Required npm dynamic 결과의 최종 aggregate/Evidence 형식과 정확한 edit map은 이 B1 값
  결과 및 실제 lifecycle primitive 검증 뒤 확정한다. 미확정 실행 방식을 먼저 interface로
  고정하거나 기존 required lifecycle을 optional로 내리지 않는다.
- Host layout/records/transaction/CLI는 이 후속 검역 경계가 닫힌 다음이다. 원격 push는
  의미 있는 연결 단위로 묶고 CI가 필요할 때 동일 profile 실측에 따른 실제 타이머를 사용한다.

완료한 실제 payload/진단 source·중복 binary는 all-UID refs/query errors0 뒤 삭제한다.
현 production/checker source·고정 공급 archive·다음 소비자가 필요한 cache만 유지한다.

## 5. B1 로컬 결과와 다음 실행 경계 검토

첫 canonical format/Quick131.191s/security6.977s/docs.330s는 PASS다. 기본 tests가 평소보다
길었으나 전체 PASS이며 지연 원인은 미확정이다. 후속 exact subject의 다른 RunID 거부와
matching instance만 map-copy하는 보완 뒤 관련 npm/sandbox 전체 Race22.133s도 PASS다.
기존 ProjectSnapshot constructor/단일 Backend·관찰기·Policy는 유지한다. 새 값/producer의
로컬 검증이며 실제 PNPM dynamic/전체 P2의 qualification은 계속 미완료다.

다음 실행 방식 검토에서 고정 npm11.19.0/tag commitbfacd33의 rebuild가 directory spec으로
정확한 node.path를 선택하고 regular dependency의 preinstall/install/postinstall을 실행함을
확인했다. Link node는 prepare도 실행하며 script approval 설정의 skip 경로가 있으므로
exit0만으로 대상 coverage를 인정하지 않는다. 기존 observer의 IsExactNpmLifecycleArguments는
기존 install argv만 승인하고 SharedObserver의 NPM mount contract에도 PNPM project volume이
없다. 따라서 임의 rebuild/extra mount를 기존 권한에 끼워 넣지 않는다. 별도 실제 lifecycle
primitive의 exact argv·target selection·root/dependency scripts 무실행·관찰/이송 수정 지도를
확정한 뒤 구현한다. 기존 NPM admission을 일반화하거나 stream 경고를 무시하는 방법은 제외한다.

최종 B1 format/Quick53.953s/security6.966s/docs.273s와 관련 전체 Race22.133s는 PASS다.
B1의 IMPLEMENTED/WIRED=YES, LOCAL VERIFIED이며 QUALIFIED/ACCEPTANCE_CLOSED는 전체
P2 기준 NO다. 입력 결합만으로 dynamic check를 완료로 만들지 않았다. 다음 B2는 실제
target lifecycle primitive와 필요한 exact admission/mount 계약을 먼저 문서화·재검토한다.
