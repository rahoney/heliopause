# M13-004 — frozen release 획득의 구현 전 대조

기준 source `6b4c82748c9fc0b4ec245e84e4342269aa128bfd`.
전체 active item은 계속 M13-004다. [provenance primitive](./provenance-qualification-20261020.md)는
로컬 검증됐지만 본체 workflow에 아직 연결되지 않았다. 이 문서는 다음 실제 gap의
수정 위치·방법과 second review를 기록하며 future milestone의 scaffold를 만들지 않는다.

## Responsibility audit

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | 실제 위치와 gap |
| --- | --- | --- | --- | --- | --- |
| 한 exact GitHub asset의 Resolve/Acquire | YES | YES | YES, 기존 M6/SEC-UPD | YES, 기존 범위 | `artifact/githubrelease/client.go`: Resolve/Acquire/release/downloadAsset, release.go parser. API ID·size/digest·bounded HTTPS와 credential/proxy 차단 재사용 |
| 여러 asset을 단 한 release response에 고정 | NO | NO | NO | NO | 현재 Resolve를 파일별 호출하면 live metadata를 다시 읽음. 새 snapshot이 필요하며 기존 단일-asset API는 유지 |
| public attestation의 credential 없는 획득 | NO | NO | NO | NO | production source 없음. actual API audit와 SDK verifier는 이 download 구현의 대체물이 아님 |
| 이미 획득한 native bytes를 새 Inspect Run에 전달 | NO | NO | NO | NO | 기존 InspectService·intake handle·static/dynamic·Evidence/Policy 재사용. 다운로드/resolve를 다시 수행하지 않는 adapter가 gap |

## 실제 API 대조와 구현 결정

공개 manifest의 exact digest로 고정 API version2026-03-10을 요청해 익명·no-proxy
HTTP200을 실제 확인했다. API는 inline bundle 대신 `repository_id`와 `bundle_url`을
반환했다. URL host는 `tmaproduction.blob.core.windows.net`, path는
`/attestations/<repository-id>/...`다. blob은 JSON text가 아니라 **raw Snappy block**이며
bounded decode 결과는 보존한 실제 signed bundle과 의미적으로 같았다. 이는 TLS 응답과
형식의 audit이며 출처 검증·검역·승격 권한이 아니다. 상세 hash/정리 범위는 local
`docs/github-logs/self-update-20261020/public-attestation-acquisition-audit.json`이 소유한다.
URL의 SAS query는 repository에 남기지 않았고 완료한 body/operator도 삭제했다.

기존에 pin된 `github.com/klauspost/compress v1.18.6`의
`snappy.DecodedLen`/`Decode`를 사용한다. [upstream source](https://github.com/klauspost/compress/tree/v1.18.6/snappy)는
Snappy block API이고 표준 library에는 해당 decoder가 없다. 별도 module/version을
추가하거나 decoder를 재구현하지 않는다. decoded length를 먼저 확인해 allocation을
제한하고, 새 reachable consumer의 vulnerability를 다시 확인한다.

## 수정 위치와 순서

1. `internal/artifact/githubrelease/client.go`의 existing Resolve에서 Domain identity를
   만드는 작은 helper만 공유한다. discovery/stream/digest/redirect 책임은 유지한다.
2. 같은 package에 `ReleaseSnapshot`과 `Client.FreezeRelease`를 추가한다. 한 번의
   release GET에서 bounded asset set의 ID/name/tag/size/digest/state/publication과
   immutable 속성을 고정한다. 각 선택은 기존 parser를 거친다. 이름·ID 중복, 초과,
   draft/미게시·잘못된 tag를 거부한다. snapshot 내부 map/slice는 노출하지 않는다.
   나중에 `ResolveAsset`은 이 snapshot만 읽으며 HTTP를 호출하지 않는다.
3. snapshot에 속한 asset에 대해서만 attestation을 획득한다. URL을 사용자 입력으로
   받지 않고 API endpoint는 frozen owner/repo/digest로 생성한다. API는 redirect를
   따라가지 않는다. 최대16개 attestation/2MiB API 응답, 전체90s 안에 처리하며
   pagination URL을 임의로 따라가지 않는다. inline JSON 또는 공개 production Azure
   blob만 받는다. blob은 HTTPS/정확한 host·port 없음·userinfo/fragment 없음·row의
   numeric repository ID path prefix를 요구하고 redirect를 금지한다.
4. blob encoded/decoded 각각1MiB 한도와 Snappy DecodedLen 사전 검사를 적용한다.
   원격 URL/query/HTTP 원본 오류를 결과나 로그로 노출하지 않는다. HTTP200이나
   repository_id는 authenticity가 아니며 이후 고정 Sigstore 검증이 필수다.
5. 같은 package에 frozen-intake Artifact adapter를 추가한다. reference/resolved/acquired
   identity·declared/observed digest·size를 서로 확인하고 기존 run-scoped handle만
   읽는다. source FD와 digest를 고정하며 새 Inspect Run으로 copy하는 바로 그 바이트를
   해시한다. 불일치·link/path/identity 변조·Close 후 요청은 거부한다. 새 resolve/download
   fallback은 없다. Close는 소유 FD만 정리하고 caller 소유 intake를 임의 삭제하지 않는다.
6. `InspectService`와 동일 static/dynamic inspector가 이해하는 기존 intake handle을
   생성한다. 현재 Application/Policy/CLI 명령·root installer의 의미는 바꾸지 않는다.
   전체 release metadata·검역·activation orchestration 연결은 다음 gap으로 유지한다.

## 검사와 second review

- 실제 한 번의 release GET 이후 server metadata를 바꿔도 snapshot identity는 그대로이며,
  snapshot lookup은 추가 HTTP0회임을 확인한다. duplicate name/ID, wrong tag/draft/
  publication/oversize·과도한 set과 zero snapshot을 거부한다.
- production-origin 조건을 적용한 fake HTTP transport로 API→blob request/credential
  부재와 timeout/redirect/악성 host·path·userinfo·token 비출력을 확인한다. 실제 공개
  API 읽기는 opt-in/별도 evidence이며 default test의 network/개인 경로 입력이 아니다.
- committed signed bundle에서 만든 Snappy fixture를 decode하고 원본과 비교한다.
  truncated/malformed·decoded-size bomb·과도한 rows/API/blob·잘못된 repository ID를
  거부한다. source 획득과 SDK signature/Policy는 각각의 증거로 판단한다.
- frozen-intake는 기존 InspectService의 실제 결과/Evidence 연결, 복사 후 digest·size,
  source path/file 교체·변조·unknown reference·Close 뒤 요청과 cleanup을 검사한다.
- canonical format/Quick/security/docs를 먼저 적용하고 신규 compression import의
  vulnerability, 관련 Race와 실제 public read/검증을 별도로 확인한다. 기존 GitHub
  inspect/install/update 소비자를 보존하며 검사가 끝난 전용 payload/operator는 삭제한다.

두 번째 대조 결과: existing Resolve/Acquire를 반복 호출하는 안은 frozen discovery가
아니므로 제외했다. inline JSON만 가정하는 안은 실제 Azure/Snappy 응답과 달라 제외했다.
generic URL reader·ambient auth·무제한 decode·attestation row의 repo ID를 trust authority로
쓰는 안도 제외했다. 기존 parser/intake/InspectService를 재사용하며 별도 Policy나 root
권한을 source adapter에 넣지 않는다. 본체 activation 성공을 반환하는 stub은 만들지 않는다.
