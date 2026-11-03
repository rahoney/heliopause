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


## 아홉 번째 단위 시작 — 기존 bounded binary 적용 범위 baseline

기준 `a9fe065`. GitHub/자체 release·Terraform의 기존 구조/무결성/제한 동적 검사와 Policy는
IMPLEMENTED/WIRED=YES이며 교체하지 않는다. S05의 binary 한계 표시/기존 연결은 네 상태가
모두 NO다. 동일 공통 Inspector가 이미 unsupported mapping을 보고할 수 있지만 모든 source에
required=true여서 그대로 연결하면 원래 bounded binary 지원 전체를 막는다.

설계 §4의 고정 applicability에 따라 인증된 `github-release`·`terraform-provider` subject만
component mapping 한계를 optional UNSUPPORTED/NOT_EXECUTED check와 bounded Evidence로
명시한다. 공개 package ecosystem의 mapping 실패·조회 실패/미완료는 계속 required이고
일반 unknown source도 required다. Artifact 출력·네트워크 실패·도구 부재로 optional로 바꾸지
않는다. Binary가 취약점이 없거나 모든 내부 성분을 검사했다는 표현은 금지한다.

수정 위치는 공통 `ExpectedChecks`/`report`의 동일 applicability helper와 기존 Root의 GitHub,
Terraform, native self-update `InstallInspect`·release-set composition이다. 원래 concrete
metadata/runtime owner를 유지하며 외부 Application Port만 한 번 감싼다. Alias/영향/조회/
저장소를 복제하지 않고 새 source mapping을 이름으로 추측하지 않는다. 정상 bounded binary의
종합 판정을 유지하고 required check 미완료·기존 BLOCK은 여전히 자동 허용을 막는 회귀를
연결한다. 실제 binary native/CLI·전체 지원/최종 qualification은 별도 MISSING이다.


Binary applicability 단위의 IMPLEMENTED/WIRED는 YES다. GitHub inspect/install/update,
Terraform project 및 self-update base/release-set의 기존 owner에 공통 wrapper를 연결했다.
Release-set은 outer Port만 감싸 concrete native/metadata/pipx dispatch를 중복 연결하지 않는다.
고정 두 source만 optional UNSUPPORTED/NOT_EXECUTED와 `advisory-applicability` Evidence를
남기며 profilev3로 선언한다. pypi local/invalid npm/unknown mapping은 계속 required다.
정상 bounded M6 ALLOW·required 미완료 REVIEW·기존 BLOCK 보존을 확인했다. 최초 unit4.990s의
새 incomplete fixture limitation 누락만 고쳤고, 전체 영향 advisory/Policy/bootstrap Race9.449s,
Staticcheck2.795s/vet1.476s가 PASS다. [부분 원장](./advisory-binary-local-20261103.json)이
source·원본/범위를 소유하며 QUALIFIED/ACCEPTANCE_CLOSED는 actual/최종 조건이 남아 NO다.
새 compiler/runtime version owner·guest 권한·CI·Host 변경은 없다. 다음은 NOTICE 실제 공급과
의미 있는 공통 검사/actual CLI 묶음이다.


## 열 번째 단위 시작 — CVSS NOTICE 실제 binary 공급 baseline

기준 `3c0ec3f`. 고정 CVSS source NOTICE의 IMPLEMENTED=YES, WIRED/QUALIFIED/
ACCEPTANCE_CLOSED=NO다. 기존 release/install/update는 CLI binary 자체를 인증·배포하므로
별도 passive asset/manifest schema/installer 역할을 늘리지 않는다. 같은 NOTICE bytes를
`internal/inspection/advisory/NOTICE` 한 곳으로 옮겨 표준 go:embed로 해당 binary에 포함하고,
기존 Root/CLI 등록 경계로 `helox licenses`의 human/JSON 출력에 연결한다. 상대 embed는
package 밖 파일을 허용하지 않아 source owner와 colocation이 필요하며 payload/허가문은
변경하지 않는다. 과거 evidence의 old path/hash는 역사 기록으로 보존한다.

CLI는 Inspection을 직접 import하지 않고 composition root가 고지 text만 전달한다. 명령은
runtime/network/storage 구성 전에 실행하며 실행 파일의 embedded bytes를 읽는다. 실제
배포된 새 binary를 기존 install/update가 복사하면 고지 원문도 함께 이동한다. 개발 cache의
외부 notice 경로에 의존하지 않으며 새 version selector·generated pin·duplicate NOTICE나
release member를 만들지 않는다. 기존 고정 library/module checksum 검토는 재사용한다.

표적 acceptance는 원본 bytes 동일성·실제 linked binary의 offline 출력, 추가 args/취소·출력
오류·JSON 원문 보존과 help의 무구성이다. 이어 accumulated common foundation/security와
영향 CLI/bootstrap 회귀를 묶는다. 이 방식은 새 library의 license 보존을 다루며 전체 signed
release/protected installation/자체 update qualification을 완료로 바꾸지 않는다.


NOTICE 단위의 IMPLEMENTED/WIRED는 YES다. 원래1217B/SHA256
`a46dc4ea765352593e37d7dd5ce59d512e43d62ce9e30c1d023c71d44cbdb78a`를 변경 없이
Inspection의 단일 NOTICE로 옮겨 binary에 포함했다. `helox licenses`는 runtime/network/
storage 구성 전에 실행하며 human/JSON 모두 원문을 보존한다. 실제 linked CLI의 빈 cwd
확인0.087s에서 human·JSON·help/추가 인수 거부와 작업 디렉터리 무변경을 확인했다.
명령의 취소·출력 실패 원인도 표적5.214s와 전체 CLI/bootstrap Race34.490s가 확인한다.

누적 후보의 canonical foundation27.020s/security15.422s·Staticcheck4.715s 및 vet는 PASS다.
정확한 source·기존 source bytes/실제 binary와 원본은
[고지 공급 원장](./advisory-notices-local-20261103.json)이 소유한다. 추가 release role·고지
사본/핀·새 버전 owner는 만들지 않았다. 실제 서명 배포·보호 설치본/자체 update qualification은
미수행이므로 QUALIFIED/ACCEPTANCE_CLOSED는 NO로 유지한다. 확인용 임시 cwd/단위 payload는
제거했고, 실제 CLI 검역 묶음에 바로 사용할 current binary 한 벌만 유지한다. 다음은 S05
실제 CLI 정상·차단/오류·cache/refresh와 원래 target 보존이다. 새 CI·Host 변경은 없다.


## 열한 번째 단위 시작 — 실제 production CLI·검역·target 경계

기준 `312a987`. 공통 API/record/CVSS/report/Policy/cache·현재 소비자/고지의 부분 구현과 연결은
재사용한다. 실제 CLI 정상/거부·미완료/캐시/refresh·target 보존의 QUALIFIED/
ACCEPTANCE_CLOSED는 NO이며 이 acceptance만 시험한다. 현재 production CLI binary 한 벌과
기존 고정 kernel/observer/responder/policy를 사용하며 불변 CPP를 다시 빌드하지 않는다.
Go helper의 실제 dependency closure/입력은 현재 source와 대조해 필요한 것만 준비한다.

같은 무해한 공개 `packaging==24.2` wheel의 실제 resolver/취득/무결성·static/dynamic·기록/
Policy·격리 설치/target 경로를 `pip install -r`의 production CLI로 통과시킨다. 공격코드나
실제 MAL package는 내려받지 않는다. Advisory의 deterministic 분기는 실제 Local cache에
controlled HTTP로 complete DATA만 seed한다. 이 seed의 원래 검사는 명시적 synthetic이고,
뒤 production CLI의 실제 Artifact 검사는 별개로 필수 수행해야 한다. Synthetic MAL record를
해당 공개 package가 실제 악성이라는 주장으로 사용하지 않는다.

Normal/동일 fresh cache 재사용, no-CVSS MAL fixture의 BLOCK/반입 전 거부, payload 변조의
INCOMPLETE/거부, 강제 refresh의 실제 production API 새 응답·기존 malformed/악성 cache
미사용을 확인한다. 타깃의 user marker/기존 파일은 매 실패에 동일해야 하며 success의 현재
subject/check/Evidence·원래 fetch proof와2개 환경 state를 확인한다. API 시점 변화는 진짜
조회 결과로 보존하고 고정 정상으로 억지 교정하지 않는다. 각 분기의 실제 CLI JSON/exit와
원본을 기록한다. 이 한 cohort로 다른 언어/모든 binary·S04·C01/전체 보강·hosted를 닫지 않는다.

시험은 기존 protected-source/private mount namespace·정확한 lifecycle/원복·helperwait0
절차를 재사용하며 UNKNOWN/foreign 상태에는 재바인드·삭제하지 않는다. 태그 빌드는 준비
확인일 뿐 실제 실행이 아니며 noSKIP/named census/원래 Host 및 시험 payload 정리까지
결과에 포함한다. 새 CI/job/20분 타이머 중복이나150분 integration 한도 변경은 없다.


열한 번째 최초 실제 결과는 FAIL이다. `s05-advisory-cli-native-v1`3.867s에서 production
`pip install -r`의 첫 fresh-cache가1.40s에 `requirements installation lacks a complete
native resolver`로 중단됐으며 Artifact 검사/새 advisory 판단에는 미도달이다. Helperwait0,
Host 동일/원복과381,869,782B 시험 사본 정리를 확인했다. 이 실패를 advisory 추가의 인과로
단정하지 않는다. 기존 B01 native 검증은 concrete PyPIResolver를 직접 bindings에 주입했고,
production Root는 PyPI/PyTorch의 `pythonSourceResolver`를 전달한다. Router의 일반 install/
update는 이미 구현됐지만 B01이 요구하는 `ProjectLockUpdateResolver` 전달은 빠져 있다.

이 연결의 IMPLEMENTED/WIRED/QUALIFIED/ACCEPTANCE_CLOSED는 NO다. Native PyPI의
requirements resolver는 이미 YES이므로 재구현하지 않는다. 기존 router에 fixed public PyPI
경로의 동일 Port 전달 한 메서드만 추가한다. Native 자체의 public venv/frozen input/selector
검증을 유지하며 PyTorch/다른 source로 fallback하지 않는다. 기존 native enrollment의
ResolveDependencies routing은 바꾸지 않는다. `pypi_update_test.go`의 기존 route fixture로
frozen controls/install/ctx·cancel·원래 오류와 missing/unsupported/다른 source 거부를 확인한
뒤 actual cohort를 계속한다. 새 framework·resolver·source/version selector는 없다.

실행 전 한 번의 별도 INPUT_PREFLIGHT는 기존 부분 snapshot에 실제 compiled/embedded
324개 입력이 없음을 검출했다. Host 변경/native 실행 전 중단한 근거와 실제1422개 repo
compiler/test/embedded 입력의 root/scratch 동일성·complete build0.318s/test-build0.760s를
raw input 원장에 남겼다. 그 교정은 guard 생략이 아니며 이전 부분 기록을 full snapshot으로
전용하지 않는다. 이 준비 실패와 뒤 실제 router 연결 실패를 구분한다.


기존 router의 public PyPI 전달 한 메서드만 보강한 bootstrap Race7.752s·Staticcheck3.869s/
vet1.117s 및 realcorpus-tagged Staticcheck2.263s가 PASS다. Source/selector/frozen controls/
install/ctx·cancel/원래 오류의 전달과 missing/unsupported/torch-only route의 미호출을 확인했다.
실제 V2는 parent255.21s/operator258.093s에서5개 child/production CLI6회를 PASS했다.
정상 fresh/cache2회는 ALLOW/실제 설치, synthetic no-CVSS MAL은 BLOCK/target 무변경,
cache payload 변조는 required INCOMPLETE/거부·target 무변경, explicit refresh는 실제
production OSV 응답의 새 취득 증거와 ALLOW를 확인했다. 해당 공개 packaging24.2를 악성이라고
주장하지 않으며 controlled seed와 실제 API를 구분한다. 정상/차단의4개 필수 check·현재
subject에 저장된2개 환경 state/원래 cache proof도 확인했다. 이 결과는 pip requirements의
scoped 실제 consumer 증거이며 다른 언어·모든 profile/binary나 whole S05 성공은 아니다.

NoSKIP/helperwait0/Host 동일·원복과381,873,878B 시험 사본 제거를 확인했다. 이어 Root의
105process executable/FD/maps 조회에서 references0/query errors0을 확인하고 완료 CLI/test
runner와 교체 전 helper102,449,863B를 제거했다. Current policy helper/responder/SDK는 한 벌만
보존한다. [실제 Root 부분 원장](./advisory-root-cli-local-20261103.json)이 최초 준비/실제
실패·현재 source와 원본/정리 범위를 소유한다. Source version selector는 불변이며 새 test의
site-packages 경로도 고정 Python version에서 파생한다. Whole S05 네 상태·최종16번은 여전히
OPEN이다. 다음은 user-visible advisory 상세 전달, source/profile의 적용 범위와 나머지 실제
소비자 대조다. Root 결과는 현재 판정 code/check/Evidence references를 제공하지만 advisory
ID/수정 경계/URL을 직접 표시하는 경로는 아직 대조·보강해야 한다. 새 CI/영구 Host 변경은 없다.


## 열두 번째 단위 시작 — user-visible advisory 상세 baseline·수정 지도

기준 `74372f5`. `advisory.report`의 bounded record facts와 Local.ReadReference의 정확한
Run/ref/identity/digest/profile 조회는 이미 IMPLEMENTED/WIRED=YES다. 현재 Result/Dependency
Inspection은 Evidence reference만 전달하므로 CLI의 ID/심각도·영향·수정 경계/URL 표시 연결은
IMPLEMENTED/WIRED/QUALIFIED/ACCEPTANCE_CLOSED 모두 NO다. Actual MAL 결과도 판정 code와
opaque reference까지만 보여 이 gap을 직접 확인했다. 기존 reader/Policy/Domain 결과를
재구현하거나 전체 review persistence/query를 이 작업에 끼워 넣지 않는다.

기존 CLI presentation 상태에 현재 Local.ReadReference와 동일한 read-only 경계를 Root에서
한 번 공급한다. 모든 해당 registry 경로의 cacheRoot/heliopause/evidence는 같은 owner임을
Go/Cargo 포함 현재 constructor로 확인했다. Store constructor는 경로만 갖는 inert object이며
새 DB/연결/영구 사본은 없다. Source별 writer·검역/판정/승인/반입 흐름은 변경하지 않는다.
UI는 같은 CheckPlan의 Run/identity/digest와 current advisory check/profile에 결합된 recorded
facts만 읽는다. Schema·kind·binding·무결성/조회 오류는 상세 미확인으로 표시하고 새 Policy나
안전 판정을 만들지 않는다. Domain API에 provider용 결과 fields나 새 evidence authority를
추가하지 않는다.

공통 loader/view 한 곳을 standalone/project/install/execution의 기존 human/JSON presenter에서
재사용한다. 정상 ALLOW의 기본 세부 출력/I/O는 늘리지 않고, review/block의 advisory record를
현재 subject와 함께 표시한다. Human은 bounded 핵심 record/수정 경계·고정 URL을 보이며
나머지는 JSON으로 확인할 수 있게 한다. 조회 실패는 원래 Policy/operation/installation과
별도 상세 상태로 남겨 primary 오류·exit 의미를 덮지 않는다. Raw tool/env/network payload는
표시하지 않는다. Graph별 repeated 전체 store scan이나 cache/network 조회도 만들지 않는다.

Inspection에서 영향 enum을 wire상 의미 있는 문자열로 정규화하고 UI는 점수/영향을 다시
평가하지 않는다. 이 실제 출력 의미 변경은 advisory profile을 갱신해 구분한다. 완료 조건은
record ID/unknown severity/영향·수정 예/URL의 human/JSON 전달, exact subject/profile·record
오류/legacy 값·cancel의 안전한 상세 미확인, 원래 BLOCK/정리 축과 user data 보존이다.
표적 presenter/store 연결을 먼저 검사하고 실제 추가 UI qualification은 나머지 S05 source/
profile·consumer 변경과 모은다. 앞 actual5case를 출력만 추가한 helper마다 반복하지 않는다.


열두 번째 실제 source 대조에서 기존 binary 적용 분기의 잘못된 가정을 확인했다. 구현과
시험이 `terraform-provider`를 공유했지만 실제 adapter.Source()는 `terraform-registry`다.
실제 Source()와 존재하지 않는 alias를 쓰는 RED2.293s에서 두 방향 오류를 모두 확인했다.
직접 Source/SourceName을 사용하고 canonical architecture의 해당 두 adapter import만
명시해 교정한다. 임의 source는 required로 유지한다. 아홉 번째의 controlled literal 시험은
실제 Terraform 적용을 증명하지 않았으며 이전 좁은 green을 근거로 이 오판을 유지하지 않는다.
해당 원래 evidence는 편집하지 않고 이 반증과 후속 검증에 연결한다. Version owner·실제
Terraform native/형식/서명·동적 필수 조건은 바꾸지 않는다.


열두 번째 단위의 IMPLEMENTED/WIRED는 YES다. Existing Local.ReadReference를 공급한 공통
presentation loader/view를 standalone/install/project/execution 결과에 연결하고 advisory
profilev4의 영향 문자열을 표시한다. Domain/Policy/승인 구조·실행 권한·runtime version owner는
변경하지 않았다. 실제 Local 기록/조회와 human/JSON20case가 정상 field 전달, unknown severity의
0점 오표시 금지, foreign subject/profile·변조/누락·cancel·legacy 거부와 primary Policy/exit
보존·정상 ALLOW의 추가 조회0을 확인했다. 표적3.639s, 영향3package Race38.901s와 canonical
foundation53.629s/security22.576s·Staticcheck9.854s/vet1.387s가 PASS다. Tagged build는
실제 test 미실행이다. [부분 원장](./advisory-details-local-20261103.json)이 source·최초
SourceID RED/현재 checks를 소유하며 QUALIFIED/ACCEPTANCE_CLOSED와 whole S05는 NO다.

표시 계약은 [Terminal presentation](../../../user-journey-cli-ia/05-terminal-presentation.md#recorded-advisory-detail--m13-004-s05)이
소유한다. 앞 actual5case를 helper/출력별로 다시 돌리지 않았다. 나머지 source/profile 적용과
현재 public consumer·text-only legacy 경로를 대조해 실제 확인을 모으며, 새로운 metadata의
native 결과를 앞 v3 profile 결과로 표시하지 않는다. Known package mapping 미지원으로 기존
지원 profile 전체를 뜻하지 않게 막지 않는지도 확인해야 한다. 새 CI/영구 Host 변경은 없다.

## 열세 번째 단위 — 공식 Python profile/local build 적용성 (진행 중)

기준 f8ce970. 기존 named SourceProfile/runtimeidentity lock·sourceOwnsProject·
ParseReferenceForSource와 public PEP440 comparator는 IMPLEMENTED=YES/WIRED=YES다.
그 owner를 재작성하지 않는다. Advisory의 official local profile mapping·경계 보존은
IMPLEMENTED=NO/WIRED=NO/QUALIFIED=NO/ACCEPTANCE_CLOSED=NO이며 MISSING은
profile-owned project/full version 검증, local 순서, 유한 exact+public 후보 조회·보수적
판정, 기존 public 소비자 및 cache/report 회귀다. Native/hosted qualification은 별도다.

공식 OSV PyPI helper는 PEP440 version ordering을 사용한다
([source](https://raw.githubusercontent.com/google/osv.dev/master/osv/ecosystems/pypi.py)).
2026-11-03 metadata-only live batch는 torch2.5.1/public·cpu·cu124에22개씩,
2.6.0에24개/2.6.0+cpu에21개, 현재 고정2.14.0의 네 local profile에는0개를 반환했다.
이0은 당시 provider 응답이며 안전/검역 성공 증명이 아니다. Raw s05-local-versions-*
및 s05-local-version-boundary-record.json을 보존한다. GHSA-c678-jfcj-6jmf의
last_affected=2.6.0 때문에 local 순서상 그 뒤인2.6.0+cpu가 제외된다. Exact query만으로
upstream 경계까지 검토했다고 할 수 없다.

기존 공식 profile의 ownedProjects와 정확한 build suffix에만 namespace를 연결한다.
Identity를 public 버전으로 바꾸지 않고 exact local query와 public-base candidate query를
같은 bounded batch/예산에 넣어 union한다. Exact local 영향이 확인되면 기존 판정,
public-base만 affected이고 exact local 영향은 확인되지 않으면 UNKNOWN/manual review다.
후보 누락·실패는 incomplete이며 local tag를 제거한 단일 조회/자동 clean으로 대체하지 않는다.
이 비교는 dependency specifier matching과 다르며
[PEP440 local ordering](https://packaging.python.org/en/latest/specifications/version-specifiers/#local-version-identifiers)을 따른다.
Cache/profile에는 새 의미를 결합하고 Evidence에 두 후보의 의미를 명시한다.
Runtime pin/버전 updater·동적 검사·권한·CI 예산은 유지한다. 새 CI는 시작하지 않았다.

현재 named local 단위는 IMPLEMENTED=YES/WIRED=YES이며 QUALIFIED/ACCEPTANCE_CLOSED는
NO다. Existing SourceProfile membership/normalizer·public comparator를 재사용한 installed
local comparator와 exact+upstream dedup/union만 연결했다. Public-only API는 그대로다.
Upstream-only affected는 UNKNOWN/manual review, exact affected는 기존 strongest 판정,
withdrawn/public fixed는 구분한다. Profile v5로 이전 cache와 의미를 분리한다.

RED1.322s는 missing companion/dedup·예산/기존 local 해석 실패를 보존한다. 첫 target5.544s는
RequestedVersion의3반환값을2개로 받는 compile 오류였고 getter를 실제 계약에 맞췄다.
표적1.349s·영향3package fullRace33.251s·foundation78.777s/security33.630s·Staticcheck23.029s는
PASS다. Cache review 재사용/강제 refresh 실패·정확한 subject 보존, 네 profile의 소유/다른
build 거부와 기존 public comparator 회귀를 포함한다. Native PyTorch 전체/실제 새 UI/hosted는
미도달이며 전체 S05는 계속 OPEN이다. [부분 원장](./advisory-profiles-local-20261103.json)이
실제 입력과 범위를 소유한다. 새 패키지/DB/동적 권한/버전 pin·CI를 추가하지 않았다.

## 열네 번째 단위 — 기존 non-TTY 텍스트 결과 연결 (진행 중)

기준4d6ae9b. 저장된 advisory reader/view·rich human/JSON renderer는 이미
IMPLEMENTED=YES/WIRED=YES이며 재구현하지 않는다. `WriteHumanResult`와
`WriteHumanInstallResult`의 기존 non-TTY/auto 텍스트 분기만 같은 view 호출이 빠져 있다.
이 연결의 IMPLEMENTED/WIRED/QUALIFIED/ACCEPTANCE_CLOSED는 NO다. MISSING은
동일 reader/render 호출과 기존10상태 회귀의 텍스트 분기 대조다. 명령별 JSON 미지원은
현재 preflight가 명시 거부하는 별도 계약이며 이번에 build/resolve schema를 확장하지 않는다.
기존 exit/판정·출력 상한/정상 ALLOW의 추가 I/O 없음과 최초 오류 보존을 유지한다.

Text 분기 RED3.679s는9상태에서 detail 누락을 검출했고 정상 ALLOW는 추가 I/O 없이
통과했다. 두 기존 writer가 같은 reader/render를 호출하도록 연결한 표적3.534s와
CLI fullRace30.304s/Staticcheck3.085s/vet0.785s는 PASS다. 기존10상태를 rich human/JSON/
non-TTY text의30case로 확장했고 판정·exit20/10/0/누락·변조·취소·subject/profile 경계와
ALLOW no-I/O를 보존한다. Writer의 첫 오류도 반환한다. IMPLEMENTED/WIRED=YES,
QUALIFIED/ACCEPTANCE_CLOSED=NO이며 실제 install/detail 소비자는 다음 묶음에 남는다.
[부분 원장](./advisory-text-local-20261103.json) 참조. 같은 입력의 foundation/native/CPP/CI는
반복하지 않았다. 다음은 기존 public 언어 소비자의 실제 production CLI qualification이다.

## 열다섯 번째 단위 — 남은 공개 언어의 production CLI (진행 중)

기준fa15234. 기존 npm install/Go tidy/Cargo fetch의 native 선택·공급 인증·검사·기록·
Policy·반영과 S05 공통 연결은 IMPLEMENTED/WIRED=YES다. 새 resolver/pipeline을 만들지
않는다. 이 후보의 세 실제 CLI 정상/통제 MAL 거부·target 보존·저장 상세 검증은
QUALIFIED/ACCEPTANCE_CLOSED=NO이며 MISSING이다. 기존 protected-role/operator의
일시 bind·helper wait·Host 원복과 동일 compiler 입력을 재사용해 작은 공개 package만
검사한다. Seed는 통제된 advisory DATA이며 정상 공개 package가 악성이라는 주장이 아니다.
실제 package 실행·검역과 통제 HTTP seed를 분리한다. Go tidy/Cargo fetch는 명시된
의존성 준비 범위이며 application/build-script 실행 성공으로 확대하지 않는다.

[현재 실제 원장](./advisory-public-cli-local-20261104.json)의 v5는 parent100.09s/operator102.670s,
세 언어 정상/통제 MAL의6case/noSKIP/helperwait0/Host 동일·원복과381914742B 사본 정리를
PASS했다. npm is-number7.0.0은 required4, Go uuidv1.6.0과 Cargo itoa1.0.17은 required3이다.
정상은 ALLOW/반영 완료, MAL은 BLOCK/각 기존 대상 무변경이고 저장 상세는 AVAILABLE다.
Go tidy/Cargo fetch는 의존성 준비이며 fixture init/build.rs는 실행하지 않았다.

같은 CLI90ca83d3의 Python JSON entry 상세/required4/BLOCK은 v3 원본에서 보존한다.
그 parent는 잘못된 human assertion 때문에 FAIL이며 PASS로 재명명하지 않는다. 별도
malware-human의 parent는 v4에서41.257s에 PASS해 명시 human/exit1/unknown severity·
MAL ID와 대상 보존을 확인했다. Auto/non-TTY는 JSON이므로 legacy text native 성공이라고
확대하지 않는다. v4의 뒤 npm parent FAIL과 v5의 독립6case PASS를 구분한다.

최초 준비 실패는 새 CLI와 이전 binary receipt의 비교였고 Host bind/native 전에 멈췄다.
Current receipt로 연결하며 guard를 유지했고105process refs0/errors0·Host 동일 후
381912840B만 정리했다. v2는 project entry detail을 root array로 가정했고 v3는 auto/exit20을
추정했다. v4의 npm은 필수 원본 package.json/lock 없는 fixture와 standalone install JSON을
project entries로 읽는 가정이 틀렸다. 실제 source의 machineOutput/installStatuses/
installMachineResult/freezeNPMProject와 이미 저장한 JSON을 대조해 시험만 교정했다.
제품·예산·보안 조건을 바꾸거나 성공한 Python/동일 CPP/runtime를 반복하지 않았다.

실제 current Cargo required3에 맞춰 기존 hosted fetch fixture도 advisory ID·완료와3개를
확인하도록 고쳤다. Direct injected static-only Cargo cache fixture의2개는 변경하지 않는다.
같은 CI job/timeout을 유지한다. 이번 범위의 IMPLEMENTED/WIRED/로컬 native QUALIFIED는
YES이며 전체 S05/hosted ACCEPTANCE_CLOSED는 NO다. 공유 CLI/test/helper 한 세트는
후속 binary/consumer 확인에 재사용할 활성 입력으로 보존한다. 별도 CI는 시작하지 않았다.

## 열여섯 번째 단위 — legacy 프로젝트 명령의 실패 사실 표시

기준893522a. Go exact get/Cargo add/Terraform exact init의 guarded selection·기존
ProjectPolicyFailure/ProjectUpdateFailure와 공통 partial-result/report/detail renderer는
IMPLEMENTED/WIRED=YES다. Root의 기존 summary binder는 err 때 바로 반환해 이미 보존한
검사·판정을 사용자에게 표시하지 않는다. 이 표시 연결의 네 상태는 NO이며 MISSING은
기존 typed error facts→동일 bounded human renderer의 연결과 cause/exit·성공 응답 보존이다.
별도 resolver/Domain DTO/재검사·새 machine success schema·JSON 지원 범위는 만들지 않는다.
실제 세 caller가 공유하는 좁은 표시 조정만 두며 helper를 미래 consumer용으로 확장하지 않는다.

기존 Go download도 검사 snapshot의 error facts를 표시하지 않아 같은 네 번째 caller에 포함한다.

현재 네 caller의 IMPLEMENTED/WIRED는 YES, 전체 QUALIFIED/ACCEPTANCE_CLOSED는 NO다.
같은 presentation owner의 좁은 coordinator가 기존 projectUpdateFacts/partial failure·
check/reason/detail renderer를 호출한다. 새 DTO/다른 검사·resolver·mutable presentation
mode 전환은 없다. Error-only before-inspection은 기존처럼 사실을 만들지 않으며, 성공
summary/explicit JSON 미지원 계약과 primary cause/exit는 유지한다.

RED1.392s(세 source·recorded6 실패/early6 통과) 뒤 표적3.941s와 초기3 caller의
CLI/bootstrap fullRace34.640s가 PASS다. Download 포함4 caller의 최종16case Race8.170s와
tagged Staticcheck10.411s/vet1.425s, actual driver compile8.906s/production build6.703s도 PASS다.
서로 다른 source 경계를 하나의 full-frozen green으로 합치지 않는다.

현재 CLI62fbe056의 실제 Go get/Cargo add 두 통제 MAL 거부는17.935/9.616s, parent27.56s/
operator29.892s에 PASS했다. Native 선택/각 required3을 마치고 현재 stored ID/unknown
severity·M13_ADVISORY_MALICIOUS와 exit1·대상 무변경을 확인했다. Helperwait0/Host 원복·
381927030B 사본 정리도 PASS다. 이전 정상6case/Python/C++/helper/runtime는 재사용했으며
Go download/Terraform native·전체 S05/hosted는 별도 OPEN이다.

105process refs0/query errors0 뒤 교체된 이전 CLI/test/old helper의102652416B를 정리했다.
현재 한 세트와 SDK·공유 cache는 다음 binary/consumer qualification의 활성 입력으로
남긴다. [부분 원장](./advisory-legacy-failure-local-20261104.json)이 정확한 source·범위를 소유한다.
