# P1b — uv 고정 공급·runtime 연결 baseline

기준 source `b710b5606cca905fb5ffd28561c7ebd4bf4a030a`. M13-004/P1만 IN_PROGRESS다.
P1a의 검증된 parser/graph를 재작성하지 않는다. 기존 PR38/CI360은 해당 source를 검사 중이다.

## 독립 baseline 판정

| 책임 / 기존 owner | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| P1a native 선택·인증 graph, pypi/uv_lock.go | YES | NO | NO | NO | production 공급·resolver·CLI, 원격 CI |
| HAA release 서명 검증, releaseprovenance.Verify | YES | YES | YES | YES | HAA 정책 변경 없음; UV publisher 검증은 별도 |
| Public Sigstore offline root·고정 library | YES | YES | YES | YES | 같은 root/암호 primitive 재사용, 중복 root owner 금지 |
| UV image provenance·index→amd64 binding | NO | NO | NO | NO | 실제 signed fixture·엄격 publisher/subject/workflow/source·digest/size/중복 거부 |
| UV runtime pin/maintenance/provision/admission | NO | NO | NO | NO | lock schema migration·generated identity·manifest·updater·actual gVisor |

Read: [전체 계획](../../20-package-client-quarantine-plan.md),
[자료 경계](../../../user-journey-cli-ia/09-package-client-quarantine.md),
[유지보수](../../../engineering/05-runtime-upgrade-maintenance.md),
[trusted tooling D-012](../../../threat-model/04-trusted-tooling-and-evidence.md),
[Artifact Port](../../../domain-model/07-contract-artifact-port.md).

## 수정 순서와 영향

1. 기존 releaseprovenance package 안에 UV publisher 전용 좁은 검증을 추가한다.
   기존 HAA Verify·tag/source/workflow 권한은 수정하지 않는다. 고정 Sigstore/root를
   재사용하며 owner/workflow를 사용자 설정 regex로 받지 않는다. 실제 UV0.13.4 bundle과
   OCI index/amd64 manifest를 portable fixture로 남긴다. Signed index와 child manifest의
   전체 bytes hash/size·유일한 linux/amd64 선택을 연결한다. 서명은 안전성/실행 승인이 아니다.
2. 실제 소비자는 기존 scripts/update-runtime의 명시적 offline verify-uv-image 단계다.
   단일 외부 디렉터리의 선언된 입력을 bounded read하고 검증된 identity 영수증만 출력한다.
   네트워크·Host 실행·runtime pin 변경은 없다. 기존 maintenance 계획의 자료 검증으로 사용한다.
   scripts/check architecture는 이 소비자의 기존 verifier import 하나만 허용한다.
3. 검증된 공급 identity를 확보한 뒤 runtime lock/generator/manifest/updater/drift를 연결한다.
   기존 Python3.14.7/Bookworm/ABI는 유지하며 UV distroless tool만 공급한다. 공식 Python
   derived Trixie는3.14.8이므로 대체하지 않는다. Artifact권한 없는 root-owned private
   tool mount·exact admission을 기존 lifecycle 안에 연결한다. 포괄 unknown exec 허용 금지.
4. Runtime manifest는 현재 네 image exact set을 검증하므로 새 UV 기록은 해당 생산자와
   releaseinstall.ValidateRuntimeImages·bootstrap/activation consumers를 함께 갱신해야 한다.
   같은-runtime self-update 거부를 느슨하게 하지 않는다. 배포 전 새 canonical runtime으로
   최초 설치·업데이트 경계를 별도로 검증한다. 과거 signed same-runtime 결과를 전용하지 않는다.

## 사전 두 번째 검토

공식 문서 예시 build-docker.yml을 실제 signer라고 단정하지 않는다. 실제 bundle의
미인증 payload는 release.yml caller·publish-docker-image.yml builder·workflow_dispatch를
가리킨다. Certificate/서명·공식 exact source에서 확인하기 전은 관측일 뿐이다.
Release version label은 서명된 index가 묶은 config에서 확인해야 하며 label만 신뢰하지 않는다.
Index digest와 child digest를 혼동하거나 live tag 재resolve로 frozen 자료를 바꾸지 않는다.

검증: canonical format/quick/security/docs, 기존 HAA signed verification 회귀, 실제 upstream
bundle offline 성공과 wrong publisher/source/run/digest/index-child/platform·변조·과대·누락 실패.
Production UV invocation/관찰기 수정 뒤에는 현재 영향을 받는 실제 resolver·wheel/sdist·npm·
GitHub/CPU/CUDA 소비자 matrix를 따로 닫는다. 지금 verifier 성공으로 닫지 않는다.
완료한 시험 bulk는 마지막 소비 뒤 삭제하며 작은 fixture·hash·결과만 보존한다.


## P1b 공급 metadata 검증의 로컬 결과

실제 UV0.13.4 signed index의 offline 암호 검증과 amd64/config 연결이 통과했다.
[영수증](./p1b-uv-supply-local-qualification-20261021.json)이 exact source와 canonical
Quick36.83s/security4.78s/docs0.20s 및 실제 command11.705s의 범위를 소유한다.
HAA/UV publisher 교차 거부, altered proof/source/run/version/digest·platform/config
chain 부정 회귀도 포함한다. 첫 nil Context 직접 호출 테스트는 Staticcheck SA1012로
거부됐으며 그 불필요한 직접 호출을 제거한 최종 후보의 결과와 구분한다.

Supply verifier IMPLEMENTED=YES/WIRED=YES(유지보수 offline command)/QUALIFIED=NO/
ACCEPTANCE_CLOSED=NO다. Runtime/CLI는 아직 미연결이고 새 source 원격 CI는 남는다.
P1a sourceb710b56/CI360은 다른 검사와 별개로 Go1.26.9의 reachable GO-2026-6809 때문에
Vulnerability가 FAIL이다. 공식 Go1.26.10 패치가 필요하며 suppression하지 않는다.
P1a 완료 bulk19항목/57,489,010B는 마지막 소비 뒤 참조0/query errors0을 확인해 삭제했다.
활성 P1b source/cache와 아직 owner 확인 중 preview만 다음 소비를 위해 유지한다.


## P1b2 — c2c1750에서 이어가는 공급·pin baseline

Go 보안 선행조건은 최종 로컬 source/component 검증과 커밋·push를 마쳤다.
원격 CI361/CPU 및 cu126→cu132→cu130의 독립 qualification은 확인 중이다.
같은 branch에 새 source를 push해 진행 중 PR CI를 취소하지 않는다.

| 책임 / 현재 owner | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| UV native parser/graph, releaseprovenance.VerifyUVImage | YES | YES(자료/유지보수) | NO | NO | production native runtime/CLI |
| 실제 signed OCI tool bytes | YES(조사) | NO | NO | NO | bounded layer→UV file 인증 API와 maintenance consumer |
| UV lock/generated identity·지원 검토·배포 manifest | NO | NO | NO | NO | 명시 schema2·UV owner·생성·drift·release consumer |
| 기존 runtime lifecycle/network/helper admission | YES | YES | YES(기존 scope) | YES(기존 scope) | UV tool 전용 admission·실제 resolver |

서명한 index→amd64/config의 실제 첫 layer19,395,988B와 expanded42,908,672B의
SHA256/diffID가 일치했다. UV file42,544,544B의 SHA256은
 de77a01a86452cc465a734257b5a2073d41115d1b8bbfe84ac636af53851b499다.
UVx는361,608B다. 두 regular file은 root0:0/0755이고 Host에서 실행하지 않았다.
이 수치는 runtime lock 하나로 채택해야 하며 code 곳곳의 version/hash literal로 복제하지 않는다.

### 정확한 수정 순서

1. 기존 releaseprovenance owner에 인증된 OCI layers의 bounded tool 추출을 추가한다.
   모든 layer의 compressed digest/size와 config diffID·순서를 검증하고 마지막 layer의
   overwrite/whiteout도 확인한다. UV/UVx regular files와 io directory만 허용한다.
   총 compressed/expanded 각64MiB, entries32 이내, root ownership·static ELF amd64·
   정확한 version/image binding을 확인한다. 이미지 서명은 Policy나 실행 승인과 구분한다.
2. 실제 소비자는 기존 scripts/update-runtime이다. UV exact-version plan은 state 안에
   official proof/고정 layer inputs를 보존하고 capture를 재검증한다. Offline render/apply는
   같은 bytes를 다시 확인하며 tag·layer를 다시 resolve하지 않는다. Same-version은 기존
   pin을 보존하고 network0이다. 외부 layer file은 size/hash/경로·소유권/변경을 검사한다.
3. runtime lock schema2의 UV image/version·index/source/invocation·native identity 한 owner를
   generator→runtimeidentity와 runtime version/support checks에 연결한다. Python3.14.7·
   Bookworm/ABI와 다른 image pins는 유지한다. runtime-image manifest의 새 UV 기록과
   releaseinstall.ValidateRuntimeImages·합성 fixture binder·release/bootstrap consumers를
   함께 갱신한다. 공개 HAA custom image를 만들지 않으며 Astral 공식 image만 고정한다.
4. 기존 sandbox runner의 RunOutput/RunInput과 docker cp를 재사용해 시작하지 않은
   고정 공급 image의 /uv만 검증·전송한다. Python container의 root-owned private tool
   mount를 UV profile에서만 bounded64MiB로 준비한다. Host uv 실행·임의 bind mount는 없다.
   observer.cc의 고정 path/mount/UID·nonce/generation 입장 검증에 UV class를 좁게 연결한다.
   원래 pip/Python roles와 ARTIFACT 권한은 유지한다. 진짜 native startup/네트워크·cleanup을
   확인한 뒤 uv install/upgrade/sync caller를 연결한다.

### 두 번째 검토

Signed index만 검증하고 layer를 생략하거나 마지막 layer의 tool 교체를 무시하지 않는다.
Archive/native hash를 unsigned label·INSTALLER로 추정하지 않는다. state의 pinned blob이
없거나 바뀌면 apply를 거부한다. 새 compiled runtime tuple과 old installed tuple의 self-update
불일치 거부는 그대로 유지한다. UV 전용 tmpfs 증가는 기존 소비자의 예산 상향이 아니다.
각 단계에서 canonical format/Quick/security/docs와 malformed/substitution/oversize 회귀를
확인하고, 실제 공급·resolver/observer/CLI는 각각 qualification을 닫는다. 현재 다른 client
placeholder나 UI 지원 예시는 만들지 않는다.

## P1b2 native tool 공급의 로컬 종료

[Native tool 영수증](./p1b-uv-tool-local-qualification-20261021.json)은 c2c1750 기준의
모든 layer compressed SHA/size·config diffID·root ownership·static ELF와 최종 파일 hash를
연결한다. 실제 공식 두 layer의 realcorpus와 유지보수 v2 command가 PASS다. Synthetic
default 회귀는 별도 structural 범위이며 publisher 검증으로 표시하지 않는다.

Canonical Quick/security/docs와 기존 다섯 target 및 새 UV layer target의 fuzz가 PASS다.
실제 유지보수 호출의 relative/missing/substituted/symlink/oversize 자료는 receipt를 출력하지
않고 거부된다. Native tool 공급은 IMPLEMENTED=YES/WIRED=YES(offline maintenance)/
QUALIFIED=NO/ACCEPTANCE_CLOSED=NO다. Production pin·격리 실행·CLI·원격은 남는다.
마지막 실제 공급의 중복 추출 UV 파일은 다음 소비에 필요하지 않아 삭제 대상으로 분류하며
인증된 layer 원본은 다음 frozen-plan/runtime 소비가 끝날 때까지 보존한다.

## P1b3 — b89cb0a 단일 pin과 유지보수 연결 전 점검

기존 lock schema1/generator·runtime-image-manifest·releaseinstall exact4·support drift·
updater는 IMPLEMENTED/WIRED=YES다. UV pin/maintenance는 NO이며 기존 owner를 확장한다.
Schema2에 uv_image(reference/uv_version/architecture/provenance/tool)를 추가하고 generator는
고정 identity만 출력한다. Release image manifest v1은 같은 entry 구조의 exact5로 확대한다.
현재 compiled tuple은 old4를 계속 거부한다. 과거 signed fixture를 편집하지 않고 unsigned
file-mechanics binder만 UV entry를 덧붙인다. 기존 node/python/go/rust pin과 ABI는 유지한다.

유지보수의 uv 변경은 --uv-proof/--uv-layers의 명시 absolute 입력을 offline 인증한 뒤
state 안의 고정 파일로 복사한다. Plan에는 proof bytes와 layer digest/size만 묶고 render/apply는
state의 실제 모든 layer를 재검증한다. 거대한 blob을 plan에 넣거나 live tag를 다시 조회하지
않는다. Same-version은 입력 flag를 거부하고 네트워크0/원래 pin 보존이다. 기존 apply의 stale
source/journal/checked source는 재사용한다. Runtime pin 선택은 sandbox 호환성 완료와 다르다.

정확한 영향: generate-runtime-lock, runtime-image-manifest, check/runtime_versions,
release-manifest의 lock-schema reader, update-runtime의 selectedVersion/renderChanges/
UV metadata, testutil/releasefixture 및 releaseinstall.ValidateRuntimeImages다. Go bundle capture의
lock-schema reader도2로 갱신한다. Bootstrap/activation UV readiness와 CI pull은 실제 UV
provision/admission과 함께 닫는다. Canonical Quick/security/docs/workflow/release-gate/
freshness와 schema1/unknown/missing/substitution·same-version/no-network·frozen tamper
회귀를 검사한 후 actual frozen plan/apply/verify를 별도 실행한다.

## P1b3 단일 pin·유지보수의 로컬 종료

[Pin/maintenance 영수증](./p1b-uv-pin-local-qualification-20261021.json)의 후보는 canonical
Quick/security/docs/workflow/release-gate/freshness와 실제 서명 자료의 native 공급·변조
회귀가 PASS다. 실제 developer CLI로 합성 이전 pin0.0.1에서 공식0.13.4의 plan/apply/
verify를 실행해 세 owner 파일만 변경하고 canonical7개를297.281s에 통과했다. 합성 이전
버전의 공개 존재나 실제 UV 실행·호환성 증거로 쓰지 않는다. Migration_complete=false다.

현재 pin/maintenance IMPLEMENTED=YES/WIRED=YES(개발 도구)/QUALIFIED=NO/
ACCEPTANCE_CLOSED=NO다. UV runtime provision/admission과 production CLI는 남는다.
Go1.26.10 원격 cu126은 sample-late FAIL이며 기존 CPU/macOS/Required의 성공을 보존한다.
[조사 계획](./go12610-cu126-diagnostic-plan-20261021.md)의 timing/OS 관측은 아직 미구현이다.
완료한 중복 추출 UV42,544,544B는 참조/query errors0 확인 뒤 삭제했다. 유지보수 fixture의
source/state/binary도 작은 원본을 보존한 뒤 정리한다. 활성 SDK/cache·qualified runtime과
다음 runtime 소비가 필요한 두 upstream layer만 보존한다.
