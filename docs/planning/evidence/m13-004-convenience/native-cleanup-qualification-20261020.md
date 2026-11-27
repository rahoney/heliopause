# M13-004 — 완료된 native request payload의 명시적 정리 검증

Parent `4e6948f`. [사전 source map/재검토](./session-cli-plan-20261020.md)의
source·staging retention gap만 구현했다. M13-004는 IN_PROGRESS다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED |
| --- | --- | --- | --- | --- |
| Source가 만든 replay Run의 명시적 discard | YES | YES, explicit adapter method | YES, local | YES, component |
| Promotion이 확인한 staging payload의 Close | YES | YES, concrete adapter | YES, local | YES, component |
| 실제 public session에서 actual drain 뒤 호출 | NO | NO | NO | NO |

기존 Source.Close의 의미는 유지한다. Acquire가 만든 copy의 root/directory/asset inode를
기록하고 `DiscardReplayRuns(ctx)`만 그 copy를 제거한다. 다른 Run/Evidence는 대상이 아니다.
Native Promotion은 실제로 검증한 stage/artifacts/file identity를 기록하고 Close에서 exact
owned payload만 제거한다. Final Manifest/SBOM records는 남긴다. Unknown replacement,
추가 파일, link/권한/content drift는 보존하며 정리 오류를 성공으로 덮지 않는다.
새 session은 이 API들을 actual runtime drain 이후 호출하도록 이어서 연결해야 한다.

원본은 `docs/github-logs/self-update-20261020/native-cleanup-*`다. 최종 source inventory와
동일 bytes의 canonical format 이후 focused7.60s, Quick104.04s/security6.45s/
vulnerability11.41s/docs0.24s와 관련 전체 Race127.50s는 모두 exit0이다.
Replay8 cases와 stage6 cases에서 완료 파일 제거/최종 records·unrelated evidence 보존,
unknown 상태의 무삭제, 독립적으로 안전한 다른 Run 정리, 취소·반복 Close/첫 오류 보존을
확인했다. 기존 Source concurrent Acquire/Close와 native Promotion 회귀도 PASS다.
Darwin amd64/arm64의 Artifact test compile도 각6.89s/exit0이며 실제 macOS 실행이 아니다.
완료된 두 compile binary32,942,976B/31,150,002B는 즉시 삭제했다.

이번 검증의 test.TempDir는 종료 시 삭제됐고 completed native test binary는 없다.
실제 설치/config/service/public release 변경은0이다. Signed native4/systemd/crash나
전체 self-update 명령의 성공을 입증하지 않는다. 현재 private source/build cache는 다음
session/CLI 검사에 바로 재사용하며 공유 pinned 도구·원본 근거를 삭제하지 않는다.
