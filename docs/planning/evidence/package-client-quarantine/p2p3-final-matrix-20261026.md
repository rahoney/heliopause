# P2/P3 최종 지원·실패·영향 profile matrix

기준 frozen candidate는 `50fd3b2964669f27793c30caa0b2b61113df3e00`, tree
`d6a63abdc426da4065fdfa248151ad64323a29dc`/2475 files다. 이후 P4 로컬 변경은 이
qualification에 포함하지 않는다. 아래는 acceptance 감사이며 추가 runtime 예외/한도/
CI job/승인 관문을 만들지 않는다. Canonical 지원 구문은 client 설계§9/§10을 따른다.

| 실제 acceptance | PNPM P2 | Bun P3 | 남은 확인 |
| --- | --- | --- | --- |
| 공개 native selection·독립 source/SRI·전체 static inventory/graph | CI408/409/410; CI411 Linux named 성공 | B3c/B4 actual named local·CI412 backend | CI413 whole current source의 원본 |
| fresh 전체 physical instances·exact coverage·현재 recorded Evidence/Policy | 기존 whole backend 및 CLI 검증 | B6b 실제 Bun JS normal/typed37, B6c public2/4 whole current Policy | CI413 native JS/whole/project consumers |
| 승인 cache·private output·guarded transaction | B5a/b·CI407 이후 | B5a/b·기존 shared owner 재사용 | CI413 whole current source |
| add/install/selected update/full update/no-op/명시 empty | named local9operations·CI408 이후 | B5c/B6c named local9operations | CI413 hosted CLI/native lane |
| native nonempty copy original source proof/명시 adoption | B5d source9410·CI408 이후 | B5d/B6c3cases·bounded ownership proof | CI413 hosted original lane |
| symlink/hardlink escape·foreign files·concurrent ownership·corrupt state | 기존 exact fixture/negative·failure leaf | typed Bun fixture 및 동일 guard source/negative | 신규 동작 gap 발견 시 해당 경계만 추가 |
| actual SIGINT·SIGKILL·ENOSPC·rollback/재진입 | [P2 failure leaf](./p2c5b5e-pnpm-failure-local-20261025.json) | [P3 failure leaf](./p3b5f-bun-failure-local-20261025.json) | 이 named 경계는 검증됨; whole remote 결과에 전용하지 않음 |
| failure cause·불완전 observation·cleanup/Host baseline | named actual 원본별 helperwait0/Host 동일 | B6b/B6c actual helperwait0/Host 동일 | CI413 full 원본·최종 helper/client 복원 |
| native macOS CLI/build/default checks | CI412 성공, 과거411 timeout 원인 미확정 | CI412 backend까지, B5/B6 source는 CI413 대기 | CI413 macOS + Required |
| CPU/full + observer canonical | 기존 qualified source 각 증거 | observer121 latches·B6b canonical/actual named local | CI413 current whole source CPU/full·121 |
| cu126/full | CI409 source9410 named remote success | 이후 공통 observer/native Bun source 미검증 | frozen current source 필수 영향 qualification |
| cu130/full | CI410 source9410 named remote success | 이후 공통 observer/native Bun source 미검증 | frozen current source 필수 영향 qualification |
| cu132/full | CI411 Linux 실제 성공, overall/macOS+Required 실패 | 이후 공통 observer/native Bun source 미검증 | frozen current source Required 포함 qualification |

## 상태와 실행 선택

P2/P3 기존 기능은 IMPLEMENTED/WIRED=YES이며 위 named qualification을 보존한다.
Whole P2/P3 ACCEPTANCE_CLOSED=NO다. CI413은 grouped backend/cache/transaction/CLI/
native runtime 소비자와 ordinary CPU/full을 한 번 확인하는 현재 실행이다. Actual75분
첫 timer due에 run/head·각 status/conclusion·원본10logs·checkout whole tree를 확인한다.
모든 job success를 확인하기 전에는 all green으로 쓰지 않는다.

공통 observer의 Bun Artifact image/ABI read/strict negative 변경은 CPU와 세 CUDA
소비자를 건드린다. 확장 계획§5와 기존 invariant에 따라 ordinary 성공 뒤 frozen 동일
source에서 필요한 cu126/cu130/cu132를 각각 한 번 순차 검증한다. 이전9410 CUDA 성공과
현재121-latch 결과를 합쳐 최신 source의 full profile 성공이라고 주장하지 않는다.
새 P4 helper commit으로 이 원격 후보를 취소하거나 각 helper별 CI를 실행하지 않는다.
각 profile 최근 실제 elapsed와 추가 Bun project519.694s/JS·준비 여유로 다음 실제 due를
등록하고 active turn에서 결과를 소비한다. 실패하면 최초 exact logs/source를 보존하고
원인/영향을 검토하며 같은 입력의 맹목적 재실행·timeout 확대를 하지 않는다.

첫 lane에서 미지원인 workspace/global/local/VCS/direct URL/임의 native 옵션·native empty
adoption·shared hardlink/다른 ABI/OS를 성공으로 표시하지 않는다. 이 명시 거부와 미래
목표의 구현 미완료를 구분한다. Bunx/PNPM dlx/P4 persistent tools는 이 matrix 밖의 후속
단계다. 새 발견 없는 기존 구현·해결된 문제를 재감사하거나 전체 로그를 복제하지 않는다.
