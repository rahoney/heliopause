# P2c5-B5b1 — 실제 operation의 승인 records와 peer provenance

Baseline fd3506b/tree6307df7976c427324bca0a053234db6718339ee7. 같은 M13-004/P2다.
CI406/run40117566240은 이 baseline의 원격 검증이며 첫 확인03:40:25 KST 전 재조회하지 않는다.
B5a cache와 B4d 호출 교정을 대체하지 않고 B5b의 records gap만 먼저 연결한다.

Read: Queue·B5a/B4d leaf, client 설계§4·확장 계획§3/5, Architecture02,
Domain09/13, 실제 Evidence Generator.BuildProject/기존 PythonProjectCache 소비자,
artifact L15/L19/L20·sandbox L08/L12·evidence L11/L16·ci L04/L06.

## Source 작성 전 baseline audit

| 책임 / 기존 symbol | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| actual source·static-all·instance Evidence/Policy | YES | YES | YES(named local/public) | NO(P2) | 이번 변경 없음 |
| bounded sealed PNPM data cache·actual record recheck | YES | YES | YES(B5a named local) | NO(P2) | operation/Manifest/SBOM와 full peer provenance |
| project Manifest/SBOM — evidence.Generator.BuildProject | YES | YES(other clients) | YES(existing) | NO(P2) | PNPM의 실제 operation과 typed lock projection 소비 |
| full native peer context — PNPMInspectionContext.Lock/Controls | YES | YES(inspection) | YES(named local) | NO(P2) | records에 손실 없는 결합 |
| PNPM persistent independent approval / Host extraction / transaction / CLI | NO | NO | NO | NO | B5b 다음 단위; 이번에 placeholder 생성 안 함 |

## 정확한 edit map

`promotion.PNPMVerifiedCache` 생성자는 기존 ProjectRecordBuilder와 실제 OperationID를 필수로
받는다. 기존 Stage/Open API와 승인·sources·data tar·sync/no-replace publish 의미는 유지한다.
`pnpm_cache_records.go`는 현재 frozen snapshot의 unique archive별 transitive record node와
그 instance dependencies를 union한 archive edge를 기존 Generator에 전달한다. Self archive
edge는 unique archive graph에 표현하지 않으며 완전한 instance별 edge·peer context·root
selector·package metadata는 별도 typed provenance JSON으로 빠짐없이 보존한다. Fake primary나
한 instance를 대표하는 다른 instance의 성공을 만들지 않는다. 원본 두 controls는 Manifest에도
그대로 결합되므로 full native lock과 source-neutral approval digest 관계를 추적할 수 있다.

Provenance는 실제 operation/Manifest ID/graph/runtime/recorded layout digest와
PNPMLock.Roots/Packages/Snapshots 전체를 결합한다. Guest path를 Host authority로 사용하지
않는다. RecordPath는 native package key이며 존재하지 않는 Host/archive 파일 경로를 주장하지
않는다. 중복 archive edges만 합치며 peer instances는 합치지 않는다.

새 records는 data와 분리된 `records/{manifest.json,sbom.cdx.json,pnpm-provenance.json}`에
exclusive write·read-only seal한다. 공용 regular-only inventory/sync는 그대로 재사용한다.
각 Manifest/SBOM32MiB는 기존 Generator 한도, provenance4MiB는 기존 PNPM receipt 한도이며
native archive·controls·entry 예산을 늘리지 않는다. Receipt schema2가 operation/Manifest ID와
records 전체 inventory를 결합한다. Cache hit/Open은 현재 actual Evidence/Policy뿐 아니라
같은 frozen controls/lock에서 records를 다시 생성해 전체 bytes/digest/mode를 재확인한다.
이전 미출시 schema1 cache는 승격하지 않고 새 Stage를 요구한다.

`promotion/pnpm_cache_test.go`는 실제 Generator와 새 operation을 사용한다. 기존 모든
staging/tamper/late cleanup 회귀를 유지하고 record tamper·누락·foreign·operation mismatch와
generator 실패 전 writes0, dependency-free records를 확인한다. `bootstrap`의 기존 public
peer/empty actual consumer는 실제 operation ID 하나를 ProjectInspectService factory와 cache에
함께 전달한다. 실제 Manifest/SBOM·full peer context 및 재Open/drift 거부까지 검사한다.

## Source 작성 전 두 번째 검토

기존 Generic Generator는 unique archive graph와 complete project records를 소유한다.
PNPM instance graph를 그 Core node 모델로 억지 flatten하지 않고 native provenance leaf로
원래 정보를 보존한다. Existing generator/다른 client 계약은 수정하지 않는다. Policy 승인 없는
record 생성·operation ID 사후 제조·cache receipt만 보고 ALLOW 재사용은 금지한다.
Source bytes 재확인과 typed stored Evidence gate는 모든 cache writes보다 앞선다.

먼저 canonical format/Quick/security/docs와 관련 promotion/bootstrap/Evidence 회귀를 실행한다.
Actual public consumer는 같은 authenticated client/helper·mount namespace/Host 불변·정리를
검증한다. 새 observer/runtime/source guard 변경은 없어서 닫힌 CPP/기존 native consumers는
반복하지 않는다. 원격406과 이 뒤 source의 검증 범위를 분리하며 전체 P2 acceptance는 NO다.

## 검증 결과와 남은 경계

첫 ready는 format1.431s/Quick152.753s/security9.797s/docs0.239s PASS다. Record graph의
package lookup/edge dedup을 bounded map으로 선형화한 최종 production source는 format0.545s/
Quick197.931s/security9.772s/docs0.234s와 promotion/bootstrap/Evidence 전체 Race216.546s PASS다.
첫 ready의 이전 source도 raw에 별도 보존했다. 실제 source/검사 범위를 합치지 않는다.

새 realcorpus 소비자 첫 compile2.774s는 같은 block의 기존 manifest control 변수와 decoded
Manifest 변수 이름 충돌로 FAIL이었다. Native는 미실행이다. Decoded test 변수만 recordManifest로
고쳐 actual compile3.428s와 final canonical format0.575s를 통과했다. Default/Race compile selection은
realcorpus 파일을 제외하며 final production source는 그 검사 뒤 바뀌지 않았다. 첫 실패 source/로그와
교정 notice를 보존했다. 이 실패를 production/기존 검사 실패로 합치지 않는다.

Actual44.668s/test42.22s는 public peer4·실제 Evidence24개/static-all/모든 instance/Policy ALLOW를
새 schema2 cache와 실제 operation Manifest/SBOM/full root·package·peer instance provenance의
Stage/Open에 연결했다. 원래 lock state 전체 JSON bytes와 record operation/graph/runtime/Manifest
결합을 대조했고 data tar 변조는 거부했다. Explicit empty도 실제 operation records를 만들었지만
per-entry Evidence/native layout/dynamic instance는 만들지 않았다. Header true를 empty 미실행
범위에 전용하지 않는다. Helperwait0/CONFIRMED_STOPPED·runtime directory 정리와 보호 Host
파일/Docker 구성 전후 동일이다. Host project는 계속 부재다.

[로컬 영수증](./p2c5b5b1-pnpm-project-records-local-20261025.json)은 이 named scope의
IMPLEMENTED/WIRED/QUALIFIED=YES, ACCEPTANCE_CLOSED=NO(whole P2)를 기록한다.
CI406은 앞 fd3506b만 포함한다. 이 로컬 records source를 그 원격 성공으로 전용하거나 작은
helper만 재push하지 않는다. 다음은 Host private materialization/independent retained approval과
기존 transaction 재사용이며 whole P2/CLI qualification은 열려 있다.
