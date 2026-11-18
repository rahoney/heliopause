# R4 선별 결과 D5 독립 검토

읽기 전용 검토자 `/root/d3_entry_review`는 원 F6+144/148 선별 tree71df0e3c의91경로 해시와 기존 연결을 확인했다. U1/S04 혼입·중복 ELF 시험·새 framework는 발견하지 못했다. R3의 엄격한 supervisor/helper/bootstrap과 gVisor patch/lock은 유지됐다. 이는 실제 통합 qualification이 아니다.

원148의 Python unit에서 후손 대기가 실패한 뒤 종료까지 실패하면 최초 drain 오류가 사라지는 P2가 있었다. 현재5a170fee는 기존 `pythonCleanupFailure`에 `unit-descendants`와 `unit-termination`을 함께 전달한다. 같은 INCOMPLETE·기존 오류 redaction·한도를 보존한다. 기존 diagnostic/종료/observer-control owner를 잇는 표적은 통과했다.

추가 subcase는 실제 UNKNOWN 응답·drain 함수와 종료 fake·오류 결합 owner를 시험한다. **수정된 전체 pipeline 반환 분기는 직접 실행하지 않으므로 원 caller의 회귀 검출이나 전체 Python 실물 검증으로 기록하지 않는다.** 호출부 연결은 별도 소스 검토로 확인했다. [정확한 신원·검토 범위](./r4-independent-source-review-20261118.json)를 따른다.

D3 U1 자동 회수 계약은 기존 REVISE/미승인 상태다. R4 전체·실제 정상/조건부/후손 정리·큰 profile·최종R8는 OPEN이다.
