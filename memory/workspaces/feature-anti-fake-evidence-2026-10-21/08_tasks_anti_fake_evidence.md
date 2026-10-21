# 08 — Tasks `anti_fake_evidence`

- [x] T1: `tooling/evidence_run.py` — chạy lệnh thật, ghi `06_evidence/EV-*.json` (cmd, cwd, kind, exit, stdout_tail, sha256, git HEAD, diff_sha256).
- [x] T2: Claim Linter (H4c pre-write `06_validation|11_report|audit_report`): mọi dòng khẳng định PASS/OK/KILLED/xanh/chạy thật/`n/n` phải trích `EV-…` hợp lệ.
- [x] T3: Kiểm EV: tồn tại; có `run_command` tương ứng trong transcript; `diff_sha256` == diff hiện tại (không cũ); số trong claim có trong stdout; claim "runtime/chạy thật" phải là kind `runtime|integration` và lệnh không phải test runner.
- [x] T4: `mutation_gate.py --base <ref>`: mỗi mutant phải nằm trong dòng đã đổi so với base; mỗi file source đổi phải có ≥1 mutant.
- [x] T5: `verify_governance.py` thay kiểm từ khoá audit bằng Claim Linter dùng chung; H7 chặn Done khi còn claim không EV.
- [x] T6: Test Red→Green + mutation gate cho chính T1–T5; runtime: replay transcript phiên 05bada14 (các claim cũ phải bị bắt).

DoD: replay báo cáo cũ (audit harness2 AF-07, audit_report rea_cleanroom_gap) → linter chỉ ra đúng các claim không có EV; claim có EV hợp lệ qua.
- [x] T7: fingerprint theo cwd của từng biên lai.
