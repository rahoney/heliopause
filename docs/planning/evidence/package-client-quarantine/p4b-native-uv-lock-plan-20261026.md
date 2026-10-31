# P4-B native UV lock와 complete target export 결합

Baseline619b214bf7dcb8145652881146740d5a72007c4a/tree3f5dc6efc1966101e5b32dfdaf106fba30e96d12/2482files.
Read: P4-A baseline/result, client 설계§1/§2/§4, 확장 계획§3–6, UVProjectInput,
기존 UVProjectLock/parseUVLockedPackages/BindProject·독립 metadata/whole wheel binder.
실제 UV0.13.4 offline 빈 lock은 version1/revision5/virtual root다. 해당 exact tag의
[serializer](https://github.com/astral-sh/uv/blob/0.13.4/crates/uv-lock/src/lock/serialize.rs)와
[deserializer](https://github.com/astral-sh/uv/blob/0.13.4/crates/uv-lock/src/lock/deserialize.rs)의
package/dependency/source/metadata/wheel fields도 대조했다. Official fixture의 과거 revision3을
현재 native5 지원 증명으로 쓰지 않는다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| frozen virtual input/intent | YES(P4-A) | YES(control replay) | YES(named local) | YES(input only) | native result consumer |
| complete wheel pylock 1.0 parser/independent binding | YES(P1) | YES(uv pip sync) | YES(P1) | YES(existing scope) | native uv.lock와 export 간 identity/closure 연결 |
| native uv.lock version1/revision5 reader | NO | NO | NO | NO | virtual root·public registry/edges·metadata·target export binding |
| native project resolver/CLI/promotion | NO | NO | NO | NO | reader 다음 substep; 이번에 placeholder 없음 |

## Exact map·두 번째 검토

`internal/artifact/pypi/uv_native_project_lock.go`에 별도 unapproved reader와 테스트를 추가한다.
기존 UVProjectLock/ParseUVLock과 P1 consumer는 대체·재작성하지 않는다. Native wheels를
strict pylock validation view로 변환해 기존 URL/filename/hash/size/alternative/graph budgets를
재사용한다. 이 내부 view를 native export 증거로 주장하지 않는다. Native 원본 bytes를 보존한다.

첫 subset은 정확한 version1/revision5, 단일 manifest와 일치하는 virtual root, public PyPI
registry package, marker-free bounded dependency edges다. Unknown fields/다른 source/중복/
root 혼동/없는 edge/disconnected entries를 거부한다. Root metadata requires-dist는 manifest의
명시 dependency requirement와 대응하고 선택된 version이 각 요구를 만족해야 한다.
Wheel 목록 외 sdist는 공개 URL/hash/size의 **inert alternative metadata**로만 읽으며
선택/취득/build/실행하지 않는다. Existing wheel-only bound export/census가 실제 선택을 소유한다.

Target export의 package/version과 wheel URL/hash/size가 native lock에 모두 존재해야 한다.
Root virtual entry는 artifact set에 포함하지 않는다. Empty는 명시 virtual root와 empty
export의 조합만 허용한다. Native graph의 선언은 인증된 wheel Requires-Dist/독립 PyPI
metadata·실제 installed census를 대체하지 않는다. 이 reader만으로 Policy나 publication
권한을 만들지 않으며 resolver 연결/실제 public native qualification은 계속 MISSING이다.

Canonical format/Quick/security/docs·artifact/PyPI full Race와 root/source/schema/edge/
manifest/export substitution negatives를 적용한다. 실제 native empty 원본과 synthetic
nonempty 구조 검증을 구분한다. Source 불변인 기존 backend/CI·CUDA full을 parser마다
반복하지 않고 CI413 timer를 그대로 소비한다. M13-004 하나만 IN_PROGRESS다.

## Named local 결과

[Reader 영수증](./p4b-native-uv-lock-local-20261026.json)의 Quick202.858s/security12.989s/
docs0.255s와 artifact/PyPI fullRace5.679s가 PASS다. Actual native empty 형식과 synthetic
nonempty graph/export·거부 경계를 구분했다. 기존 parser/native/CPP/runtime/CI owners8개는
baseline bytes 그대로다. Reader named 경계만 닫았으며 actual public nonempty/export schema와
resolver/독립 source binding·transaction/CLI/persistent tool·whole P4는 계속 미완료다.
