# P2c5-B5b2 — 승인 cache의 private Host materialization

Baseline aa204f2. M13-004/P2 하나만 IN_PROGRESS이고 CI406은 앞 fd3506b를 검증 중이다.
Read: B5a/B5b1·client 설계§4·Domain09/13·Architecture02/03, 기존 npm transaction/member와
regular-only sync/inventory, shared guard/backup, artifact L15/L19·sandbox L08/L12·evidence L11/L16.

## Source 작성 전 baseline audit

| 책임 / symbol | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| PNPM approved cache + actual-operation records — PNPMVerifiedCache | YES | YES(actual) | YES(named local) | NO(P2) | private Host tree로 byte-checked 반출 |
| Native full tree parser/binder/internal links — ParsePNPMLayoutArchive/BindLayout | YES | YES | YES(named local) | NO(P2) | Host의 private copy와 readback |
| npm existing transaction/sync/member | YES | YES(npm) | YES(existing) | NO(P2) | pnpm 내부 link를 무차별 허용하지 않는 typed 소비자 |
| private PNPM Host materializer | NO | NO | NO | NO | 이번 단위 |
| independent retained approval / target commit / adoption / CLI | NO | NO | NO | NO | 후속; private workspace 성공을 설치 완료로 쓰지 않음 |

## 정확한 edit map과 두 번째 검토

기존 `promotion` package에 `pnpm_materialize.go`를 추가한다. PNPMVerifiedCache.PrepareProject는
현재 실제 Evidence/Policy·operation records·typed staged receipt를 Open으로 재검증한다.
그 receipt의 SHA를 다시 확인하고 pinned os.Root/regular-file identity로 bounded data tar를 읽어
receipt inventory SHA와 비교한다. 새 mutable cache 경로가 아닌 캡처한 immutable bytes를 사용한다.
Controls는 frozen 원본, records는 같은 actual operation의 trusted builder에서 다시 만들고
receipt의 각 byte digest와 비교한다. 모든 검증/parse가 workspace 생성보다 앞선다.

파싱한 exact tar entry의 regular bytes와 hardlink alias bytes를 **각각 새 single-link regular
file**로 쓴다. 외부 store와 inode를 공유하거나 원래 cache file을 link하지 않는다. Native mode와
승인된 내부 symlink text는 보존한다. Symlink target은 explicit owned native directory/regular
entry여야 하고 모든 부모는 실제 directory여야 한다. Namespace나 native metadata의 storeDir는
Host 경로 선택에 쓰지 않는다. 기존 parser/binder/공용 sync의 symlink 계약은 변경하지 않는다.

Target sibling의 새0700 workspace에 frozen controls와 node_modules 및
`.heliopause/{manifest.json,sbom.cdx.json,pnpm-provenance.json}`만 exclusive create한다.
생성된 root/parent descriptor와 각 entry inode/mode/hash/link text를 opaque PNPMPreparedProject에
고정한다. File fsync와 모든 actual directory fsync 뒤 전체 descriptor-relative readback으로
foreign/missing/byte/mode/link/hardlink drift를 거부한다. Root/parent 이름 identity도 다시 확인한다.
Directory()/Verify()/Close()는 이 private workspace에만 적용하며 target rename/설치 완료나
independent persistent target approval 권한을 주지 않는다.

Close는 전체 known inventory/identity를 먼저 확인하고 known file/link, 가장 깊은 directory부터
descriptor-relative remove한다. Foreign/identity 불명은 삭제하지 않고 오류와 workspace를 보존하며
모든 descriptor를 닫는다. 생성 실패의 partial file도 실제 만든 inode와 기록 가능한 bytes만
정리하고 확인 불가능하면 보존한다. 취소의 원래 cause와 cleanup failure를 함께 반환한다.
전역 RemoveAll·cache/source/store 삭제나 unknown user files 정리는 없다.

`pnpm_materialize_test.go`는 기존 실제 static/Store/Policy·cache fixture를 즉시 소비한다.
normal/empty·internal link·private copy/single link·cache 변조 전 writes0, payload/records/mode/
symlink/foreign hardlink/unknown entry/root drift와 foreign 보존·late failure/취소를 확인한다.
기존 bootstrap public peer/empty actual consumer에도 Prepare/Verify/Close를 연결해 controls/
node_modules/records readback·target 부재·close 뒤 workspace 부재를 확인한다. Host에서 package
code/native manager는 실행하지 않는다. 이후에 기존 transaction owner의 PNPM fixed profile과
독립 retained approval을 연결하며, 지금 병렬 transaction engine을 만들지 않는다.

검증은 canonical format/Quick/security/docs·promotion/bootstrap 관련 전체 Race·새 actual이다.
새 runtime/observer/source predicate가 없으므로 닫힌 CPP/native baseline을 반복하지 않는다.
Whole P2 acceptance와 사용자 target publication·CLI는 계속 NO다.

## 초기 검토에서 확인한 cleanup 경계

첫 source의 canonical4와 관련 전체 Race는 PASS지만 whole inventory 뒤 삭제 직전에는 inode/
mode/owner만 확인했다. 같은 inode bytes를 initial Verify 뒤 바꾸는 controlled checkpoint의
직접 회귀가 intended assertion으로 FAIL했다. 초기 통과를 이 경쟁 경계의 완료로 쓰지 않는다.
기존 npm member.remove와 같이 per-entry byte/link 확인을 삭제 직전에 다시 수행하고 parent/root
및 각 ancestor directory identity도 재확인한다. 같은 검증 body를 전체 readback과 cleanup에서
재사용하며 syscall 직전의 조건부 unlink를 OS가 제공한다는 주장은 하지 않는다. 변경된 bytes와
foreign data는 보존하고 첫 source/로그는 cleanup-red 원본에 남긴다. Directory fsync도 pinned
descriptor로 수행하며 empty project의 node_modules는 controller가 만든 빈 directory다.

Final fixture 검토에서 symlink 자체의 hardlink는 plain link의 OS별 동작에 기대지 않고
고정 SDK os.Root.Link의 linkat flags0 경로로 만든다. [POSIX link/linkat](https://pubs.opengroup.org/onlinepubs/9699919799.orig/functions/link.html)의
구분과 실제 Go1.27.3 root_unix.go/at_libc.go를 대조했다. Runtime skip/coverage 축소가 아니며
해당 final fixture의 focused Race와 Darwin test compile을 별도로 확인한다. 이 test-only 변경은
이미 전체 Race를 통과한 production source와 실제 bootstrap binary를 바꾸지 않는다. Native
logical copy에는 기존200MiB transport 상한을 적용하고 앞128MiB inspection admission은 그대로다.

## Named local 결과와 다음 경계

초기 source는 format1.421s/Quick150.062s/security10.729s/docs0.233s/관련 세 package 전체
Race131.136s PASS였지만 controlled cleanup-red5.485s가 최초 inventory 뒤 같은 inode의 변경된
bytes 삭제를 확인했다. 초기 성공을 그 경계 해결로 쓰지 않는다. 동일 intended test의
cleanup-green5.431s는 PASS이고 원래 실패 source·로그를 보존했다.

최종 production source는 format0.547s/Quick179.296s/security9.777s/docs0.235s 및
promotion/bootstrap/Evidence 전체 Race193.207s PASS다. 이 뒤 source는 변경하지 않았다.
Test-only symlink hardlink fixture를 고정 SDK Root.Link/linkat flags0로 바꾼 final format0.546s와
새 PNPM preparation 전체 focused Race10.693s도 PASS다. 이 final fixture와 production의 Darwin
arm64 test compile24.130s는 PASS이고 native macOS 실행은 NOT_RUN이다. 범위를 합치지 않는다.

최종 실제 client compile3.929s 뒤 public actual45.074s/test42.58s는 peer4와 explicit empty의
actual Evidence/Policy→operation records/cache→private Host Prepare/Verify/Close를 확인했다.
Controls byte equality, 모든 Host regular file single-link, native 내부 link와 metadata readback,
Close 뒤 owned workspace 부재·사용자 target 부재를 확인했다. Controller의 빈 node_modules와
실제 native layout을 구분하며 empty의 dynamic/native data를 제조하지 않았다.
Helperwait0/CONFIRMED_STOPPED·runtime directory 정리·보호 Host/Docker 구성 전후 동일이다.

Unit은 실제 static/Store/Policy·sealed cache 위에서 normal/empty, native hardlink alias의 독립
Host inode/copy, cache 실패 전 writes0, byte/mode/link/foreign hardlink/unknown/missing/root drift,
initial inventory 뒤 content·ancestor 변경 보존, late failure/취소 cause와 owned cleanup, unknown
file 보존을 확인했다. Root/ancestor/path drift 거부를 atomic conditional unlink 보장으로 확대하지
않는다. 기존 공용 symlink guard·sync·npm transaction은 변경하지 않았다.

[로컬 영수증](./p2c5b5b2-pnpm-private-materialization-local-20261025.json)이 이 named scope의
IMPLEMENTED/WIRED/QUALIFIED=YES를 소유한다. Whole P2 ACCEPTANCE_CLOSED=NO다.
다음은 기존 transaction owner의 typed PNPM profile/독립 retained approval·adoption·rollback이며
CLI/사용자 target publication·whole P2 qualification은 계속 미완료다. CI406은 앞 fd3506b만
포함하므로 이 후속 local source에 결과를 전용하지 않는다.
