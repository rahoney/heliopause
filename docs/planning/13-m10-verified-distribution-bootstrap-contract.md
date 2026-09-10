# M10 Entry Decision — Verified Distribution & Bootstrap

M10은 source checkout이나 사용자의 local Go build 없이 `helox`와 필요한
observer/runtime을 설치할 수 있는 verified bootstrap chain을 만든다. M8의
Host·observer trust contract와 M9의 transaction contract를 변경하지 않으며,
release artifact의 provenance는 content safety verdict를 대체하지 않는다.

## 1. Bootstrap trust decision

M10은 GitHub Actions가 보호된 release workflow에서 생성한 다음 artifact만
배포 대상으로 인정한다.

- native `helox` binary: 지원 OS/architecture별 별도 asset
- Linux observer helper: gVisor exact source/runtime lock과 연결된 별도 asset
- 필요한 경우 HAA-owned runtime image: GHCR immutable digest reference
- release manifest: asset name, byte size, SHA-256, source commit, workflow run,
  supported platform와 runtime lock identity

`latest`, mutable image tag, branch build, unverified upload, 사용자가 제공한
임의 URL/registry와 ambient credential은 bootstrap 입력이 아니다. manifest와
artifact digest가 일치하지 않거나 provenance가 expected repository/workflow/
commit과 일치하지 않으면 설치를 시작하지 않는다.

GitHub Artifact Attestation은 artifact의 source repository, commit, workflow와
OIDC provenance를 연결하지만 그 자체가 artifact의 안전성 verdict는 아니다.
따라서 installer는 attestation verification, manifest digest verification,
지원 platform/runtime identity 검사를 모두 완료해야 한다. 자세한 upstream
근거는 [GitHub Artifact Attestations](https://docs.github.com/en/actions/concepts/security/artifact-attestations),
[Attestation generation and verification](https://docs.github.com/en/actions/how-tos/secure-your-work/use-artifact-attestations/use-artifact-attestations),
[Offline verification](https://docs.github.com/en/actions/how-tos/secure-your-work/use-artifact-attestations/verify-attestations-offline)를
참조한다.

## 2. Release signing identity ownership

M10 release signing identity의 owner는 `rahoney/heliopause` repository의
protected release workflow와 repository administrators다. product code나
개별 developer workstation은 signing authority가 아니다.

- GitHub Actions OIDC-backed keyless attestation/signing을 primary release
  provenance로 사용한다.
- verifier가 허용하는 signer는 repository, immutable release workflow path,
  protected branch/tag policy와 expected commit provenance에 bound한다.
- signing workflow는 `id-token: write`, `contents: read`와 필요한 attestation
  write 권한만 가지며 pull request/fork build는 release signer가 아니다.
- identity rotation은 기존 trust policy에 새 identity를 overlap window로
  추가하고, published manifest와 verification fixture를 갱신한 뒤에만
  이전 identity를 제거한다.
- compromise/revocation은 release channel을 즉시 quarantine하고 affected
  manifest와 identity를 denylist에 넣는다. installer는 revoked identity,
  unknown signer, expired/invalid proof를 fail-closed한다.

M10은 long-lived private signing key를 product repository secret이나 developer
machine에 배포하지 않는다. keyful signing이 필요해지는 경우 별도 HSM/KMS
ownership, rotation, recovery와 offline trust-root contract를 먼저 추가한다.

## 3. Artifact and image contract

- release manifest가 native binary/helper와 runtime image digest의 single
  source of truth다. CI, installer와 README는 version/digest를 다시
  하드코딩하지 않고 generated manifest를 소비한다.
- GHCR image는 official Node/Python image의 단순 wrapper가 아니라 HAA가
  소유하는 runtime configuration이 필요한 경우에만 생성한다.
- image는 tag가 아니라 immutable digest로 pull하고, image provenance와
  manifest digest가 일치하지 않으면 실행하지 않는다.
- observer helper, runsc, pod-init config, network helper는 서로 호환되는
  exact lock identity 묶음으로 검증한다. 일부만 최신으로 교체하지 않는다.
- SBOM은 release asset와 image에 대해 생성하며, SBOM이 없거나 artifact
  digest와 binding되지 않으면 release를 publish하지 않는다.

## 4. Installer and doctor boundary

M10 installer는 native `helox`가 소유하는 narrow bootstrap operation이다.
installer는 다음 순서만 허용한다.

```text
discover supported host → acquire manifest/artifact → verify digest
→ verify attestation/signing identity → verify platform/runtime lock
→ install into trusted versioned location → atomic activation
→ run helox doctor → publish active version
```

- install/upgrade는 versioned directory와 atomic pointer/activation을 사용하며
  기존 active version을 먼저 삭제하지 않는다.
- download, extraction, signature/provenance, permission, runtime probe,
  rollback 또는 cleanup uncertainty는 active version을 바꾸지 않는다.
- `helox doctor`는 installed binary/helper/image digest, runtime lock,
  observer endpoint ownership, Docker/runsc capability와 required permission을
  검사한다. doctor의 unavailable/incomplete result는 healthy로 표시하지
  않는다.
- installer는 `sudo helox` 전체를 요구하지 않는다. privileged network-policy
  helper 등 M8에서 정의한 좁은 boundary만 별도 installation/permission
  contract를 사용한다.

## 5. M10 work breakdown

| Order | ID | Scope | Status |
| --- | --- | --- | --- |
| 1 | M10-001 | release identity·manifest·bootstrap trust entry decision | COMPLETE |
| 2 | M10-002 | reproducible native/helper build와 provenance-bound release manifest | COMPLETE |
| 3 | M10-003 | runtime image 필요성 결정과 digest/provenance 검증 | COMPLETE |
| 4 | M10-004 | installer·atomic activation·`helox doctor` | COMPLETE |
| 5 | M10-005 | bootstrap rollback·qualification·release gate | COMPLETE |
| 6 | M10-006 | Apache-2.0·CLA policy와 release authority | COMPLETE |
| 7 | M10-007 | verified public release publication workflow implementation과 attestation qualification | COMPLETE |

M10-001은 canonical contract와 ownership만 확정하며 product installer, release
workflow, image, signing secret 또는 runtime placeholder를 생성하지 않았다.

M10-002는 Linux amd64/arm64 및 macOS amd64/arm64 native `helox`, Linux amd64
observer/network-policy helper를 exact runtime lock으로 재현 빌드한다. tag-only
workflow는 immutable SHA-pinned action으로 candidate artifact, CycloneDX SBOM,
manifest와 checksum을 생성하고 GitHub Actions OIDC attestation을 붙인다. 이
workflow는 public GitHub Release를 publish하지 않고 PR CI 권한을 넓히지 않는다.
license/release publication authority가 아직 없으므로 public release asset publish는
M10-005 gate까지 유보한다.

M10-003은 resolver가 사용하는 Node/Python 공식 이미지가 이미 immutable digest로
lock되어 있고 HAA-owned runtime 구성이 필요하지 않은지 검증한다. 필요한 HAA-owned
구성이 없는 동안 공식 이미지를 GHCR에 단순 재포장하지 않으며, 대신 lock-owned
runtime image manifest에 upstream repository, exact digest, runtime version과
provenance 요구사항을 기록한다. 향후 observer/runtime wiring이 HAA-owned image를
실제로 요구할 때만 별도 GHCR image와 provenance publication을 연다.

M10-004는 attestation을 아직 검증하지 않은 임의 local directory를 CLI install
입력으로 받지 않는다. private installer transaction은 이미 expected repository,
workflow, source commit 검증을 통과한 release bundle만 typed boundary로 받고,
manifest·asset·runtime-image manifest digest를 다시 확인한 뒤 versioned user
installation으로 stage한다. activation은 same-filesystem atomic pointer 교체만
사용하며 기존 active version, pre-existing version directory 또는 symlink/digest
mismatch를 덮어쓰지 않는다. `helox doctor`는 install record, selected binary/helper,
runtime image manifest와 Linux Host capability를 independent check로 보고하고,
unavailable·permission 부족·incomplete 상태를 healthy로 축약하지 않는다.

remote release acquisition과 GitHub attestation verifier는 M10-007의 protected
publication scope다. 따라서 M10-004는 caller-controlled path나 boolean flag로 trust를
승격시키는 public install command를 만들지 않는다.

M10-007은 tag push로 생성된 성공한 `Heliopause Release Build` run의 immutable
candidate artifact만 입력으로 받는 manual `workflow_dispatch` 경로다. workflow는
release tag와 source run의 commit을 일치시키고, exact artifact name을 내려받아
manifest·checksum·asset size/digest를 확인한 뒤 GitHub `gh attestation verify`로
각 asset의 SLSA provenance, signer workflow와 source commit을 검증한다. 기존 release
tag가 있거나 어떤 검증/API 응답이 불확실하면 덮어쓰지 않고 중단한다. 게시 job은
protected `release` environment 승인을 필요로 하며, draft release를 만든 뒤
`gh release verify`와 `gh release verify-asset`이 성공할 때만 공개 상태로 전환한다.
release immutability와 protected tag policy는 repository 관리자가 별도로 유지한다.

M10-007의 workflow implementation과 PR/CI qualification은 M10의 완료 evidence다.
실제 `release` environment·tag policy·main Ruleset activation·`develop` branch
전환과 public release publication은 M11 detection depth가 완료된 뒤 M12에서
수행한다. 따라서 M10을 완료했다고 public release가 이미 게시되었다고 해석하지
않는다.

## 6. Acceptance and handoff

- release signing identity owner, rotation, revocation/recovery와 verification
  policy가 명시된다.
- one fact = one manifest owner가 CI/installer/runtime lock에 적용된다.
- mutable latest, unknown provenance, digest mismatch, unsupported host,
  incomplete doctor와 rollback uncertainty가 모두 fail-closed임을 후속
  qualification에서 재현한다.
- `go run ./scripts/check release-gate`는 법적 `LICENSE`, CLA policy와 검증된
  public release publication workflow가 모두 없으면 `unavailable`로 종료하며,
  이 상태에서 public asset을 publish하지 않는다.
- M10-002부터는 이 contract를 먼저 읽고 한 번에 하나의 work item만 연다.

Status: COMPLETE
Evidence: M10-001 canonical contract; M10-002 candidate build and attestation workflow; M10-003 immutable upstream image manifest and exact digest inspection; M10-004 verifier-gated atomic installer transaction and fail-closed doctor; M10-005 rollback/tamper regression and explicit fail-closed release-gate check; M10-006 Apache-2.0 LICENSE, Harmony copyright-license CLA Option Five, contribution provenance audit and hosted CLA status policy; M10-007 protected exact-run publication workflow, PR/CI qualification and attestation/release verification contract
Blocker: none for M10 implementation. Production activation and public publication are intentionally deferred to M12 until M11 is complete.
Next: M11 dynamic detection depth entry decision
