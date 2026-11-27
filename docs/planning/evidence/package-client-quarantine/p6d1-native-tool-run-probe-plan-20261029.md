# P6-D1 — native tool run의 DATA 준비 경계 진단 전 감사

Baseline2bab27c5688f9b1bded432e77aac9c54031668c0/tree24956122bbf97ab31855d5f387354ab23209bb25,
3064 tracked/runtime1368이다. 단일 item은 M13-004다. CI433/source87c817d/1368은 실행 중이며
worker71198/PID3172981이12:22:09UTC부터20분마다 실제 GET을 수행한다. 새 push는 하지 않는다.

Read: PROJECT/Queue/docs routes·lessons INDEX와 artifact L01/L02·sandbox L03/L08/L12/L18·
ci L05/L06·evidence L16, client09§3·plan20§5, P6 baseline/고정 UV0.13.4 source inventories와
run.rs:385..469,1090..1207·CachedEnvironment::from_resolution, 기존 UVResolver/native fixture/
boundary admission/runtime helpers·integrationRunner·observer UVToolQueryEnvironment/
IsExactUVPythonQuery/IsPinnedUVPythonQueryRead를 대조했다.

| 기존 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| UV 고정 공급/native private runtime·network-none 실제 fixture/trace/cleanup | YES | YES | YES (기존 startup/query/local lock probes) | YES (named) | 같은 owner 재사용 |
| UV fresh tool-install·current Python execution workflow | YES | YES | YES (named local; initial hosted432) | YES (named only) | tool-run 의미를 별도 확인 |
| UV tool-run cached environment의 실제 native query 준비 | NO (HAA 지원 판정) | NO | NO | NO | 현 observer에서 실제 첫 실패/도달 경계 확인 |
| 새 tool-run DATA-only diagnostic | NO | NO | NO | NO | tagged test1/actual network-none probe |
| Native public selection equivalence/golden·aliases/CLI | NO | NO | NO | NO | 위 probe와 별도 후속, 이 결과만으로 닫지 않음 |

## Source 전 exact map·두 번째 검토

새 internal/sandbox/uv_tool_run_probe_real_test.go(linux&&realcorpus) 하나만 추가한다.
기존 withActualUVNativeRuntime의 ordinary UID·fixed UV/Python/gVisor/observer·network none·
rootfs readonly·owned volumes와 teardown을 그대로 소비한다. 기존 linuxDynamicProjectWheel의
무의존 local raw wheel을 DATA로 복사하고 `uv tool run --isolated --from haa-uv-probe==1.0`을
offline/no-index/no-build/no-config/no-cache/no-download/고정 Python으로 호출한다. 선택된
프로그램 위치는 상수 `/dev/null`이며 artifact command/shebang/PATH를 호출하지 않는다.
고정 native source의 Command::new absolute path는 실행 불가능한 null device에서 EACCES로
끝나야 한다. Module body는 실행되면 실패하는 raw fixture이며 검증 완료 객체를 만들지 않는다.

Expected terminal은 native `/dev/null` Permission denied의 한정 준비 도달 진단이다. Query나
observer가 먼저 거부하면 실패 원본을 그대로 보존하고 local baseline을 성공으로 바꾸지 않는다.
Expected native error를 검사한 뒤에도 기존 helper의 trace/cleanup 오류는 test를 실패시킨다.
Package safety/Policy·native public golden/selected-program execution 완료를 의미하지 않는다.
Fixture wheel은 public 인증으로 표시하지 않으며 M3/M5/Evidence는 NOT_RUN이다.

예상 runtime1369/new tagged1/기존1368 불변이다. Production/Core/CLI·CPP132·query grammar/
권한·한도·native runtime owner를 변경하지 않는다. Cache path/query compatibility는 실제 근거
없이 확대하지 않는다. Source-derived cache-query alias 미지원 가능성은 가설이며 원인 확정이 아니다.

Canonical format을 먼저 적용하고 동일 default Go membership/bytes로 C2 Quick348.386s와
checker Race62.629s를 명시 재사용한다. 새 excluded tagged test는 tagged vet/pinned Staticcheck/
sandbox test build와 docs/security로 확인한다. 같은 CPP132의 보호 namespace actual에서
원본/Host5/42/ZIP/Docker/source 불변·helperwait0/전체 정리를 기록한다. Actual 실패는 첫 원본
동결 후 해당 책임만 조사한다. 완료 사본은 원본 보존과 full root proc 무참조 뒤 정리한다.
이 진단만으로 새 hosted job/profile·public alias나 production helper를 만들지 않는다.

## V1 첫 actual과 V2 test-only 관측 지도

Canonical6와 default membership/기존1368 동일성 뒤 actual5.32s(operator9.242s)는
OPEN_RESULT_CLASSIFICATION_IMAGE/OTHER·CONTROL·CLONE_CHILD/flags557056에서 실패했다.
Source/Host5/42/ZIP/Docker 보존·helperwait0/전체 정리는 확인했고 expected /dev/null terminal은
미입증이다. FNV literal 후보는 libpython3.14.so.1.0/base Python이지만 실제 execfn/argv와 거부
원인은 미확정이다. 원본 build/actual inventories와 v1 receipt를 보존했다.

V2는 같은 새 tagged file 하나에 test-only recorder를 추가한다. 기존 admission-aware wrapper의
검증된 underlying Host executor Output을 감싸 raw native error/stderr를 제한된 test 진단으로
보존하고 동일 admission wrapper/세션으로 실행한다. DirectExecAdmission.Output의 finishErr
우선 반환 때문에 v1에는 underlying command error 유무가 기록되지 않았다. 이를 특정 native
오류가 실제 소실됐다는 결론으로 쓰지 않는다. UV verbose2만 더해 실제 query/execfn을 확인하며
원래 observer refusal·원래 결과/teardown 실패는 그대로 실패시킨다. 이는 원인 수정/권한 확대가
아니며 CPP132/production은 유지한다. 추가된 진단 source만 tagged checks/build로 검증한 뒤
새 actual을 수행한다. CI433 poll01은8SUCCESS/Linux 진행, next12:42:09UTC이며 새 push는 없다.

## V2 결과와 실제 다음 missing

Same CPP132/current1369의 v2 actual4.77s(operator6.808s)는 여전히 observer FAIL이다. 다만
raw native exit2/stderr4683B는 cached archive-v0 Python query 뒤 /dev/null EACCES까지
도달했음을 확인했다. 이 native 오류는 Output의 finishErr 우선 반환 뒤 사라졌다. 기존
RunDiscard는 두 오류를 join하므로 다음 P6-D2는 같은 owner의 나머지 네 method만 교정한다.
관찰기 실패를 지우거나 cached query compatibility의 원인 해결로 합치지 않는다. 그 alias의
실제 kernel binding/음성 사례·native public golden/CLI/final은 별도 missing이다.

첫 두 cleanup은 mountinfo EINVAL에서 삭제0으로 멈췄다. 정확한6PID/starttime의 stateZ/FD0/
maps0/cmdline0/exe·cwd·root·mnt ENOENT를 전후 재검증해 구분한 root83proc/ref0/errors0 뒤
완료 ELF4개61940507B만 삭제했고 모든 원본/CI433 monitor를 보존했다. 누적213/~7.18GiB
Linux 논리 용량이며 Windows VHD 축소가 아니다.
