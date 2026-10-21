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
