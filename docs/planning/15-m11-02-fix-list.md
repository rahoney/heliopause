# HAA M11 이후 Release Hardening Fix List

- 파일명: `15-m11-02-fix-list.md`
- 기준 저장소: `rahoney/heliopause`
- 검토 기준 commit: `b3e8ec68783d3c647bfe7830f5483902575780f3`
- 상태: M11 완료 후, M12 public release 전에 처리할 bounded fix
- 원칙: 아래 5개 항목만 처리한다. 새로운 아키텍처 재설계나 탐지 알고리즘 확대는 이 문서의 범위가 아니다.

## 목적

M11 완료 시점의 구현은 public release를 진행할 수 있는 수준까지 도달했다. 다만 release activation과 publication trust chain에서 실제 수정 가치가 있는 4개 bounded issue와 observer timing bug 1개가 남아 있다.

이 문서는 해당 5개 항목의 **수정 위치, 수정 방법, 필요한 regression test, 완료 기준**을 고정한다.

---

## FIX-01. Release activation 실패 시 `current`가 새 버전으로 남을 수 있는 문제

**중요도: High — release 전 수정**

### 위치
- `internal/releaseinstall/installer.go`
- 함수: `activateCurrent(root, version string)`
- 테스트: `internal/releaseinstall/installer_test.go`

### 문제
현재 흐름은 대략 다음과 같다.

```go
os.Symlink(...)
os.Rename(nextPath, current)
syncDirectory(root)
```

`Rename()` 성공 뒤 마지막 `syncDirectory(root)`가 실패하면 함수는 error를 반환하지만 `current`는 이미 새 버전을 가리킬 수 있다.

```text
Install() = 실패
current   = 새 버전
```

이는 activation failure/uncertainty에서 기존 active version을 유지한다는 계약과 충돌한다.

### 수정
1. activation 시작 전에 기존 `current`를 snapshot한다.
   - `current` 없음
   - 또는 검증된 `versions/<old-version>` symlink
   - 그 외 상태는 즉시 거부한다.
2. 새 symlink를 temporary path에 만든 뒤 현재처럼 atomic rename한다.
3. rename 후 `syncDirectory(root)` 실패 시 이전 상태를 복원한다.
   - 기존 pointer가 있었다면 rollback temporary symlink를 만들어 `current`로 rename 후 다시 sync.
   - 기존 pointer가 없었다면 새 `current`를 제거 후 다시 sync.
4. rollback 또는 rollback sync도 실패하면 명시적인 activation uncertainty error를 반환한다.
5. 테스트용으로 전체 filesystem adapter를 만들지 말고 sync fault-injection seam만 둔다.

권장 형태:

```go
func activateCurrent(root, version string) error {
    return activateCurrentWithSync(root, version, syncDirectory)
}

func activateCurrentWithSync(
    root string,
    version string,
    syncFn func(string) error,
) error {
    ...
}
```

### 테스트
- 기존 `current -> versions/v1.2.3`, 새 target `v1.2.4`, 첫 sync 실패/rollback sync 성공 → error + `current`는 v1.2.3.
- 최초 설치에서 첫 sync 실패 → error + `current` 없음.
- rollback sync도 실패 → 성공 금지 + activation uncertainty error.

### 완료 기준
- error를 반환하면서 새 version이 정상 active 상태로 남는 경로가 없어야 한다.
- `go test ./internal/releaseinstall/...` 통과.

---

## FIX-02. Release tag가 protected `main` 및 exact SHA의 Required CI를 통과했는지 재검증

**중요도: High — release 전 수정**

### 위치
- `.github/workflows/heliopause-release-publish.yml`
- 필요 시:
  - `scripts/check/workflow.go`
  - `scripts/check/workflow_test.go`
  - `scripts/check/release.go`
  - `scripts/check/release_test.go`

### 문제
현재는 `release tag SHA == successful Release Build head SHA`는 확인하지만, 그 SHA가 실제 protected `main` history에 속하고 exact SHA의 `Required` check가 성공했는지는 publish workflow가 독립적으로 재검증하지 않는다.

### 수정

tag SHA 계산 직후 다음 두 검증을 추가한다.

#### 1. tag SHA가 `main` history에 속하는지

```bash
main_sha="$(gh api "repos/$GH_REPO/branches/main" --jq '.commit.sha')"

compare_status="$(
  gh api "repos/$GH_REPO/compare/$tag_sha...$main_sha" --jq '.status'
)"

case "$compare_status" in
  ahead|identical) ;;
  *)
    echo "release tag commit is not in protected main history" >&2
    exit 1
    ;;
esac
```

허용 의미:

```text
tag SHA == main SHA
또는
tag SHA가 main의 ancestor
```

#### 2. exact tag SHA의 `Required` check 성공 확인

```bash
required_count="$(
  gh api     -H "Accept: application/vnd.github+json"     "repos/$GH_REPO/commits/$tag_sha/check-runs?per_page=100"     --jq '[.check_runs[]
      | select(
          .name == "Required"
          and .status == "completed"
          and .conclusion == "success"
          and .app.slug == "github-actions"
        )
      ] | length'
)"

test "$required_count" -ge 1
```

### 테스트
workflow invariant checker에 최소 다음 negative case를 추가한다.
- `main` ancestry 검증 제거 → reject.
- exact SHA `Required` success 검증 제거 → reject.

### 완료 기준

```text
valid immutable tag
AND tag SHA == successful Release Build SHA
AND tag SHA ∈ protected main history
AND Required(tag SHA) == success
```

---

## FIX-03. Release candidate 10개 파일 전체 attestation 검증

**중요도: Medium-High — release 전 수정**

### 위치
- `.github/workflows/heliopause-release-publish.yml`
- 필요 시 `scripts/check/workflow.go`, `scripts/check/workflow_test.go`

### 문제
Build workflow는 candidate directory 전체에 attestation을 만들지만 publish workflow는 `manifest.assets[]` 중심으로 검증해 metadata 일부가 동일한 검증 loop에 포함되지 않는다.

### 검증 대상 exact 10개

```text
haa_gvisor_observer-linux-amd64
haa-network-policy-helper-linux-amd64
helox-darwin-amd64
helox-darwin-arm64
helox-linux-amd64
helox-linux-arm64
helox-runtime-images.json
helox-release-manifest.json
helox-release-sbom.cdx.json
helox-release-checksums.txt
```

### 수정
`Verify GitHub artifact provenance` 단계에서 위 10개 exact set을 순회한다.

```bash
expected_files=(
  haa_gvisor_observer-linux-amd64
  haa-network-policy-helper-linux-amd64
  helox-darwin-amd64
  helox-darwin-arm64
  helox-linux-amd64
  helox-linux-arm64
  helox-runtime-images.json
  helox-release-manifest.json
  helox-release-sbom.cdx.json
  helox-release-checksums.txt
)

signer="$GH_REPO/.github/workflows/heliopause-release-build.yml"
source_sha="$(jq -er '.source_commit' "$candidate/helox-release-manifest.json")"

for name in "${expected_files[@]}"; do
  test -f "$candidate/$name"

  gh attestation verify "$candidate/$name"     -R "$GH_REPO"     --signer-workflow "$signer"     --source-digest "$source_sha"     --predicate-type 'https://slsa.dev/provenance/v1'     --deny-self-hosted-runners
done
```

candidate top-level file set도 단순 `count == 10`이 아니라 **exact name set equality**로 검증한다.

### 완료 기준
- 10개 전부 attestation 검증 성공해야 publish 진행.
- 1개라도 누락/추가/attestation 실패하면 fail closed.
- manifest/SBOM/runtime-images/checksums도 binary/helper와 동일한 provenance 조건을 만족.

---

## FIX-04. Draft asset을 public 전 name/size/SHA-256으로 재검증하고 post-publish 실패는 quarantine 처리

**중요도: High — release 전 수정**

### 위치
- `.github/workflows/heliopause-release-publish.yml`
- M12 release operation/runbook
- 필요 시 `scripts/check/workflow.go`, `scripts/check/workflow_test.go`

### 문제
현재 핵심 흐름은:

```text
draft 생성
-> publish
-> gh release verify
-> gh release verify-asset
```

Immutable Release는 publish 후 asset/tag 변경이 제한되므로 public 전 마지막 검증을 draft 상태에서 끝내야 한다.

GitHub Release asset API의 `digest`(`sha256:...`)를 local candidate와 대조한다.

### 수정

draft 생성 직후 `Verify draft release asset bindings before publication` 단계를 추가한다.

검증:

```text
draft == true
tag_name == RELEASE_TAG
asset count == 10
asset name set == expected 10
각 asset.state == uploaded
각 asset.size == local file size
각 asset.digest == sha256:<local SHA-256>
```

예:

```bash
release_json="$(gh api "repos/$GH_REPO/releases/tags/$RELEASE_TAG")"

test "$(jq -r '.draft' <<<"$release_json")" = "true"
test "$(jq -r '.tag_name' <<<"$release_json")" = "$RELEASE_TAG"
test "$(jq '.assets | length' <<<"$release_json")" -eq 10
```

각 파일:

```bash
expected_size="$(stat -c '%s' "$candidate/$name")"
expected_digest="sha256:$(sha256sum "$candidate/$name" | awk '{print $1}')"

asset_json="$(
  jq -c --arg name "$name"     '.assets[] | select(.name == $name)'     <<<"$release_json"
)"

test -n "$asset_json"
test "$(jq -r '.state' <<<"$asset_json")" = "uploaded"
test "$(jq -r '.size' <<<"$asset_json")" = "$expected_size"
test "$(jq -r '.digest' <<<"$asset_json")" = "$expected_digest"
```

모두 성공한 뒤에만:

```bash
gh release edit "$RELEASE_TAG" --draft=false
```

를 실행한다.

### post-publish 처리
공개 후 `gh release verify`, `gh release verify-asset`은 그대로 유지한다.

단, 공개 후 검증 실패는 overwrite/delete/re-tag로 복구하지 않는다.

상태를:

```text
PUBLISHED_BUT_QUARANTINED
```

로 취급한다.

운영 절차:
1. 동일 tag/asset을 수정해 살리지 않는다.
2. 해당 version 사용 중단을 공지한다.
3. 원인을 조사한다.
4. 수정 후 **새 SemVer version**으로 release한다.

### 테스트
workflow checker가 최소 다음 순서를 강제한다.

```text
draft create
< draft exact asset verification
< draft=false publish
< post-publish release verification
```

Negative:
- draft digest verify 삭제 → reject.
- digest verify가 publish 뒤로 이동 → reject.
- exact asset name/count 검증 삭제 → reject.

### 완료 기준
public transition 직전 다음을 모두 확인한다.

```text
exact source
exact candidate
exact provenance
exact 10-file set
exact size
exact SHA-256
```

---

## FIX-05. Observer profile wait가 첫 50ms timeout에서 실패할 수 있는 bug

**중요도: Low-Medium — 보안 fail-open 아님, release 전 같이 수정 권장**

### 위치
- `tools/gvisor-observer/observer.cc`
- 함수: `AwaitProfile(...)`
- 테스트: `tools/gvisor-observer/observer_latch_test.cc`

### 문제
현재:

```cpp
const int result = poll(&descriptor, 1, 50);
if (result <= 0) return nullptr;
```

`poll()`의 정상 timeout은 `0`이다. 따라서 `kProfileRegistrationWaitMilliseconds = 2000`이어도 첫 50ms timeout에서 즉시 실패할 수 있다.

### 수정

```cpp
const int result = poll(&descriptor, 1, 50);

if (result < 0) {
  if (errno == EINTR) continue;
  return nullptr;
}

if (result == 0) {
  continue;
}

if ((descriptor.revents & POLLIN) == 0) {
  return nullptr;
}
```

다음 loop 상단의 `DrainProfiles()`가 profile을 다시 확인하게 한다.

의미:

```text
50ms timeout = 계속 기다림
EINTR        = 계속 기다림
실제 poll 오류 = 실패
POLLIN       = 다음 loop에서 profile drain
최대 약 2초  = 계약대로 대기
```

### regression test
`observer_latch_test.cc`에 delayed registration case를 추가한다.

1. remote connection/handshake.
2. container start event로 container ID latch.
3. profile 등록을 50ms보다 늦게, 예: 150~250ms 후 수행.
4. 2초 이내 profile이 도착하면 정상 classified record가 나오는지 확인.

기대:

```text
150~250ms delay -> success
configured wait window 초과 -> fail closed / stream-fault
```

### 완료 기준
- 50ms idle만으로 observer가 실패하지 않음.
- observer helper/latch CI 통과.
- gVisor Linux Integration 통과.

---

# 최종 실행 순서

```text
FIX-01 release activation rollback/durability
FIX-02 protected main + Required provenance
FIX-03 candidate 10-file attestation
FIX-04 pre-publish draft asset digest verification
FIX-05 observer profile wait bug
```

그 후:

```bash
go run ./scripts/check quick
go run ./scripts/check security
go run ./scripts/check vulnerability
go run ./scripts/check release-gate
```

그리고 GitHub Required CI와 gVisor integration을 전부 통과시킨다.

---

# Release Gate

- [ ] FIX-01 activation error 시 이전 active version 보존
- [ ] FIX-02 release tag SHA가 protected main provenance 및 Required success를 증명
- [ ] FIX-03 candidate 10개 모두 GitHub attestation 검증
- [ ] FIX-04 draft 10개 asset name/size/SHA-256 검증 후에만 public 전환
- [ ] FIX-04 post-publish 실패 시 overwrite가 아니라 quarantine 절차 적용
- [ ] FIX-05 observer 50ms timeout bug regression 통과
- [ ] canonical Quick/Security/Vulnerability/Release Gate 성공
- [ ] GitHub Required 성공
- [ ] gVisor Observer Helper 성공
- [ ] gVisor Linux Integration 성공

이 체크리스트 이후 새 발견이 **fail-open, trust-boundary bypass, data-loss/rollback violation, release provenance break** 중 하나가 아니라면 첫 release를 다시 연기하지 않는다.
