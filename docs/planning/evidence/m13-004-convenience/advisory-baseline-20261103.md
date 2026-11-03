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

## 두 번째 단위 — 상세 record와 공유 예산

`osv_record.go`에서 고정 GET과 batch의 ID/modified 대조·OSV1.x schema·withdrawn/alias/
affected package·version/range/severity의 bounded facts를 읽는다. 아직 해당 version의 실제
영향 여부나 severity 판정을 만들지 않는다. Cookie/redirect/HTTP 읽기 제한은 POST와 GET이
한 owner를 사용하고, 상세 record는 ID별 한 번만 읽어 원래 query에 돌려준다. 전체30초/
64요청/16MiB를 공유하며 상세 조회마다 새 예산을 부여하지 않는다. 실패한 완전 응답의 hash도
보존하며 변경된 modified·잘못된 subject·잘린/모호한 record는 complete 결과가 아니다.

새 record decoder가 struct field를 해석하므로 JSON 중복 검사는 ASCII 소문자화 대신
Go JSON과 동일한 Unicode simple-fold 동치도 거부한다. Alias와 range/event를 bounded
facts로 유지하고 description의 자유 텍스트나 remote URL을 실행 입력으로 쓰지 않는다.

또한 기존 첫 단위의 중복 조회 병합만으로는 노드별 결과 복사의 총량을 제한하지 못했다.
현재 record 후보에서 이전 fanout 논리만 복원한 작은 counterexample0.743s는4096개 중복
node/3references의 확대를 거부하지 못해 실패했다. 현재 owner의 원래 순서 환원에 기존
8192 references 상한을 함께 적용했으며 더 큰 예산/별도 정리 owner는 추가하지 않았다.
이는 실제 OOM/공개 provider 실행이 아니라 해당 메모리 상한의 controlled 직접 회귀다.

해당 후보의 전체 advisory Race4.995s·Staticcheck0.624s·vet0.384s는 PASS다.
[Record 부분 원장](./advisory-record-local-20261103.json)이 원본 counterexample/현재 source와
검증 범위를 소유한다. 이전55.335s foundation/security를 새 record 소스의 전체 검사로
전용하지 않으며 다음 의미 있는 연결 묶음에서 필요한 검사를 모은다. **전체 S05의 네 상태는
여전히 NO**다. Production public transport·실제 package/version mapping/영향 해석·
check/Evidence/Policy/cache/CLI 연결과 실제 provider 검증은 계속 MISSING이다.

## 세 번째 단위 — 정확한 package/version 영향 해석

`osv_impact.go`는 namespace/name을 대조하고 explicit versions와 introduced/fixed/
last_affected/limit의 포함·제외/재도입 구간을 평가한다. Unsupported ordering과 영향 정보
부재는 UNKNOWN, provider subject 불일치·잘못된 구간은 incomplete다. Withdrawn은 별도
상태이며 안전성/다른 advisory 부재의 증명이 아니다. PyPI local build를 public release로
투영하지 않는다. 알려진 explicit match가 있는 경우와 평가 불명 데이터를 구분한다.

Python은 기존 `artifact/pypi/reference.go`의 grammar/정규화를 그대로 재사용한
`ComparePublicVersions`에서 [공개 PEP440 순서](https://packaging.python.org/en/latest/specifications/version-specifiers/#summary-of-permitted-suffixes-and-relative-ordering)를
비교한다. Epoch/release zero padding·pre/post/dev·큰 정수 비교를 하며 다른 parser나
Python 실행 의존성을 만들지 않았다. Cargo의 ASCII name 문법도 기존 ParseReference로
검증하고 Unicode lookalike를 registry alias로 취급하지 않는다. Go/npm/Cargo SemVer는
기존 고정 `golang.org/x/mod/semver`를 재사용하며 현재 consumer만 import guard에 등록했다.
언어 runtime/compiler 버전 owner나 update-runtime 절차는 바꾸지 않았다.

2026-11-03에 고정 URL에서 실제 취득한 GHSA-462w-v97r-4m45의 bounded projection1327B를
default testdata로 고정했다. 원본 hash와 출처/CC-BY4.0·삭제한 descriptive fields는 fixture
README에 기록했다. 실제 record schema→decoder→Jinja2 2.10 영향/동일 advisory의2.10.1
경계 대조를 확인했다. 이는 한 개 provider record의 frozen 재생이며 전체 Jinja2 안전성이나
제품 HTTP transport/CLI·전역 DB snapshot 검증이 아니다.

현재 후보의 표적 Race9.433s·Staticcheck0.902s·vet0.982s는 PASS다.
[영향 해석 부분 원장](./advisory-impact-local-20261103.json)이 정확한 source/fixture를
연결한다. 앞서 통과한 batch/record의 불변 native/CPP/전체 CI를 반복하지 않았다.

심각도는 다음 단위다. [GitHub의 공식 설명](https://github.com/github/advisory-database#database_specific-values)은
database_specific의 내부 필드에 의존하지 말라고 명시한다. 현재 decoder가 읽는 HIGH 문자열은
표시용 raw fact이며 Policy 입력으로 단독 채택하지 않는다. 실제 CVSS vector의 점수 해석을
위해 고정 library 후보의 source/checksum/license·scoring 계약을 검토 중이다. 현재 후보를
go.mod에 추가하거나 실행하지 않았다. 전체 S05의 네 상태와 최종16번은 계속 미완료다.

## 네 번째 단위 — 고정 CVSS parser와 base score

이전 문단의 후보 검토 이후 `go-cvss v0.6.4`를 채택했다. 공급·의존성·license·unsafe 호출
범위의 단일 검토는 [Coding Rules](../../../engineering/02-coding-security-rules.md#m13-004-advisory-cvss-dependency-review)가
소유한다. go.mod 한 줄과 검증된 go.sum 두 줄만 추가했고 기존 선택 version은 변경하지 않았다.
Runtime/compiler pin과 update-runtime 절차도 그대로다. Source `tools/cvss/NOTICE`의 실제
release/설치·업데이트 공급은 **아직 MISSING**이며 이번 수학 API 시험으로 닫지 않는다.

`osv_severity.go`는512B 이하 CVSS2/3.0/3.1/4.0 vector를 검증하고 base score를 계산한다.
추가 threat/environment 값으로 base 위험도를 낮추지 않는다. Malformed/unsupported vector,
parser panic, NaN/Inf는 실패이며 정상0점이 아니다. `Vector()` 직렬화 API는 호출하지 않고
자체 source에 unsafe/cgo를 추가하지 않는다. 단순 HIGH 문자열은 사용하지 않는다.
Scoring fact는 최종 Policy Decision이 아니며 아직 report/Policy에 연결하지 않았다.

실제 표적1.095s·Race2.543s·Staticcheck2.964s·vet0.560s,
canonical foundation77.841s/security30.184s/vulnerability29.356s는 PASS다.
Native Go fuzz는 seed4를 포함해 실제10,496회/6.024s PASS이며 operator43.887s에는 build가
포함된다. 이는 retained corpus/해당 bounded wrapper의 검증이고 모든 vector의 수학적
정확성이나 미공개 취약점 부재를 증명하지 않는다. 다른 profile/실제 설치·CLI·전체 S05
성공으로 전용하지 않는다. [CVSS 부분 원장](./advisory-cvss-local-20261103.json)이 exact
source/module·metadata와 check 범위를 연결한다.

다음은 공개 OSV 전송 경계와 기존 graph/결과/판정·유한 cache 연결이다. 계산 profile과 cache는
실제 linked CVSS version/checksum에 결합해 module update 후 예전 결과를 무조건 재사용하지
않아야 한다. 별도 version selector를 만들지 않는다. 전체 S05와 최종16번은 계속 OPEN이다.


## 다섯 번째 단위 — public transport와 기존 graph/report 연결

`osv_http.go`는 고정 API의 typed POST/GET만 받고 proxy/cookie/auth/redirect를
상속하지 않는다. TLS1.2 이상·정상 인증서/SNI를 사용하며 DNS의 모든 응답이 기존
`networkpolicy.IsPublicIPv4` 규칙을 통과한 뒤 numeric IPv4로 연결한다. 같은 주소 규칙의
기존 resolver 소비자는 유지하고 guest나 Root service에 새 operation을 주지 않는다.

`Inspector`는 기존 Inspect/InspectGraph/InspectProject/InspectProjectSelection/
InspectExecutionSelection과 CheckPlanner를 전달한다. Python의 static-all/closure·실제
frozen control·supplemental root 입력을 per-artifact 호출로 바꾸지 않는다. Node의
identity/declared integrity와 전체 acquired membership을 기존 검사 실행 전에 확인하고,
고정 graph의 package/version만 한 번 조회해 각 원래 subject의 report에 합친다.
실제 구성은 현재 Go·Cargo project inspector 두 경로에 연결했으며 다른 기존 consumer와
cache/CLI 전체 연결은 아직 MISSING이다. 구현된 기존 inspector를 교체/재작성하지 않았다.

Check profile은 `debug.ReadBuildInfo`의 실제 linked go-cvss version/검증 checksum에
결합한다. go.mod 외의 새 version selector/lock은 없다. Missing/replaced/unchecksummed
calculator는 거부한다. Response request/hash/time·record의 원래 modified/영향·base score를
기존 bounded Evidence로 기록하며 provider 설명을 실행·최종 판정 입력으로 사용하지 않는다.
M3 version3은 advisory 영향/심각도 평가 불명을 review로 받고, 확인된 affected base7.0
이상은 기존 BLOCK 우선순위로 처리한다. 더 강한 integrity/static/dynamic/unknown Finding을
가리지 않는다. 상세 계약은 기존 M3 Policy leaf가 소유한다.

실제 HTTPS DATA v1/v2는 incomplete였다. Batch의 GHSA-cpwx-vrp4-4pq7 modified는
`2026-09-10T03:49:48.526758Z`, 동일 상세 원본은 `.526758681Z`였다. 공식
[querybatch 구현](https://raw.githubusercontent.com/google/osv.dev/master/go/internal/api/query_affected.go)은
저장소의 GetModified를 사용하며 [Datastore timestamp 명세](https://docs.cloud.google.com/datastore/docs/concepts/entities)는
microsecond 미만을 버린다. 상세 GetFull의 원본 protobuf는 더 높은 정밀도를 보존한다.
확인한 공식3file은 immutable blob 해시로 raw 원장에 연결했다. Live 실패의 captured
projection1048B를 repo testdata에 두고 RED1.065s를 확인한 뒤, **microsecond-aligned
index만 그 정밀도로 비교**하도록 고쳤다. Nanosecond index·다른 microsecond는 계속
거부하며 원래 상세 시각을 변경하지 않는다. 같은 microsecond 내부의 변경은 해당 index로
구분할 수 없으므로 modified를 암호학적 snapshot이라고 주장하지 않는다. 실제 응답 해시도
계속 보존한다. 다른 오류를 정밀도 차이로 포괄 수용하거나 일반 clock-skew를 허용하지 않는다.

해당 advisory Race5.057s와 실제 production HTTP/linked calculator DATA v3/4.381s가
PASS다. V3는 Jinja2 2.10의12개 record/affected findings와3.1.6의 당시 빈 결과를 확인했다.
Synthetic original inspection을 사용했으므로 실제 package 검역/Policy/설치·CLI 성공이 아니다.
Private 확인 command는 제거했고, frozen 공개 응답14개·작은 원장을 보존한다. 앞선 연결
후보의 Race6.662s/static11.494s/vet7.671s와 networkpolicy0.342s도 범위별 증거이며,
최종 공통 검사는 이 묶음에서 진행한다. Whole S05의 네 상태·최종16번은 여전히 OPEN이다.


다섯 번째 묶음의 canonical foundation63.665s/security20.493s·bootstrap 전체
Race29.521s는 PASS다. 마지막 optional capability 거부 한정 변경은 현재 advisory
Race5.211s·Staticcheck1.124s/vet0.457s·format no-op로 확인했으며 그 두 줄 때문에
불변 HTTP/live/CPP/전체 공통 검사를 반복하지 않았다. 각 정확한 source snapshot과
검증 범위는 [연결 부분 원장](./advisory-connected-local-20261103.json)이 소유한다.
완료된 확인용 binary와 보존된 응답 사본7,678,407B는 Root의99process FD/maps 조회에서
references0/query errors0을 확인한 뒤 제거했다. 공용 cache/runtime은 보존했다.
최종 사용자 CLI·모든 실제 소비자·cache/24h refresh·명시 악성/alias·NOTICE 공급과
whole S05 qualification은 계속 MISSING이며 새 CI는 시작하지 않았다.


## 여섯 번째 단위 시작 — 실제 소비자 연결 baseline

기준 `1f04d6d`. 공통 transport/adapter/report/M3는 재사용한다. 현재 Go·Cargo의 구성은
연결했지만 전체 S05 네 상태는 계속 NO다. 다음 기존 소비자의 advisory wiring만 MISSING이다.

| 현재 연결 묶음 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| npm 기본 inspect/install | YES | NO | NO | NO | bootstrap의 기존 composite를 한 번 감싸 두 Application consumer에 전달 |
| Python 기본 설치·requirements·UV pip sync·tool 공유 binding | YES | NO | NO | NO | 기존 prerequisite 설정 후 감싼 동일 Port를 direct/install/bindings에 전달 |
| UV project add/sync/lock | YES | NO | NO | NO | 기존 standalone UV client의 inspector binding만 연결 |

기존 resolver·static/dynamic·layout/프로젝트 입력을 다시 구현하지 않는다. NPM/PNPM/Bun의
frozen layout 및 실행 선택/기타 소비자는 이 뒤의 별도 연결·실제 검증 대상이다. Local project
root code를 공개 registry package 이름으로 추측해 조회하거나 두 번째 graph를 만들지 않는다.
Binary의 매핑 불가는 설계 §4의 bounded 지원 유지/범위 한계 표시 계약을 먼저 적용하며
전체 GitHub/자체 release를 무조건 required unsupported로 감싸지 않는다.

고정 Go1.27.3의 실제 test binary에서도 go-cvss version/checksum이 존재함을 작은 review로
확인했다(`s05-native-test-build-identity`,1.010s; review source는 raw에 보존 후 제거).
이것은 metadata 존재 확인이고 실제 native/설치 성공이 아니다. 확인 없이 test binary에는
BuildInfo가 없다고 가정해 generated version selector나 별도 build flag를 추가하지 않는다.


이 묶음에서 frozen npm/PNPM/Bun project와 NPM/Python/UV/Pipx 실행 준비 소비자도
동일한 기존 factory 경계에 연결한다(공통 구현 YES, advisory WIRED/QUALIFIED/
ACCEPTANCE_CLOSED는 시작 시 NO). 기존 concrete inspector는 layout cache·root dependency·
실행 runtime proof의 owner로 그대로 사용하고, Application에 전달하는 Inspection Port만
감싼다. 새 constructor/DI framework·두 번째 resolver·check 누락 fallback을 만들지 않는다.
공개 advisory 사용 자체는 기존 fixture를 이용한 native/CLI 결과가 자동으로 새 기능 검증이
되는 근거가 아니며, 전체 cache/freshness/실제 consumer acceptance는 후속으로 남는다.

기존 내부 tool-install execution factory는 현재 공개 CLI 경로의 caller가 없음을 확인했다.
이 경로를 미래 소비자 때문에 미리 변경하지 않고, 실제 공개 UV/Pipx native factory와
supplemental selection 경계만 연결한다. NPM local project root는 별도 local source이며
동일 이름의 공개 package를 추측 조회하지 않는다.


현재 공개 npm 기본·Python 공유 binding/UV project 및 frozen npm/PNPM/Bun project와
NPM/PNPM/Bun·UV/Pipx 실행 준비의 advisory wiring을 같은 기존 helper로 연결했다.
별도 class/interface나 resolver는 추가하지 않았으며 각 concrete layout/runtime owner를
유지했다. 위 세 경로의 현재 WIRED는 YES, QUALIFIED/ACCEPTANCE_CLOSED는 NO다.
기본 bootstrap unit5.543s·Staticcheck2.578s/vet1.386s와 tagged build5.316s는 PASS다.
Tagged `-run ^$`는 실제 test 미실행이며 native 성공이 아니다. 현재 module/test binary
검토도 보존한 [소비자 연결 원장](./advisory-consumers-local-20261103.json)이 source와 범위를
소유한다. 의미 있는 기존 입력/CPP/live API/전체 foundation을 다시 실행하지 않았다.
다음은 C01 소유의 cache·24h freshness/refresh와 나머지 실제 CLI/판정·범위 한계다.
Whole S05와 최종16번·새 CI/배포는 계속 미완료다.


## 일곱 번째 단위 시작 — C01 advisory cache baseline·수정 지도

기준 `5bc81b8`. S05 cache의 IMPLEMENTED/WIRED/QUALIFIED/ACCEPTANCE_CLOSED는 모두 NO다.
`storage.Local`에는 indexed journal·예약/실제 bytes·소유권/파일 해시·durable 게시와 intake
한정 삭제가 이미 있다. Cache를 같은 이름으로 다시 만들거나 별도 영구 ledger를 추가하지
않는다. 현재 `validResourceHandles`는 evidence/intake만, `resourceRoot`도 해당 두 root만
다루며, `StoragePolicySpec`의 CacheBytes/AdvisoryTTLSeconds는 cache 소비자에 미연결이다.

현재 gap은 다음 범위에서 메운다.

- `ports/storage.go`: 현재 advisory 소비자만 사용하는 opaque key/body read/write 계약.
  Local은 OSV/CVSS/package·Policy를 해석하지 않는다. 임의 URL/path·실행 권한은 없다.
- `storage.Local`: 같은 journal에 advisory kind/key·expiry/index와 quota를 연결하고
  같은 root 아래 lazy private cache 영역을 소유한다. 새 cache payload에는 기존 생성
  owner가 없으므로 이 Local adapter가 opaque bytes만 생성·검증·게시·회수한다. 기존
  Artifact/Promotion/Evidence payload 알고리즘·소유권은 그대로다. 기존 RunID 형식은
  cache allocation generation 식별에만 재사용하며 InspectionRun/Policy 완료 authority로
  만들지 않는다. Root/UID/mode/link/inode/hash 대조와 사전 예약·실제 사용량 검사를 유지한다.
- `inspection/advisory`: 고정 request/provider/schema/calculator profile의 content key와
  bounded complete response bundle을 구성한다. ID별 record를 한 번만 직렬화해 중복 node
  fanout으로 큰 body가 복제되지 않게 한다. 참조/결과 누락·변조·시각 역행·만료·현재 profile
  불일치는 clean zero가 아니다. 오류/부분 응답은 cache에 넣지 않는다. 원래 취득 시각과
  response hash를 보존하며 재사용 때 시각을 갱신하지 않는다.
- API 응답 cache는 Artifact에 경로나 FD를 넘기지 않는 DATA다. Read는 ledger lock 안에서
  읽고 닫은 bytes만 반환한다. 삭제도 같은 exclusive lock과 확인된 RETAINED ownership 아래
  처리하며 network 동안 lock을 잡지 않는다. IN_USE/foreign/UNKNOWN은 삭제하지 않는다.
  TTL/공간 수거와 crash의 DELETING/absence를 bounded하게 확인하고 최초 실패를 보존한다.
- default freshness는 중앙 policy의24h를 사용한다. 강제 refresh는 기존 CLI/config 경계와
  실제 lookup에 연결한다. 용량 때문에 cache를 남기지 못한 것을 악성/clean 판정으로
  바꾸지 않으며 미완료 기록을 재사용하지 않는다. cache hit도 현재 report/Policy를 거친다.

표적 acceptance는 fresh hit의 무조회·정확한 원래 사실, expiry/refresh의 새 조회,
query/profile 분리·bundle 변조/부분 거부·동시 reader/writer와 quota/실제 bytes,
소유권 교체·실패/cancel·DELETING 복구/UNKNOWN 보존이다. C01 전체 staging/cache/rollback·
shared lease/crash/전체 retention·journal maintenance/설정·최종 qualification은 이 cache
단위만으로 완료 처리하지 않는다. 기존 runtime/compiler version selector나 CI job은 늘리지 않는다.


일곱 번째 단위의 cache 저장/조회·기존 CLI `--refresh-advisories` 연결은 구현했다.
같은 Local 원장의 사전 예약·확정 byte 회계·인덱스를 재사용하며, 새 공개 응답은 private
DATA로만 저장한다. 원래 static/dynamic 검사는 cache hit에서도 수행하고 이전 subject의
승인이나 검사 완료를 재사용하지 않는다. 실제 Local을 닫고 다시 연 공통 Inspector 연결
테스트12.170s가 원래 response hash/time과 현재 subject 재결합·강제 재조회를 확인했다.
이는 controlled HTTP와 synthetic original 검사이며 실제 검역/설치 성공은 아니다.

TTL·용량 회수/동시 갱신/DELETING 재생·foreign/UNKNOWN/IN_USE 보존을 검증했다.
16KiB 안의 기존10KiB가80% 기준선 아래여도 새10KiB를 넣지 못한 첫 실패1.573s는
회수 목표에 새 예약량을 반영해 교정했다. 새 용량이나 삭제 권한은 추가하지 않는다.
원래 storage 전체 Race는 기존 httptest의 sandbox socket 거부로 실패했고 동일 검사만
로컬 socket 권한으로 실행한3.759s는 PASS다. Advisory/CLI/bootstrap Race38.733s와
Staticcheck10.050s·vet3.185s, canonical foundation87.817s도 PASS다.
Gosec의 nested-loop index 지적은 suppression 없이 bounded query를 먼저 꺼내는 흐름으로
교정했다. 두 slice 길이 일치와 반복문의 양쪽 상한을 명시한 표적 G6020.508s 및
최종 canonical security13.286s·advisory Race6.048s가 PASS다. 최초 검사와 실패한 두
guard-only 시도도 원본으로 보존한다. 마지막 이 좁은 변경은 앞선 foundation과
정확히 같은 source로 표시하지 않고 후속 보안 검사에 별도로 연결한다.

Cache 단위 IMPLEMENTED/WIRED는 YES다. QUALIFIED/ACCEPTANCE_CLOSED는 실제 CLI와
전체 C01/final 조건이 남아 NO다. IN_USE를 expiry/Close/PID 부재로 지우지 않으며 generic
writer generation/crash 회수·journal maintenance·전체 retention은 여전히 C01의 MISSING이다.
응답 cache만으로 무제한 사본 문제 전체를 해결했다고 표시하지 않는다. 정확한 source와
checks/최초 실패는 [cache 부분 원장](./advisory-cache-local-20261103.json)에 연결하며, 다음은 explicit malware/alias·binary 범위와
실제 CLI/NOTICE 공급이다. Runtime/compiler version selector·CPP·Host는 변경하지 않았다.


## 여덟 번째 단위 시작 — 명시 악성·alias baseline·수정 지도

기준 `20d1b49`. Record decoder의 aliases/withdrawn·정확한 version 영향 평가와 공통 report/
M3 판정은 IMPLEMENTED/WIRED=YES이며 재구현하지 않는다. 명시 악성 분류와 alias 중복
Finding 병합은 IMPLEMENTED/WIRED/QUALIFIED/ACCEPTANCE_CLOSED가 모두 NO다.
현재 `report`는 record마다 review/high Finding을 만들므로 severity가 없는 MAL record가
review에 머물며 같은 vulnerability의 alias record도 중복된다. 이 두 acceptance gap만
기존 report에서 고치며 transport/parser/resolver/저장소·버전 selector는 변경하지 않는다.

[OSV 명세](https://ossf.github.io/osv-schema/#aliases-field)의 aliases는 대칭·추이 관계다.
동일 subject의 이미 취득한 record와 alias token만 O(records+aliases)로 연결하고 새 조회나
전역 database를 만들지 않는다. 각 원본 ID·modified·영향·응답 사실은 보존하며 component당
가장 강한 Finding 한 개에 관련 Evidence를 결합한다. Withdrawn/unaffected가 다른 active
record의 차단을 지우지 않으며 다른 package/subject의 결과를 합치지 않는다.

명시 악성은 [OSV 등록 MAL namespace](https://ossf.github.io/osv-schema/#id-field)와
[공식 Malicious Packages 보고 형식](https://github.com/ossf/malicious-packages/blob/main/CONTRIBUTING.md)을
기준으로 primary MAL ID의 영향 확인 시 `M13_ADVISORY_MALICIOUS`를 BLOCK으로 보낸다.
원래 M3의 strongest/default BLOCK을 재사용한다. Summary의 malware 단어·human severity·
임의 alias 문자열만으로 악성을 추측하지 않는다. 실제 MAL-2026-2307의 공개 record도
severity 없이 explicit versions를 제공함을 확인했다(정보 조회만, package 미취득/미실행).
모든 악성 package나 feed가 MAL에 존재한다고 주장하지 않는다.

표적 acceptance는 no-CVSS MAL affected/정상·withdrawn, alias 대칭/추이·단절/독립 record,
약한 record로 강한 차단이 사라지지 않음과 모든 관련 Evidence 보존이다. 이후 기존 공통
Policy의 blocker/review 순서 회귀를 묶어 확인한다. Binary 적용 범위/실제 CLI/NOTICE와
whole S05·최종 qualification은 이 단위만으로 닫지 않는다.


명시 악성·alias 단위의 IMPLEMENTED/WIRED는 YES다. 기존 원본에서 no-CVSS MAL이 review이고
alias가 중복되는 RED1.046s를 보존한 뒤 현재 report의 한정 보강으로 Race5.251s가 PASS다.
같은 subject 안의 symmetric/transitive/shared alias·순서 반전·독립 record, withdrawn의
active 차단 미소거와128alias의 bounded summary를 확인했다. CVSS/영향 해석·transport는
재구현하지 않았고, M3의 기존 default/strongest BLOCK을 그대로 사용한다. 새 code도 Policy
순서 회귀에서 BLOCK이다. Advisory profile을v2로 갱신해 앞선 profile과 구분했다.
원래 record마다 ID/modified/impact/CVSS/고정 OSV URL의 bounded Evidence를 남기며 alias
원문 배열을1024B summary에 몰아넣지 않는다. Component Finding은 모든 record fact를 참조한다.

Staticcheck1.418s/vet0.509s·현재 두 package gosec0.541s도 PASS다. Static/vet는 마지막
summary 배열 제거 전 source이며 최종 Race/gosec과 구분한다. 재현 입력·최초 RED/범위는
[부분 원장](./advisory-malware-alias-local-20261103.json)이 소유한다. QUALIFIED/
ACCEPTANCE_CLOSED는 actual CLI/최종 조건이 남아 NO다. 공통 foundation/security·실제
consumer 검증은 다음 binary 적용·NOTICE 공급과 의미 있는 묶음으로 모으며 새 CI는 없다.
