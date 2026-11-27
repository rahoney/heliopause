# P1b4 — immutable UV native 공급·private tool 전달 baseline

기준 source270c7c834d75cc7f62936c87a5c19606b72d112f. M13-004/P1만 IN_PROGRESS다.
Go 진단 PR366/run39605808634와 독립 cu126/run39605942685는 같은 source로 진행 중이다.
기존 cu130의 Linux/full980.48s는 PASS지만 native macOS production build2m timeout으로
Required는 FAIL이다. 기존 CI361/cu126/cu132의 같은 c2c1750 native PASS도 보존하며 SDK
인과를 단정하지 않는다. 원격 결과를 기다리는 동안 독립적인 data-only 공급 gap을 수행한다.

Read: [P1b 수정 지도](./p1b-uv-runtime-baseline-20261021.md),
[canonical UV 경계](../../../user-journey-cli-ia/09-package-client-quarantine.md),
[유지보수](../../../engineering/05-runtime-upgrade-maintenance.md),
[D-012/D-015](../../../threat-model/04-trusted-tooling-and-evidence.md).

| 현재 owner | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| native UV lock/graph·signed 전체 공급·schema2 pin/maintenance | YES | YES(자료/개발 도구) | NO | NO | production runtime/resolver/CLI |
| CommandRunner + RunOutput/RunInput + Docker cp | YES | YES | YES(기존 소비자) | YES(기존 scope) | UV single-file bounded reader/writer |
| boundaryHelper root tmpfs·demotion·exact admission | YES | YES | YES(기존 scope) | YES(기존 scope) | UV-only64MiB mount/profile와 새 exact class |
| UV native immutable supply/private tool 전달 | NO | NO | NO | NO | never-start supply create/cp·root/mode/hash·cleanup·target 재대조 |

## 정확한 수정과 경계

1. internal/sandbox의 같은 package에 uv_tool_supply.go를 추가한다. Runtimeidentity의
   image/config/native size/hash만 소비한다. Image inspect가 pinned config/amd64와
   다르면 create 전에 거부한다. 공급 container는 고정 image·network none·read-only·
   cap-drop ALL·고유 controller name/label로 create하고 절대 start하지 않는다.
2. Create 결과가 모호하면 고유 name의 실제 inspect와 label/config/created state를
   대조한 경우만 exact ID를 정리한다. 조회 오류를 부재로 처리하거나 unknown ID/name을
   force 삭제하지 않는다. 정상 경로도 마지막 소비 뒤 같은 owned created state를 확인해
   remove한다. Cleanup failure는 bytes 준비 성공을 무효화하고 첫 오류를 보존한다.
3. 기존 streamingDockerOutput으로 /uv의 tar를 전체 native size+64KiB 안에서 받는다.
   새 bounded writer는 Buffer/ReadFrom/WriteString을 embed하지 않는다. 하나의 regular
   uv·root0:0·0755·정확한 크기/hash와 archive 종료를 검증한다. 버전/hash literal을 새로
   쓰지 않으며 Host 실행·Host path bind mount·대상 artifact 입력은 없다.
4. 동일 bytes만 root0:0/0555의 tar로 기존 RunInput/docker cp --archive를 통해 고정된
   /haa-runtime/uv로 전달하고 다시 tar/hash/ownership을 대조한다. 이 전달 API만으로
   target tmpfs·observer admission이나 실제 실행을 승인하지 않는다. UV-only mount와
   classifier/호환성은 다음 실제 runtime 단계이며 기존 pip 경로 예산은 유지한다.
5. Unit은 작은 synthetic bytes의 구조 경계와 image/config/label/state·중복/link/path·
   크기/hash·출력 폭주·취소·cleanup 실패를 검사한다. 실제 signed-native/Docker copy는
   별도 realcorpus scope다. Default fake 성공으로 native 실행·production 지원을 닫지 않는다.
   Canonical format/Quick/security/docs 및 영향을 받는 source qualification을 연결한다.
6. P1b4 결과와 미완료를 Queue에 기록하고 한글 중간 커밋으로 보존한다. 새 source push는
   진행 중인 진단 CI의 고정 입력/결과를 보존한 후 판단한다. 공급 image/layer는 다음 실제
   runtime 소비에만 보존하며 마지막 소비가 끝난 bulk는 참조 확인 후 삭제한다.

## 두 번째 검토

기존 helper1MiB tmpfs에43MB UV를 넣지 않는다. 같은 name/label만으로 공급 tool을 믿지
않고 pinned config/native hash를 확인한다. Source cp의0755와 target root-private0555를
분리한다. Archive Writer는 명시 Write만 제공하고 stream bound 전체를 확인한다.
Create/cleanup uncertainty를 성공한 부재로 바꾸지 않는다. 새 profile을 임의 CONTROL이나
unknown-exec 예외에 연결하지 않으며 실제 실행 전에 별도 exact admission을 검증한다.

## 로컬 결과

[영수증](./p1b4-uv-provision-local-20261021.json)의 canonical Quick25.915s/security5.104s/
docs0.202s와 malformed/substitution/ownership/state/cleanup 회귀가 PASS다. 실제 signed
image의 data-only Docker 시험도 UID1000의 strict trusted executor/private runtime view에서
3.39s PASS다. Source0755→target0555/root0:0과 native42,544,544B/hash를 확인했다.
두 owned container 제거·Host protected files4/기존 image records6의 불변도 확인했다.

Target은 정지된 writable-root data fixture다. 이 결과는 실제 tmpfs/admission·UV 실행·
production 지원 증거가 아니다. IMPLEMENTED=YES/WIRED=NO(production)/QUALIFIED=NO/
ACCEPTANCE_CLOSED=NO를 유지한다. 먼저 새 Go image가 Host에 있을 것으로 가정한
operator preflight 실패를 보존했고 exact pin absence와 실제 기존 inventory로 교정했다.
기존 Host 설치·이미지를 갱신해 이 시험을 통과시키지 않았다.

다음은 UV-only tmpfs64MiB와 fixed UV class/admission·실제 native startup이다. 기존
pip helper1MiB·Python ABI·root demotion·nonce/generation과 ARTIFACT 권한은 유지한다.
완료한 private namespace runtime 사본·test binary와 마지막 소비가 끝난 upstream layer는
작은 원본을 보존한 뒤 삭제한다. 실제 다음 소비가 있는 UV Docker image만 유지한다.

최종 전달 후보의 Quick24.339s/docs0.215s/security4.685s도 PASS다. 완료된 private
runtime 사본·test binary·두 compressed layer20항목/unique regular logical368,428,870B를
참조/query errors0 확인 뒤 삭제했다. Namespace/원본/해시·실행·Host 불변 영수증은 보존하며
활성 source/SDK/cache/base runtime과 다음 소비의 UV Docker image만 유지한다.
