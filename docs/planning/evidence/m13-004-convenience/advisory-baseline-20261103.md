# S05 선택 graph의 알려진 취약점 — 구현 전 baseline

기준 `a4e4f15`, active item은 M13-004 하나다. S04의 공급 검토와 독립적으로
[설계 §6.2](../../21-goal-completion-design.md#62-s05--선택된-사용자-graph에-알려진-취약점을-대조한다)의
선행08–09를 사용하며 S04가 통과했다는 전제는 두지 않는다.

| 항목 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| S05 사용자 dependency advisory | NO | NO | NO | NO | 고정 endpoint의 bounded batch/record 조회, 정확한 mapping·전체 graph 환원, 완전성/Evidence/Policy·유한 cache/refresh, 실제 CLI 정상/차단·실패 원본 보존 |

## 기존 구현과 최소 변경 지도

- `scripts/check/security.go:runVulnerability`는 Heliopause의 `./...`만 govulncheck한다.
  사용자 package graph를 검사하지 않으므로 제품 scanner로 복사하거나 재명명하지 않는다.
- `InstallInspectService.inspectGraphAware`와 `ProjectInspectService`는 이미 고정 graph와
  acquired node 집합을 갖는다. `dependencyAwareInspection`/기존 report·check plan을
  사용하며 두 번째 resolver/설치 graph를 만들지 않는다. PyPI `InspectGraph`의 static-all/
  closure-aware dynamic 순서, derived wheel의 후속 per-artifact 검사도 유지해야 한다.
- `ports.Inspection`/`CheckPlanner`가 검사 선언·실제 결과를 연결한다. 공통 advisory
  adapter 한 곳이 기존 검사에 결과를 합치고 graph에서 같은 key를 한 번만 조회한다.
  생태계별 HTTP client·advisory parser를 중복 생성하지 않는다. 첫 단위는 실제 사용할
  bounded OSV 조회·정규화 경계이며, 전체 wiring 전에는 제품 기능 완료로 쓰지 않는다.
- `M3/M5/M6` 및 project-root Policy가 최종 판정을 소유한다. 특히 M6는 알 수 없는
  Finding을 BLOCK하므로 새 review code를 무작정 넣지 않는다. 정규화된 affected high/
  critical·명시 악성은 BLOCK, 그 외 영향/평가 불명은 review로 분류하는 공통 해석을
  검증하며 현재 integrity/구조·동적 필수 체크와 더 강한 BLOCK 우선순위를 보존한다.
- 기존 `ResolverPolicy`/Root service는 HAA Docker bridge의 public IPv4 TCP443 allowset을
  검증한다. Host의 일반 HTTP client나 OSV용 URL/method/payload 검증기는 아니다.
  Registry/resolver의 guest endpoint 목록에 OSV를 더해 권한을 상속시키지 않는다.
  Advisory adapter의 입력은 package/version query와 record ID라는 두 typed operation만
  받고 고정 `https://api.osv.dev`의 POST `/v1/querybatch`, GET `/v1/vulns/{id}`로 만든다.
  임의 URL/header/proxy/credential·redirect·Artifact 실행 권한을 입력으로 받지 않는다.
  Host HTTP는 Inspection I/O owner에 두며 guest network 정책과 별도다. Host 목적지의
  public 주소·TLS/응답/시간 한도도 확인하고 production transport를 임의 client로 대체하지 않는다.
- Cache는 C01의 유한 저장소 owner에 연결한다. 별도 영구 사본/daemon/무한 memoization을
  만들지 않으며 complete response+정확한 request/provider/schema/hash/시각만24h 재사용한다.
  조회 실패/미지원은 zero-result가 아니며 expire된 결과로 설치를 승인하지 않는다.

## 첫 조회 단위의 입력·출력 계약

1. npm 계열→`npm`, public PyPI→`PyPI`, Go module→`Go`, public Cargo crate→`crates.io`.
   기존 canonical identity를 검증한다. PyTorch local build/다른 공급자·Terraform/GitHub
   asset을 유사 이름만으로 매칭하지 않으며 exact mapping이 없으면 unsupported다.
2. 한 요청의 유한 key 집합을 안정된 순서로 deduplicate하고 그 index를 유지한다.
   Batch는 ID/modified만 돌려주므로 record를 따로 읽는다. 모든 page token을 소진하며
   결과 수·ID/modified·중복·token 순환·전체 request/byte/record 한도를 검증한다.
3. [공식 API](https://google.github.io/osv.dev/post-v1-querybatch/)는 응답 순서를 입력과
   같게 보장하지만 항목에 package identity를 echo하지 않는다. 클라이언트는 exact request
   index와 response hash를 묶고 record의 affected package를 교차 검증한다. 빈 항목 둘을
   바꾼 응답의 순서를 자체 증명할 수 있다고 주장하지 않는다. 정상 TLS provider 계약과
   decoder의 검출 범위를 분리한다.
4. [record](https://google.github.io/osv.dev/get-v1-vulns/)의 ID/modified/withdrawn과
   [OSV schema](https://ossf.github.io/osv-schema/)의 affected package/range/severity를
   확인한다. Query API는 version을 fuzzy matching할 수 있으므로 조회에 나타났다는
   이유만으로 local version/다른 source의 실제 영향 버전을 확정하지 않는다. 평가 불명은
   review 또는 incomplete로 남기고 정상 판정으로 바꾸지 않는다.
5. Live provider·취득 시각·요청/응답 hash·record modified를 연결한다. 존재하지 않는
   global DB snapshot/version을 만들지 않는다. 일반 record의 상세 설명이나 URL을
   명령·출력 제어·네트워크 입력으로 실행하지 않는다. 요약은 bounded 정규화만 한다.

표적 검증은 batch membership·page/ID·부분 응답·같은 ID의 modified 변경·withdrawn/
alias·malformed/duplicate JSON·초과 한도·cancel/timeout/redirect/Host 주소·출력 주입부터
시작한다. 이어 기존 실제 graph·Policy·Evidence/store/CLI·target 보존을 묶어서 검증한다.
단위 성공은 전체 S05나 16번 종료가 아니다. 새 CI/Host 설치는 시작하지 않았다.

## 첫 단위 — bounded batch/page 조회의 로컬 구현

`internal/inspection/advisory/osv.go`/`osv_json.go`에서 고정 POST 요청·동일 key 조회 병합·
원래 node 순서 환원·미완료 key만 다음 page 조회를 구현했다. 누락/null row·중복/모호한
JSON·잘린 응답·같은 key의 ID 중복·token 순환·같은 ID의 modified 충돌·HTTP/TLS 작업
실패·redirect·취소는 성공 결과를 반환하지 않는다. 확인한 응답은 request/response hash와
취득 시각에 연결하며 실패 앞의 사실은 별도로 반환한다. 아직 record의 affected/withdrawn/
severity 해석을 하지 않으므로 ID 목록을 최종 vulnerability finding이라고 쓰지 않는다.

신규 advisory I/O 단위의 유한 상한은 입력4096개·message1000개·전체64요청/16MiB/
8192 ID references, message2MiB·개별 row1000 references, 전체30초/HTTP10초다.
Message1000은 이 client의 chunk 한도이며 provider가 문서화한 최대 query 수라는 주장은
아니다. 같은 key를 병합한 뒤 각 원래 node에 독립 결과 slice를 돌려주며 비용은 입력·
실제 응답 크기에 선형이다. 기존 Artifact/observer/runtime·CI 한도를 늘리지 않았다.

현재 HTTP transport는 private 호출부에 명시적으로 공급한 시험 double이다. Cookies와
redirect 정책은 상속하지 않지만 **production public-address/TLS transport·record GET·
실제 graph mapping·필수 check/Evidence/Policy·cache/refresh·CLI 연결은 MISSING**이다.
반환값은 완성된 사용자 검사 보고나 설치 승인이 아니다. 새 package를 canonical architecture
owner의 빈 import 허용 목록에 등록해 concrete backend/bootstrap/외부 의존성 우회를 막았다.
기존 모듈의 역할이나 보안 예외를 늘리지 않았다.

로컬 Race4.501s·Staticcheck(advisory0.899s/checker2.418s)·vet0.623s·canonical
foundation55.335s와 security14.955s는 PASS다. 실제 network/provider 응답·전체 S05 검증은
아니며 기존 CPP/native/CI를 반복하지 않았다. 작은
[부분 구현 원장](./advisory-batch-local-20261103.json)이 정확한 source와 check 범위를 소유한다.
