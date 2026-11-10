# M13-004 — ordinary 최초 획득과 준비 리포트 연결

기준 `8706304`. Root `bootstrap-files`/register는 독립 검증한 고정 local bytes만 처리한다.
Root helper에 network나 arbitrary input을 추가하지 않는다. 현재 M13-004의 남은 ordinary
획득/첫 설치 안내 경로만 연결하며 같은 work item을 유지한다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| fixed public GitHub set/FD snapshot/proofs | YES | YES, self-update | YES, local | YES, component | frozen member reader/facts export |
| compiled runtime identity generator | YES | YES | YES | YES | 전체 lock byte digest 상수 |
| Root first file prep/intent/등록 | YES | YES | YES, local | PARTIAL, actual SDK 없음 | 재사용 |
| ordinary bootstrap acquisition/export/CLI | NO | NO | NO | NO | 아래 실제 prepare 경로 |
| 전체 actual signed/OS/systemd/reboot | PARTIAL | PARTIAL | NO | NO | 실제 새 native9와 별도 설치·권한 절차 |

## 수정 위치와 방법

1. `scripts/generate-runtime-lock.go`는 기존64KiB 한도 안의 **같은 전체 lock bytes**를
   parse/hash하고 `runtimeidentity.RuntimeLockSHA256`를 생성한다. Lock 내용·runtime 버전·
   C++ BUILD는 변경하지 않는다. 수동 digest 복제나 입력별 다른 lock owner를 만들지 않는다.
2. `artifact/githubrelease`는 resolve한 동일 frozen set의 manifest/proof facts와 selected
   member의 held FD reader를 제공한다. Reader는 Linux의 기존 FD를 duplicate하고 bounded
   section만 읽는다. User cache pathname을 재탐색하거나 download/resolve fallback하지 않는다.
   Fact getters는 provenance/Policy 허가가 아니다. 기존 ReleaseProof/workflow는 유지한다.
3. `ports`의 실제 bootstrap preparation service/DTO와 `releasebootstrap` ordinary adapter를
   연결한다. Linux amd64/ordinary UID, exact version/compiled lock만 허용하고 기존 fixed
   public client와 offline Sigstore SDK를 사용한다. SDK는 전체 native4+metadata3를 묶는다.
   Root helper·old installed record·Policy ALLOW를 위조하지 않는다. Root UID/unsupported
   platform/잘못된 입력은 디렉터리·HTTP 이전에 거부한다.
4. Preparation은 새 user-owned0700 directory의0600 files/proofs로만 export한다. File names,
   metadata sizes와 native 전체1GiB/proof bounds를 유지하고 actual copied bytes를 재해시한다.
   Source Close/owned intake 정리/fsync 뒤에만 PREPARED를 보고한다. 실패는 original cause를
   유지하며 소유 FD/inode가 확인된 파일만 정리하고 unknown/replaced 것을 보존한다.
5. `helox bootstrap prepare --version vX.Y.Z`는 기존 CLI presentation/일곱 언어/help/
   stage progress/JSON 구조를 재사용한다. SDK verified·quarantine NOT_PERFORMED·installation
   PENDING을 명시하며 Root 도구 실행·OS runtime 설치·설정 쓰기·기존 설치 migration을 하지 않는다.
   Bootstrap composition은 Host executor/runtime lease 없이 ordinary acquisition service만
   구성한다. Invalid/duplicate version을 composition/HTTP 이전에 거부한다.
6. Directory/architecture map에는 현재 실제 ordinary distribution adapter와 port만 추가한다.
   Root installer entry의 책임·imports는 유지한다. 사용자 승인 없는 registry 이름/계정
   binding/publish/tag/main merge와 영구 CLI migration은 하지 않는다.

## 검사와 두 번째 대조

Canonical formatter/Quick/security/docs/vulnerability와 generator·FD/source·adapter·CLI의
relevant Race 및 Darwin TEST compile을 확인한다. Source close/path replacement 이후 held
reader, wrong scope/duplicate/closed/size/hash/proof/SDK/cancel·writer/cleanup 실패와 known
files만 제거하는 경계를 검사한다. CLI7언어 help/EN·KO report/unsupported·unmanaged와
Root·HTTP 이전 syntax 거부를 구분한다. SDK positive가 mock이면 actual signed bootstrap으로
보고하지 않는다. 완료 output/operator/test payload를 마지막 사용 뒤 삭제한다.

두 번째 대조에서 ordinary acquire를 Root installer에 넣는 대안은 사용하지 않았다.
Existing root installation은 기존 controller가 검역하는 self-update를 사용한다. First
prepare는 독립 provenance와 bytes를 준비할 뿐 검역이나 설치 완료가 아니다. Root 실행 전
실제 root-protected helper 사본의 independent verification, exact native OS config/unit,
설치된 helper register와 일반 UID의 doctor/nominal 검사는 여전히 필요하다. Existing
Architecture·Domain·Security invariant를 바꾸거나 public release를 선행하지 않는다.

## 구현 후 재검토

Export는 source stream의 digest뿐 아니라 Source.Close/owned intake 정리 이후 실제 export
files의 owner/inode/mode/size/mtime와 읽어낸 전체 digest를 다시 확인하고 fsync한다.
Handoff 직전 변경된 파일은 PREPARED로 보고하지 않으며 바뀐 파일을 지우지 않는다.
첫 FD replacement fixture는 기존 intake filename을 `artifact`로 잘못 가정해 I/O 이전에
실패했다. 실제 owner의 `asset` 경로로 맞추며 product pathname fallback을 추가하지 않는다.
