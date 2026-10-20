# M13-004 — 실제 요청별 session과 공개 self-update 명령 연결

Parent `fd7d44b`. [사전 source map/재검토](./session-cli-plan-20261020.md)를 따랐다.
M13-004는 IN_PROGRESS이며 실제 새 signed native4/systemd/crash acceptance는 닫지 않는다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED |
| --- | --- | --- | --- | --- |
| 요청별 frozen source/authenticity/native inspection/Promotion session | YES | YES, 기존 InstallUpdateService | PARTIAL, local/negative/cleanup | NO, 실제 healthy native9 base 필요 |
| exact-version self-update 명령/7언어/help/progress/report | YES | YES, 실제 Bootstrap 서비스 | YES, local syntax/negative/UI | PARTIAL, 실제 signed activation 미완료 |
| helper asset producer/최초 native bootstrap·registration | NO | NO | NO | NO |

`NativeReleaseUpdatePreparation`은 기존 lease/controller와 고정 source/target/version을
확인하고 factory/획득 전에 UpdateOwner를 잡는다. Bootstrap의 inert factory는 기존
HeliopauseReleaseSet/ResolvedArtifactVerifier/ReleaseSetInspector를 구성한다. M6 entry,
M4 aggregate, Evidence/Generator/LocalStaging 순서는 그대로다. Session은 실제 runtime
drain 이후에만 owned replay/staging payload를 제거하고 기록·근거를 보존한다.

Source/CLI/session은 같은 bounded exact semver validator를 사용한다. Public command는
`helox self-update --version vX.Y.Z`이며 임의 repository/URL/target/adopt/latest나 중복
version을 runtime/HTTP 이전에 거부한다. Platform과 managed native controller도 tools/
HTTP 이전에 확인한다. Whole CLI sudo나 개발용 설치의 자동 인수는 없다.

기존 typed UPDATE 결과를 사용하며 전체7 subject를 단일 GitHub asset으로 설명하지 않는다.
전환/복구 오류는 이전 버전 보존을 단정하지 않는다. 설치가 확인된 뒤 guard cleanup만
실패하면 전체 FAILED와 확인된 promotion COMPLETED를 함께 보존한다. EN/KO 설명은
완전하며 다른5언어의 핵심 도움말·scope·progress와 주요 상태 설명을 추가했다.
나머지 technical 설명은 기존 English fallback을 사용하고 JSON/codes는 고정된다.

원본은 `docs/github-logs/self-update-20261020/session-cli-*`다.

- 동일 전체 input의 canonical format/focused3.71s/Quick111.28s/security8.25s/
  vulnerability11.92s/docs0.19s, 기존 runtime/Promotion/acquisition 전체 Race125.40s,
  Application/CLI 전체 Race24.59s가 모두 exit0이다.
- Native session의 concurrent Close/첫 오류 보존, unknown runtime에서 실제 stage
  payload 무삭제, 독립적으로 안전한 cleanup, managed owner 없는 factory 미호출과
  typed cause 보존을 확인했다. 실제 valid native lease의 Begin positive는 아직 없다.
- Actual new CLI의7언어 help, latest 거부, EN/KO 미관리 설치 거부10 cases는 예상 exit와
  출력이 일치했다. Root installation/state는 전후 ABSENT_CONFIRMED이고 acquisition
  cache는 생기지 않았다. 실제 설치/서비스/config를 변경하지 않았다.
- Darwin Bootstrap test compile은 amd64 5.29s/arm64 5.23s/exit0이다. Native macOS 실행이
  아니다. CLI42,624,375B와 Darwin23,540,320B/21,741,938B 및 전용 probe directory는
  완료 직후 삭제했다. 현재 private source/cache는 다음 producer 검사에 재사용한다.

두 번째 대조에서 help의 inert 동작, same-runtime 한계, default generic ports의 요청별
교체, cleanup 실패 후 SH handoff 금지, 과거 설치본과 source의 차이를 확인했다.
다음은 실제 helper producer/native bootstrap·registration과 새 signed 전체 activation/
systemd/crash qualification이다. Package names는 이름 의존 채널에 별도로 남는다.
