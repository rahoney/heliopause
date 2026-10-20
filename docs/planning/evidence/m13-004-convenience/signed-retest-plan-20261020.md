# M13-004 — 보완 소스의 새 서명 시험 입력 제안

이 문서는 다음 시험을 구체화하며 새 태그 생성 승인을 대신하지 않는다. 기존 승인의
rc1/rc2 source는 a22aa12다. 두 보호 태그를 재지정하거나 immutable rule을 변경하지 않는다.

| 입력 | 제안 |
| --- | --- |
| 고정 source | 4ff6c89234698d6fe0b6b5366f7605538e47b967 |
| 실제 CI | [CI356/run39416778306](https://github.com/rahoney/heliopause/actions/runs/39416778306), Required 포함 열 개 실제 SUCCESS 확인; 서명 입력 실행은 owner 승인 뒤 |
| 새 base tag | v0.1.1-rc.3 |
| 새 target tag | v0.1.1-rc.4 |
| 상태 | OWNER_APPROVAL_REQUIRED; 아직 생성하지 않음 |
| 유지 범위 | Linux/WSL2 amd64, 기존 runtime lock·SDK·관찰·Policy·좁은 Root 역할 |

[보완 검증](./signed-native-qualification-20261020.md)의 최종103 latch/native4/기존 npm과
부정 경계를 재사용한다. Actual CI checkout c0a0dddd617d2694b0f4b02b86e1f2def0a641cb의
전체 tree/mode가 위 source와 같은지 확인했다. 이 문서 이후의 docs commit을 태그 source로
묵시 변경하지 않는다. 새 기능·새 runtime·외부 채널 이름 결정은 포함하지 않는다.

기존 rc1은 보완 전 observer 때문에 새 native를 완전하게 검사하지 못했다. 고정 source의
새 base/target 두 버전은 보완된 이전 controller가 다음 버전의 전체 세트를 검역하는지
검증하기 위해 필요하다. 기존 태그의 재실행으로 다른 source를 서명하거나 서명/검사 면제,
unsigned helper의 managed 설치 덮어쓰기로 대체하지 않는다.

승인 뒤 순서:

1. 실제 CI·source·ref 부재·기존 creation/immutable rules를 재확인한다. Base tag 하나를
   생성해 official tag-push build/attestation의 source/tag/run과 exact13파일을 확인한다.
2. 독립 verifier와 production SDK를 사용해 새 깨끗한 전용 환경에서 first files→고정
   runtime→실제 service/register→ordinary doctor/nominal을 확인한다. 필요한 Root 사본의
   서명 검증보다 먼저 후보 helper를 Root로 실행하지 않는다.
3. Base 확인 뒤 같은 고정 source에 target tag를 생성한다. 두 버전의 실제 bytes/metadata
   차이를 기록하고 public self-update caller→required7세트 검사/새 Evidence/Policy→동일
   Root 보호 사본/전환→새 ordinary doctor/nominal을 확인한다.
4. 거부·취소·동시성·OS 관리자 인증 성공/실패, 실제 service 실패/복원, Root 중단과 durable
   journal 재개를 분리한다. Fixture transport는 actual public Release API 성공이 아니며
   guest service/PID1 restart는 Host kernel/WSL reboot의 증거가 아니다.
5. 성공·실패의 원본과 작은 public proof/manifest만 남기고 완료된 전용 환경·payload·cache를
   삭제한다. 작은 보호 태그의 삭제는 이후 해당 ref에 한정한 보호 규칙 조정이 필요하다.

Public Release·main merge·기존 Host 영구 migration·package publisher 연결은 실행하지 않는다.
기존 PR38을 이어 사용하며 M13-004만 IN_PROGRESS다. 두 번째 대조에서 이전 실패의 서명을
무효화하거나 보호 규칙을 약화하지 않고 정확한 새 source를 검증하는 순서임을 확인했다.
