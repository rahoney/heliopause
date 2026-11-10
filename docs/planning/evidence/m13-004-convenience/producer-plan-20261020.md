# M13-004 — 실제 installation-role payload의 release producer 연결

기준 `5ac400b`. [Session/CLI 검증](./session-cli-qualification-20261020.md) 뒤
남은 producer gap이다. 기존 [M13 release contract](../../18-m13-production-release-operations-contract.md)와
[M10](../../13-m10-verified-distribution-bootstrap-contract.md)을 유지한다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| canonical manifest/checksum/exact bundle/publish consumers | YES | YES, 기존8 assets | YES, 이전 범위 | YES, 이전 범위 | map에 실제 새 helper 추가 |
| `cmd/haa-release-installer` | YES | YES, native client/server | PARTIAL, local | NO, signed lifecycle | 실제 build/배포 asset |
| native build/manifest와 package의 installer payload | NO | NO | NO | NO | 아래 좁은 변경 |
| 최초 native bootstrap/registration/실제 signed transition | NO | NO | NO | NO | 다음 단위, producer 성공과 구분 |

## 수정 위치

- `.github/workflows/heliopause-release-build.yml`: 기존 deterministic native step에
  CGO0/linux/amd64/trimpath/buildvcs의 실제 `cmd/haa-release-installer` build와
  manifest `--asset`를 추가한다. 기존 job/permissions/tag-only/OIDC identity는 유지한다.
- `scripts/release-manifest/main.go`: assetSpecs에 기존 parser가 이미 아는 basename
  `haa-release-installer-linux-amd64`, platform linux/amd64, role release-installer를
  추가한다. 기존 collectAssets/checksum/verifyBundle이 같은 map을 소비하므로 별도의
  숫자·publish file list를 만들지 않는다. 새 producer는9 assets/전체13 files를 생성하며
  기존 실제 v0.1.0의8 assets/12 files 증거를 수정하지 않는다.
- `scripts/release-packages`: OS package의 기존 versioned payload에 Root-only0500의
  installer binary를 보관한다. 자동 실행/enrollment/service enable은 없다. 기존 OS
  package와 native `/versions/` 관리 설치의 구분을 README/계약에 명시한다. Package DB를
  우회한 self-update를 켜지 않으며 공식 npm/PyPI 이름을 새로 정하지 않는다.
- `scripts/check/workflow.go`: 실제 build와 manifest 인수가 active release step에
  있어야 한다. Comment/echo 위치만 있는 producer, 누락 build/누락 binding을 거부한다.
  기존 publish는 exact-tag source의 manifest verifier 목록을 소비하므로 새 배열이나
  새 workflow/job을 만들지 않는다. 해당 소비자는 그대로 회귀한다.
- SBOM은 primary Linux CLI의 실제 build info를 사용한다. Installer command는 stdlib와
  동일 releaseactivation package만 import하며 CLI도 그 package를 사용한다. 같은 platform의
  helper module dependencies가 CLI SBOM의 부분집합인지 actual build info로 확인한다.
  새 provenance/safety generator를 만들지 않는다.

## 검사와 두 번째 검토

Canonical Quick/workflow/security/docs/vulnerability/release-gate를 사용한다. 실제 helper
basename/role/platform/size/digest/13-file exact directory와 missing/tampered/helper role
거부를 확인한다. Deb/RPM의 root ownership/0500/exact bytes와 기존 reproducibility,
disabled service/no caller config를 검사한다. 변경된 generator/checker Race를 확인한다.
필요한 실제 native build output은 마지막 사용 직후 삭제하며 기존 pinned observer
source/latch 검증을 변경 없는 상태로 불필요하게 반복하지 않는다.

두 번째 대조에서 `bundle.go`/publish의 file list가 assetSpecs와 metadata에서 파생되는
것을 확인했다. 동일 목록을 다른 곳에 하드코딩하지 않는다. OS package 경로는 기존
`/usr/lib/heliopause/releases/<version>/`이고 native updater는 `/versions/`이므로 helper를
포함했다는 이유로 package 설치를 native 관리 설치로 인수하지 않는다. 실제 새 signed
native4/systemd/crash와 최초 등록은 계속 미완료이며 tag/publish/main merge/영구 host
activation을 이 source 변경으로 실행하지 않는다.
