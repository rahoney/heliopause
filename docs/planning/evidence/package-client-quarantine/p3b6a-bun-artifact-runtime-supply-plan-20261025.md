# P3-B6a Bun Artifact runtime의 격리 공급·resource owner 감사

M13-004 하나만 IN_PROGRESS다. Baseline 386777d5235411e9f892ce2de8add78717a2861f/treecc4dcd9f36614f32f1b184c32039efec39926d79/2465files.
Read: client 설계§4/명시 runtime identity, 확장 계획§3/§5, B4d lifecycle 의미·B5a..f 영수증,
BunResolverRuntimeIdentity, native_project_scope/tool transport/runtime volume, Backend와
pnpm_instance_backend, bun_instance_backend/target lifecycle, SharedObserver topology와
CPP IsPrivateNativeToolImage/HasExactNPMTargetLifecycleMount, 관련 네 주제 lessons.

| 책임 | IMPLEMENTED | WIRED | QUALIFIED | ACCEPTANCE_CLOSED | MISSING |
| --- | --- | --- | --- | --- | --- |
| protected installed Bun bytes·fixed guest tool·root-owned native volume·whole readback | YES | YES(query/selection/preparation) | YES(local/CI412) | YES(existing scope) | Artifact lifecycle의 별도 owning session에 재사용 |
| Node/npm per-instance Bun backend·whole bound payload/observation/cleanup | YES | YES | YES(local/CI412+B5e actual) | YES(Node lane) | 기존 경로를 유지하며 native runtime 공급 경로 추가 |
| Bun Artifact runtime session의 fixed supply/attested9p helper topology | NO | NO | NO | NO | 아래 transport와 no-network/cleanup 직접 검증 |
| 명시 Bun Artifact 실행·필수 coverage/Policy/whole acceptance | NO | NO | NO | NO | transport 뒤 actual 실행/admission·기존 inspection 연결 |

현재 Bun backend는 npm/Node image와 고정 target command를 사용하고 /haa-runtime에는
boundary helper만 있다. Bun native resolver의 공급을 검증했어도 Artifact session에 Bun
binary가 존재하거나 실행됐다는 증거가 아니다. 이미 runtime fingerprint에는 Bun version/
full tool SHA/Node image/runtime lock SHA가 결합돼 있으므로 새 pin/schema를 만들지 않는다.

## Exact edit map·두 번째 검토

1. `pnpm_instance_backend.go`: 기존 owning body를 private helper로 좁게 재사용한다. 기존
   호출은 nil native-runtime option으로 동일 경로를 유지한다. Bun-only option에서만 기존
   prepareBunTool/nativeProjectTool·createNativeRuntimeVolume을 사용한다. capability/tool kind
   preflight→native volume→project/container→sealed observer/helper→exact attachment→protected
   bytes copy/full readback→attachment 재검증→bound project copy→network none→lifecycle 순서다.
   종료 시 container/trace/project/native volume을 각각 기존 owner로 닫고 primary cause를 보존한다.
2. `bun_artifact_runtime.go`: 기존 `bun_instance_backend.go`/ExecuteBunInstance는 bytes를
   유지한다. 새 typed
   ExecuteBunRuntimeInstance는 동일 opaque layout/acquired subject/instance preflight와 payload를
   재사용하고 별도 observer method/runtime factory에만 연결한다. 임의 tool/path/client 옵션을
   공개하지 않는다. 이 단계는 runtime 공급 검증이며 전체 Bun 실행 지원을 선언하지 않는다.
3. `bun_artifact_runtime.go`: 기존 create recipe의 helper tmpfs 한 개만 exact native volume
   mount로 바꾼다. Root ownership/modes/full SHA/size/readback/attachments 계약은 그대로다.
   기존 Node target command를 유지하며 별도9p helper topology만 고정한다. PATH/명시 Bun
   실행 command는 exact observer admission과 함께 B6b에서 확인한다.
   기존 Node command/topology·PNPM·resolver grants는 유지한다. `shared_observer.go`에 이
   정확한 topology만 사용하는 StartBunRuntimeTargetLifecycleProfile을 연결한다.
4. 새 unit/realcorpus 소비자는 기존 Docker format fixture를 좁은 decorator로 재사용해
   legacy path 불변·fresh 두 volume/session·wrong kind/tool/mount/readback/cleanup·first cause를
   검사한다. Actual에서는 pinned supply가 도입된 별도 session의 전체 source-bound payload/
   no-network/cleanup을 확인한다. Bun Artifact script 자체의 실행/admission과 full inspection
   연결은 이어지는 B6b의 별도 gap이며 이 transport 성공으로 허위 완료하지 않는다.

CPP permission/role/network·runtime pin·예산/Policy를 이번 transport에서 바꾸지 않는다.
Canonical format/Quick/security/docs·sandbox 전체 Race와 실제 namespace 소비자, 전체
index/blob/mode/disk/tree를 확인한다. 작은 helper 단계마다 원격 CI를 시작하지 않는다.

## 공급 경로 로컬 결과

[로컬 영수증](./p3b6a-bun-artifact-runtime-supply-local-20261025.json)은 public peer source4/
exact instance4를 각각 fresh session/native root-owned volume에서42.09s에 확인했다. Protected
107,308,936B tool을 매번 전체 SHA/readback/attachment로 검증하고 기존 Node 명령을 관찰했다.
기존 PNPM transitive2/peer4 full project48.69s도 PASS이며 operator93.696s/helperwait0/Host
전체 동일·owned volume/container 정리를 확인했다. Quick181.662s/전체 sandbox Race24.570s/
tagged vet1.273s·unit/보안/문서 검사가 PASS다. Nil-runtime owner는 추가된 Bun-only guard를
제거하면 원래 body와 bytes가 같고 기존 Bun/Node/CPP·tool transport/volume 파일은 불변이다.
이 결과는 native tool 공급만 닫으며 명시 Bun Artifact 실행/필수 coverage·Policy·hosted는
B6b와 최종 기능 묶음 검증으로 이어간다.
