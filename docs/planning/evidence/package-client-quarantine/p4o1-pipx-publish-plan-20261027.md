# P4-O1 — pipx publish proof의 한정 offline 검증

M13-004의 현재 missing 공급 경계만 구현한다. 기준 HEAD27caa9a/tree95a5f475이며
전체 P4-O baseline과 CI417의 기존 UV qualification을 승계한다. 새로운 runtime 선택이나
지원 선언은 하지 않는다. Read는 [P4-O baseline](./p4o-pipx-supply-runtime-baseline-20261027.md),
Queue/plan20/client09, D-012/013/015, maintenance §2·3·6·7, Architecture와
artifact/sandbox/ci/evidence lessons다.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| Pinned Sigstore public root/SCT/tlog/timestamp 검증 | YES | YES(existing providers) | YES(existing scopes) | YES(existing scopes) | 같은 primitive의 별도 PyPI publish 정책 소비 |
| Pipx typed metadata/whole wheel binder | YES | YES(pure parser) | YES(L1/L2 named) | YES(parser/binder only) | 이후 runtime 소비; 재구현하지 않음 |
| Bounded PEP740 bridge/fixed pipx publisher receipt | NO | NO | NO | NO | O1의 exact parsing/crypto/certificate/statement 검증 |
| Offline developer supply command | NO(pipx) | NO | NO | NO | 기존 update-runtime dispatch/frozen reader에 O1 연결 |
| Immutable dependency closure/loader/runtime/backend/CLI | NO(pipx) | NO | NO | NO | O2 이후; O1 성공으로 닫지 않음 |

## 구체 수정 지도

- `internal/verification/releaseprovenance/pipx_publish*.go`: 기존 publicRoot와 pinned
  Sigstore를 재사용한다. Proof ≤1MiB, strict UTF-8/duplicate/depth/token/key bounds,
  exact-case fields, 한 bundle/attestation/log의 명시적 subset만 받는다. Canonical base64의
  certificate ≤16KiB·statement ≤64KiB·signature ≤256B를 그대로 표준 bundle에 표현한다.
  Fixed pypa/pipx repo151871286/owner647025, release.yml/tag/source/run, GitHub-hosted,
  SCT1/log1/observer timestamp1와 ECDSA P256를 검증한다. 서명 검증 후 certificate OID23의
  release와 OID24의 repo:pypa/pipx:environment:release를 정확한 DER UTF8String으로 확인한다.
  Statement는 단일 wheel filename/SHA256, publish/v1과 absent/null/empty predicate만 받는다.
  Error는 zero receipt이며 caller가 임의 publisher policy를 주입하지 못한다.
- `scripts/update-runtime/pipx_publish.go`와 기존 main dispatch: 명시 absolute frozen
  directory의 identity/provenance/wheel만 bounded read하고 bytes SHA를 먼저 대조한다.
  Offline facts만 출력하며 download/extract/Host Python/설치/활성화/pin을 수행하지 않는다.
- 작은 actual official proof/identity fixture와 focused negative tests: 정상 crypto와
  wrong identity/digest/upstream, malformed/duplicate/case/unknown/trailing/bounds,
  signature/log 변조, certificate environment/subject guards를 확인한다. 기본 test는
  개인 /tmp나 network를 숨은 입력으로 사용하지 않는다. Wheel은 기본 fixture에 추가하지 않는다.

## 두 번째 검토와 검증 경계

기존 HAA/UV/PNPM/Bun SLSA 정책은 변경하지 않는다. Artifact private JSON helper를
import해 dependency 방향을 바꾸거나 새 공통 package를 만들지 않고 scoped decoder를 둔다.
PEP740 publisher metadata는 untrusted이며 signed certificate와 별도로 대조한다.
OID23/24의 DER UTF8String 의미는 [Fulcio 공식 명세](https://github.com/sigstore/fulcio/blob/main/docs/oid-info.md)를
확인했다. 원본 fetched content는 local evidence로 보존하고 raw를 수정하지 않는다.
단일 wheel proof를 pyz/dependency closure/runtime 승인으로 전용하지 않는다.

Canonical format/Quick, 해당 두 package의 full Race, docs/security와 frozen supply actual
command를 순차 실행한다. 전체 후보 index blob/mode/disk/tree를 대조하고 의미 있는 중간
commit으로 보존한다. Helper별 새 push/full CI는 하지 않으며 O2의 concrete edit map으로
계속한다. M13-004 하나만 IN_PROGRESS이고 whole P4/M13 acceptance는 OPEN이다.

## 한정 로컬 결과

[O1 qualification receipt](./p4o1-pipx-publish-local-20261027.json)가 최종 source hashes,
정상·부정 회귀/전체 affected Race·actual bytes와 원본 inventory를 소유한다. O1은
IMPLEMENTED=YES/WIRED=YES(developer command)/QUALIFIED=YES(named local)/
ACCEPTANCE_CLOSED=YES(O1 only)다. 기존 parser/binder/runtime/publisher policy는 대체하지
않았다. Whole pipx 지원·closure/runtime/hosted acceptance는 계속 미완료다.
