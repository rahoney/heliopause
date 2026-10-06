# M12-002~004 누적 변경의 원격 CI 후속

[결과](./result.json)는 PR29의 source candidate, run/job과 실제 검증 범위를 연결한다.
Qualified source132baea의 run37503579865에서 Required와9 prerequisites 모두 실제 SUCCESS다.
[실제 실행 범위](./actual-scope-summary.json)와 [종료 결과](./qualified-ci-terminal.json)를 확인했다.
종료 문서 전용 commit의 새 HEAD CI는 push 후 별도로 확인하며 main merge는 포함하지 않는다.
M12-002/003/004의 기존 local qualification은 각 canonical evidence를 따른다.

- 첫 macOS 실패는 임시 parent alias fixture의 문제이며 helper의 거부 조건은 유지했다.
- 다음 Linux 실패는 직접 실행한 test binary의 repository cwd와 package-relative Cargo fixture 경로의 불일치다. 파일은 tracked 상태였다.
- 기존 M12-001에 없던 fixture/caller 문제를 이전 green 제품 코드의 회귀로 기록하지 않는다.

[Job capture 목록](./job-captures.json)과 [raw inventory](./raw-inventory.json)는
완료된 job의 실제 decoded UTF-8 본문과 SHA256을 연결한다. Raw bytes는 편집하지 않는다.
[macOS 교정](./macos-fixture-causal-result.json),
[Cargo cwd 교정](./cargo-cwd-causal-result.json)과
[동일 binary의 cwd 비교](./cargo-cwd-paired-result.json)는 부분 재현 범위를 명시한다.
PR merge checkout은 [tree binding](./corrected-merge-binding.json)에서 branch 후보와 대조했다.

이번 ordinary PR CI는 CPU full과 기존·신규 실제 소비자 검증을 포함한다.
CUDA 추가 remote full 및 release support/feature freeze는 별도 범위이며 M12-005는 시작하지 않는다.
성공 테스트 전용 데이터 정리는 원본 Evidence·전체 파일 inventory와 현재 사용 여부를 확인한 뒤 수행한다.
실패 재현 입력·원래 runtime 설치 사본은 보존한다.

[성공 데이터 정리](./cleanup-summary.json)는 네 테스트 전용 root의 삭제와 WSL 가용 공간 약28.2GiB 증가를 연결한다.
Windows VHDX의 실제 파일 축소는 이 측정으로 주장하지 않는다.
