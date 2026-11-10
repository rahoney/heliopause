# M13-004 — frozen native set의 기존 workflow 연결

기준 parent `9051350beac0ec2b14d936eb91eb781cf741b0ef`.
[사전 감사·정확한 수정 map·두 번째 검토](./native-set-plan-20261020.md)의 다음 gap을
구현했다. 전체 M13-004와 public CLI/protected activation은 아직 미완료다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED |
| --- | --- | --- | --- | --- |
| 한 snapshot의4 native+3 metadata graph와 FD replay | YES | YES, resolver/Artifact 및 기존 InstallService 회귀 | YES, local component | YES, 이 범위만 |
| source-bound authenticity provider·passive metadata/native 검사 분기 | YES | YES, 실제 기존 Evidence/entry M6/set M4/staging 연결 | YES, local component | YES, 이 범위만 |
| 새 signed native target의 실제 qualification | YES, 기존 dynamic backend 재사용 | NO, public caller/새 target 없음 | NO | NO |
| privileged helper/activation/복원/public CLI | NO | NO | NO | NO |

`HeliopauseReleaseSet`은 공식 production origin과 exact version, immutable
snapshot, active runtime lock/gVisor identity, manifest/API size·digest 일치를
요구한다. 원본7개를 고정하고 새 검사 Run에는 기존 FrozenIntake로 exact bytes를
복사한다. 전체 원본 download 한도는1GiB이며 기존 단일 asset 한도는 유지했다.
Close는 held FD와 실제 소유 acquisition Run만 정리한다. 다른 검사 Run/Evidence/
관련 없는 파일은 삭제하지 않으며 교체된 inode의 cleanup은 UNKNOWN으로 거부한다.

새 SBOM/runtime-images reader는 현재 producer의 고정 passive JSON schema와
source/compiled runtime identity만 확인한다. 일반 JSON 지원·metadata 실행은 없다.
ReleaseSetInspector는 exact native4개를 기존 static/dynamic CompositeInspector에
위임한다. ResolvedArtifactVerifier는 선택된 immutable proof를 기존 SDK adapter로
전달한다. 새 Application/Policy/staging 엔진이나 Core interface는 만들지 않았다.

## 실제 검사와 한계

원본 owner는 local `docs/github-logs/self-update-20261020/`다.
`release-set-final-source.json`과 `release-set-final-inventory.json`이 검사 입력 전체와
candidate bytes를 연결한다. Go1.26.9 및 기존 공유 도구/module cache를 사용했다.

- `release-set-final-checks.json`: focused2.62s, canonical Quick115.92s,
  security6.92s/vulnerability11.86s/docs0.27s, 관련 전체 Race11.11s 모두 exit0.
- 한 번의 discovery 후7 members/6 edges, lookup의 추가 HTTP0회, defensive
  proof, byte/lock/API mismatch·helper 누락·mutable release·초과·취소 거부,
  concurrent copy/Close 및 소유권 불명 cleanup 보존을 확인했다.
- 실제 기존 InstallService→Evidence→M6/M4→Manifest/SBOM writer→LocalStaging의
  연결에서7 entries/required18개/Evidence18개를 확인했다. 이 positive는 fake native
  payload/관찰 backend와 일반 digest verifier를 쓰는 **source/staging 단위 시험**이다.
  signed native self-update나 실제 sandbox 실행 성공으로 확대하지 않는다.
- 같은 전체 workflow에서 native required check 미완료 또는 production SDK의
  authenticity mismatch면 Staging/Promotion 호출0과 stage 디렉터리 무변경이다.
  실제 committed bundle은 변형된 시험 manifest/native subject를 올바르게 거부했다.
- `release-set-portable-checks.json`: Darwin 두 architecture의 GitHub artifact test
  compile 및 전이 release metadata/verifier/inspection/staging imports 범위다.
  native macOS 실행은 아니다. 완료 binary는 즉시 삭제했다.
- 최초 `release-set-focused-first.log`는 internal test→promotion→artifact의 test-only
  import cycle로 해당 package가 실행되지 못했다. Public dependency로 연결하는
  external integration test로 옮겼다. `release-set-focused-layout.json`의7.57s PASS와
  최종 동일 source의 전체 검사를 별도 기록했다. 제품 import 경계를 우회하지 않았다.

문서 전달과 exact index/commit 증거는 release-set-delivery 자료가 소유한다.
Go temporary test payload/stages는 test cleanup에서 삭제했고 source/cache만
바로 이어지는 동일 item 구현에 재사용한다. 실제 설치된 CLI/helper/config 변경은0이다.

## 다음 gap과 재검토

별도 설치 역할·global runtime lease/handoff·durable journal·helper 인증 전환·
실패 복원·old controller cleanup·새 실행본 작동 검증이 남는다. 해당 privileged
source 이전에 concrete IPC/transaction/복구 계획을 확정한다. old peer의 기본
constructor는 expected digest가 빈 root-protected inode 인증이며, 새 설치 경계의
signed digest 재확인은 별도 책임임을 감사 문서에 정확히 정정했다.
