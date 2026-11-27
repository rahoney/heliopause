# M13-004 — 설치 도우미의 실제 release producer 연결 검증

Parent `5ac400b`. [사전 map/두 번째 검토](./producer-plan-20261020.md)를 따랐다.
기존 tag-only build에 실제 Linux amd64 설치 도우미 build와 manifest binding을 연결했다.
공유 assetSpecs에서 새 release의9 assets/전체13 files가 파생된다. 기존 v0.1.0의
실제8 assets/12 files 증거는 변경하지 않았다. Publish 소비자는 기존 exact bundle
verifier를 그대로 사용한다. Deb/RPM에는 root0500 payload만 포함하며 실행·등록은 없다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED |
| --- | --- | --- | --- | --- |
| 실제 helper build/manifest/package payload | YES | YES | YES, local | YES, source component |
| 새 signed native9 bootstrap/activation | PARTIAL | NO, initial loader | NO | NO |

원본은 `docs/github-logs/self-update-20261020/producer-*`다. 동일 source의 focused37.14s,
canonical Quick70.20s/workflow0.68s/security5.43s/vulnerability10.82s/docs0.24s/
release-gate0.33s와 변경 generator/checker Race43.60s는 모두 exit0이다.
Missing/tampered helper·잘못된 role와 commented/echo-only build·누락 binding을 거부하고,
OS package의 exact bytes/root0500/재현성/disabled service/설정 미생성을 확인했다.

실제 Go1.26.9 Linux CLI build12.69s와 installer build1.14s가 exit0이다.
Installer의71개 module tuple은 CLI의82개에 모두 포함되며 누락은0이다. 무인자/help/
version은 inert exit0이다. 완료 출력42,316,598B와27,111,115B, 전용 work directory와
one-shot operator를 삭제했다. 최초 inline operator의 문법 오류는 build 이전 실패이며
제품 검사 실패나 성공으로 집계하지 않는다. Public tag/publish/설치 전환은 실행하지 않았다.

두 번째 대조에서 producer·strict parser·bundle/checksum·publish·package 소비자와
SBOM 의존성 경계를 확인했다. OS package의 `/releases/`를 native `/versions/`로
자동 인수하지 않는다. 다음은 최초 native 등록/verified loader 및 실제 signed base/target,
systemd/crash/reboot qualification이다. M13-004는 IN_PROGRESS다.
