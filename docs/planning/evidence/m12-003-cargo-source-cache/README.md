# M12-003 Cargo source/cache 부분 checkpoint

[Result](./result.json)와 [Source](./qualified-source.json)가 실제 실행 범위를 소유한다.

고정 Cargo source와 독립 인증·검사·Evidence·Policy를 소비하는 vendor cache가
연결됐으며 sdist를 포함한 네 실제 직접 소비자와 canonical 검사가 PASS다.
Kernel SocketPair FD authority는 실제 NewFDs 결과이며 guest buffer는 진단 자료다.
Cargo의 독립 초기 한도는 유지한다.

이 기록은 CLI add/build·전체 M12-003 또는 새 CPU/CUDA full qualification이 아니다.
정상·부정 회귀는 저장소의 Cargo source/cache/transaction tests와 observer latches로
재현한다. 원본 전체 로그는 로컬 baseline에 보존하고 위 결과의 hash로 대조한다.
NO PUSH / NO REMOTE CI / NO MERGE.
