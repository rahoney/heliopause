# M13-004 — 교정된 기존 caller의 최소 새 서명 시험 입력

상태: **SOURCE_QUALIFIED / OWNER_APPROVAL_REQUIRED**. rc5 단일 태그 승인은
소비했고, 이 문서는 새 tag 생성 승인이 아니다. 기존 immutable ref/rules는 유지한다.

| 입력 | 고정값 |
| --- | --- |
| source | 4a8c2787bdc2cb2d57e15b853d83dcee31b2ce25 |
| source qualification | [CI358](https://github.com/rahoney/heliopause/actions/runs/39461816815), 실제 Required+9 SUCCESS·checkout 전체 tree/mode 일치 확인 완료 |
| 제안 새 ref | v0.1.1-rc.6 하나, 아직 생성하지 않음 |
| 기존 기준 | qualified signed rc3; 원본 image/volume만 idle 보존 |
| 최초 전환 | rc3 → rc6의 실제 signed 전체 검역/정상 COMMITTED/새 doctor·npm |
| 취소 소비자 | 설치된 rc6가 동일한 actual signed rc6 exact set을 새로 획득·검역하는 실행 |

## 왜 기존 rc3에서 target만 바꾸면 부족한가

[보완 계획·두 번째 검토](./observer-cancellation-plan-20261021.md)의 결함은
**취소를 받는 기존 controller**의 long-lived helper launch에 있었다. rc3에서 rc6를
검사하다 취소하면 여전히 옛 실행기를 사용한다. 따라서 먼저 실제 signed rc6로 정상
교체해야 한다. 그 다음 취소 경계의 bytes·프로세스·config가 새 source인지 확인한다.

두 번째 새 버전을 만들 필요는 없다. 기존 `BeginInstallUpdate`/release-set acquisition은
동일 버전도 fresh inspection을 수행하고 `wireRelease.verify`/journal은 **동일 버전일 때
동일 manifest digest**만 허용한다. 이 기존 경로를 그대로 소비한다. 가짜 버전·proof·
artifact 이름을 만들거나 provenance/dynamic/Policy를 면제하지 않는다. 동일 버전의
검역 재실행은 rc6→미래 rc7 업그레이드의 검증이라고 보고하지 않는다.

## 실행 순서와 acceptance

1. Source4a8c278의 최종 로컬 후보·CI358 실제 열 개 성공·checkout 전체 tree/mode를
   확인한다. 확대 Sandbox Race7건은 변경 전 baseline에서도 확인된 기존 한계이며
   관련 owner Race·Quick 성공을 full Race PASS로 바꾸지 않는다. 이 정확한 source와
   새 ref 하나에 대한 owner 승인 후 부재/rules/source를 재확인해 official signing
   build를 실행한다. 문서 전용 후속 commit을 tag source로 바꾸지 않는다.
2. Official exact13의 고정 독립 verifier/기존 root·UID1000 offline 서명과 canonical
   bundle을 확인한다. Proof는 exact payload13 바깥에 둔다. Qualified rc3 환경을
   순차 시작해 private controller 준비·service·ordinary doctor/npm을 확인한다.
3. Guest-only HTTPS 전송에 actual rc6 bytes/proof를 제공한다. rc3 ordinary
   self-update의 required18/ALLOW→보호 사본 동일성→COMMITTED/새 doctor5/npm3를
   확인한다. 이전 rc5 정상/OS 인증/서비스·Root 중단·cold recovery 증거는 보존한다.
4. **실제 현재 rc6 controller**의 exe/inode·manifest·등록·서비스 identity를 확인한다.
   동일 rc6 set을 향하는 ordinary self-update에서 다음 두 경계를 구분해 시험한다.
   - observer readiness 뒤 실제 asset 획득 중 SIGINT: 확인된 UID1000 CLI PID/exe/
     starttime과 pidfd에만 전달한다. 현재 버전/등록/history digest 불변, 새 Root
     transaction 부재, 확인된 helper 종료와 endpoint/lock 정리, 다음 doctor/npm을 확인한다.
   - actual native dynamic 검사 중 취소: test-owned container/observer와 실제 caller
     신원을 먼저 고정한다. required 완료로 위장하지 않고 원래 취소/실패를 보존하며
     helper·container·slice의 확인된 drain과 다음 검사 가능 여부를 확인한다.
   취소 결과를 성공한 업데이트로 보고하지 않는다. Endpoint identity/정지가 불명확하면
   강제 unlink·dependent restore를 하지 않는다.
5. 같은 rc6 exact signed set의 새 whole 재실행이 required18/ALLOW와 COMMITTED에
   이르는지 확인한다. Fixture sudo 권한은 필요하면 정확한 installed serve 한 경로에만
   임시 부여하거나 실제 OS 인증을 사용한다. 이 시험 권한은 제품 sudoers 계약이 아니다.
6. 작은 public manifest/proof·원본 Evidence를 보존하고 마지막 소비 뒤 download13,
   guest/TLS/암호·owned image/volume·build/cache/operator를 정리한다. 더 필요한
   acceptance가 없으면 idle 보존 기준도 최종 정리한다. Host 설치·공유 image는 보존한다.

Public Release·main merge·Host 영구 migration·채널 이름 결정·Host kernel/WSL reboot는
포함하지 않는다. 새 tag는 보호 규칙상 즉시 자유롭게 삭제할 수 없으며, 나중에 제거하려면
정확한 ref에 한정한 별도 ruleset 조정이 필요하다. 이 계획은 그 권한을 요청하지 않는다.

두 번째 대조: 새 ref 하나로 실제 version 증가(rc3→rc6)와 교정된 caller의 실제
취소·회복(동일 signed set fresh replay)을 연결한다. 기존 source/SDK/Root protocol/
quarantine owner를 재사용하고 새 구현·runtime·신뢰 예외·미래 placeholder는 추가하지 않는다.
원격 source qualification과 owner 승인 전에는 ref를 만들지 않는다.

CI358의 열 개 실제 SUCCESS와 모든 job의 checkoutdacc156, source4a8c278/tree6a81838
전체 일치를 [원격 receipt](./signed-native-20261020/observer-remote-ci.json)로 확인했다.
Native macOS/C++103/Linux45 실제 PASS·CPU291.99s/helper5 wait0/final confirmed
cleanup도 확인했다. 최종 읽기에서 rc6는 부재, rc5는7fedaf3, 두 rules는 active이며
PR38은 draft4a8c278다. 필요한 rc3 환경은 idle stop 상태다. 이 확인으로 새 tag
생성 승인을 대신하지 않는다.
