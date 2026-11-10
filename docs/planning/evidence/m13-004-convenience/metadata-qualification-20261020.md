# M13-004 — authenticity Port와 strict manifest reader

기준 parent `5af57e9`. [사전 감사·수정 위치·검사·두 번째 검토](./native-set-plan-20261020.md)를
따라 기존 installer parsing과 GitHub digest verifier, InspectService/Evidence/M6를 재사용했다.
전체 M13-004는 IN_PROGRESS이며 release 전체 graph/보호된 설치 교체/CLI는 미완료다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED |
| --- | --- | --- | --- | --- |
| strict product manifest reader | YES | YES, 고정 metadata inspector | YES, local | YES, 이 단위만 |
| digest+공식 provenance Verification Port | YES | YES, 기존 InspectService/Evidence/Policy 회귀 | YES, local actual signed fixture | YES, 이 단위만 |
| native 전체 set·protected activation | NO | NO | NO | NO |

`releaseinstall.ParseReleaseManifest`는 기존 wire DTO/공통 validation을 공유한다.
기존 Installer.Install/activate/selectFiles의 권한·허용 범위를 대체하지 않았다.
strict reader는 bounded UTF-8 JSON의 duplicate/unknown/case-alias/trailing value와
exact roles/platform/records/version/source/run을 검사한다. 결과는 private view와
복사된 file list다. 기존 v0.1.0의8종은 읽되 별도 설치 helper가 없는 release의
NativeUpdateFiles는 거부한다. helper의 실제 build/producer 연결은 아직 없다.

`releaseprovenance.ArtifactVerifier`는 기존 IntegrityVerifier와 fixed-root SDK를
하나의 required `release-authenticity` check로 연결한다. subject 이름·digest·repo·
version/source/run이 모두 맞아야 VERIFIED다. 실패는 normalized MISMATCH/Finding,
취소는 원래 cause다. 고정된 manifest만 읽는 ReleaseManifestInspector를 추가했으며
일반 GitHub JSON 지원이나 native 동적 검사 범위는 완화하지 않았다.

원본 owner는 local `docs/github-logs/self-update-20261020/`다. 입력은
`native-set-final-source.json`, 최초 source와 실패는 `native-set-first-source.json`,
`native-set-quick.log` 및 `native-set-corrected-quick.log`에 보존했다.
첫 Quick은 Staticcheck S1016에 따라 동등 struct literal을 형 변환으로 고쳤다.
두 번째 Quick은 기존 httptest의 TCP socket가 sandbox에서 차단돼 실패했다.
skip/한도/제품 변경 없이 동일 입력을 허용된 호스트에서 다시 검사했다.

- 최종 focused1.66s exit0. 실제 signed manifest의 Port→schema→Evidence2개→M6
  ALLOW와 invalid proof의 BLOCK, source/tag/run/name/digest/declared mismatch,
  immutable proof와 cancellation 회귀를 확인했다.
- `native-set-host-checks.json`: canonical Quick24.40s/security5.20s/
  vulnerability11.17s/docs0.22s, 관련 전체 Race3.69s 모두 exit0.
- `native-set-portable-checks.json`: Darwin amd64/arm64의 releaseinstall/
  provenance/GitHub inspection test compile 범위를 기록한다. 완료 출력은 즉시
  삭제하며 native macOS 실행이나 새 release activation 증거로 전용하지 않는다.
- 문서 전달 검사는 `metadata-delivery-checks.json`, exact index/commit 연결은
  `metadata-delivery-index.json`과 `metadata-delivery-commit.json`이 소유한다.

두 번째 대조에서 signature가 inspection 책임으로 옮겨가지 않았고, 실제 기록된
check/Evidence와 Policy의 required 검사 해석이 일치함을 확인했다. old installer와
일반 GitHub 소비자는 canonical Quick에서 보존했다. 다음 gap은 단일 snapshot의
manifest/native/관련 metadata 전체 graph를 이 workflow에 연결하는 것이다.
