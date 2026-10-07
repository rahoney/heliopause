# M12-02 여덟 영역 검토

기준은 `19411ca`와 FIX-01/02/03 correction이다. 작성자의 repository 검토이며 외부
독립 감사·GitHub reviewer 승인을 뜻하지 않는다. canonical 문서의 기존 invariant와
M12-005의 77개 요구사항 매핑을 기준으로 실제 owner·호출부·거부 조건을 확인했다.
원격 실제 소비자 결과는 완료 후 이 evidence의 result에 별도로 연결한다.

| 영역 | 확인한 책임 경로와 실패 조건 | 검토 판정 및 검증 |
| --- | --- | --- |
| Source/registry identity | `artifact/gomodule/client.go`, `artifact/cargo/client.go`, `artifact/pypi/source_profile.go`, `artifact/npm/resolver.go`, `artifact/githubrelease/client.go`, `artifact/terraformprovider/client.go`: 고정 public source, source-owned identity, frozen acquisition locator, proxy/credential/redirect 및 alternate registry 거부 | 새 release blocker 미발견. PyTorch source ownership·local version/hash negatives, Cargo registry/lock source negatives, GitHub numeric asset ID와 declared size/digest binding, TF selected platform/publisher를 기존 테스트로 확인 |
| Exact graph freeze | `artifact/gomodule/graph.go`, `artifact/cargo/locked_metadata.go`, npm lockfile v3, PyPI frozen report/Simple cross-check, TF complete project snapshot; `domain/project_verified_set.go`의 exact count/identity/independent Run coverage | 새 release blocker 미발견. 누락·중복·잘못된 edge/checksum/source와 declared workspace 밖의 selection 거부. Dependency-free project와 independent TF provider roots는 기존 계약대로 표현 |
| Integrity | `verification/gomodule/integrity.go`의 pinned SumDB, `verification/cargo/integrity.go`의 독립 index+lock+bytes, npm SRI, PyPI independently declared SHA, GitHub API-declared SHA, `verification/terraformprovider/signature.go`의 signed checksums/full fingerprint/current authority | 새 release blocker 미발견. TF tampering/ambiguity/community-key/historical-current-authority/revoked-expired negatives 포함. GitHub declared SHA를 별도 publisher signature나 package safety 인증으로 표시하지 않음 |
| Observation/attribution | `sandbox/direct_exec_admission_test.go`와 실제 owner의 fresh cryptographic admission/ACK/session invalidation; Python transaction ledger/accounting/cleanup; `inspection/pypi/command_observation.go`, `inspection/terraformprovider/inspection.go` | 새 release blocker 미발견. 출력 token·help exit0·stop 시도를 완료 authority로 사용하지 않음. Required incomplete는 승격하지 않으며 TF suspicious facts와 coverage를 구분. Viewer NOT_ATTESTED와 functionality/later-enforcement false는 유지. Current actual runtime 및102 latch 결과는 원격에서 재검증 |
| Policy/Evidence binding | `application/project_inspect.go`, `project_update.go`, `project_build.go`, `cargo_build.go`; Domain `NewInspectedProjectSet`/`NewProjectVerifiedSet`; `promotion/project_cache_evidence.go`, `evidence/local/read.go`, Policy M3/M4 | 새 release blocker 미발견. Marker만으로 승인하지 않고 실제 record의 run/check/subject/digest를 재검사. Complete independent ALLOW가 없는 cache/add/build/init는 거부. Default tests에서 missing/foreign/tampered Evidence·approval 및 exact coverage negatives 확인 |
| Transaction/rollback | 기존 npm/PyPI·Go/Cargo/TF transaction과 GitHub no-replace new-target publish; identity/content/mode/single-link, competing destination, backup cleanup, first-cause preservation, recovery boundary | **FIX-01/02 확정.** Go/Cargo4개 최초 삭제와 npm/PyPI6개 최초 삭제 사례를 실제 owner에서 재현. Go/Cargo40개 및 legacy48개 정상/부정 사례와 기존 rollback 회귀 PASS. 첫 FIX-01의 안전한 복원 차단은 반증·보완. Current remote project/add/init/full 결과 대기 |
| Cross-ecosystem regression | 현재 correction의 canonical quick/docs/security/vulnerability/fuzz/freshness/release-gate, 기존 public/runtime CLI/inspection/promotion Required 소비자 | **FIX-03 확정.** 새 Required npm 프로젝트 소비자가 기존 preparation의 EEXIST를 actual CI에서 검출. 동일 기준선 fixture FAIL→빈 selected child를 쓰는 수정 후 composed first/retained PASS. Local canonical actual0는 실제 offline 성공을 대신하지 않으며 새 CI에서 판정 |
| Release integration impact | `releaseinstall/installer.go`의 verifier-before-write/manifest/assets/runtime binding, release build/publish workflows의 tag/exact run/main ancestry/Required/attestation/draft asset verification | 새 release blocker 미발견. 기존 binary/observer/helper 및 lock/Policy/resource/support 계약을 유지하는 bounded transaction correction. Canonical release-gate PASS는 실제 publish 증명이 아님. Tag 생성·publish·main merge는 미실행이며 M13 운영 acceptance를 대체하지 않음 |

## 변경 및 재사용 경계

FIX-01은 Go/Cargo control/metadata 보관·복원 owner를 수정한다. 원래 승인 guard,
resolver, graph, cache Evidence, build observer 및 자원 한도를 재사용한다. FIX-02는
npm 기존 member와 PyPI 기존 destination/append-only journal을 기록한 신원·내용과
대조하고 확인한 파일·빈 directory만 삭제한다. 임의 tree 삭제와 overwrite rename을
제거하고 primary cause와 불확실한 백업을 보존한다. 새 Core/daemon/framework,
runtime 역할, Policy 예외, dependency 또는 미래 milestone scaffold를 추가하지 않았다.

npm의 member snapshot은 기존 project transaction set의 정확한 파일/디렉터리에
국한된다. PyPI journal digest는 append 시 controller bytes로 누적하며 정리 전 길이와
inode/mode/single-link 및 bounded digest를 재검사한다. Artifact가 작성한 journal이나
metadata를 새 승인 authority로 사용하지 않는다.

PyPI correction은 네 PyTorch profile의 공유 venv commit에 영향을 주므로 기준005의
full을 correction의 새 full로 전용하지 않는다. 같은 후보의 CPU 및 cu126/cu130/cu132를
다시 실행한다. Local 실제 npm consumer는 registered runsc bundle identity mismatch로
preflight에서 FAIL했고 body는 미실행이다. 고정 runtime을 새로 설치하는 CI가 실제
project 첫/retained 소비자와 나머지 생태계의 정상 동작을 판정한다.

지원 범위는 [M12 계약](../../16-m12-01-ecosystem-expansion-contract.md)의 기존 네 tuple,
명시적 resource bounds와 설치·관찰 scope를 유지한다. GPU 계산/driver/Toolkit, 다른
ABI/OS/torch version, torchvision/torchaudio, TF RPC/cloud 및 일반 Host crash recovery
인증을 이번 red-team 결과로 새로 허용하지 않는다.

FIX-03은 frozen original controls의 private 복사본과 selected output을 같은 디렉터리에
생성하던 기존 준비 충돌을 수정한다. 원본 복사본은 private parent에 보존하고 selected
controls는 빈 child에 생성한다. No-replace/host controls 보호와 기존 검증·transaction을
재사용하며 overwrite 허용이나 fixture/Required skip은 추가하지 않는다.

Status: COMPLETE — FIX-01/02/03 remote qualification and acceptance closed.

위 로컬 checkpoint의 대기 상태는 당시 범위다. 현재 [최종 결과](./result.json)는
동일 source68099a5의 ordinary/cu126/cu130/cu132 각10SUCCESS, 실제 npm project
first/retained2회, CPU/세CUDA full 및 기존 모든 Required 소비자 결과를 연결한다.
Source/checkout whole tree/mode,102latches·5helperwait0/final cleanup을 각각 확인했다.
