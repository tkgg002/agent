# 08 — Tasks `rea_cleanroom_gap`

- [x] T1: S2 đọc enforcement clean-room thật (`hooks/agy/agy_guard.py:47-56,268-350,532-594,754-760`, `tooling/verify_governance.py:197-255`).
- [x] T2: Tái hiện runtime lỗ hổng G-01 (APPROVE false-positive) & G-02 (bypass đọc code qua `run_command`).
- [x] T3: Kiểm tra harness2 integration (`tooling/orchestrator/harness2_engine.py`) có được wire vào GEMINI.md/skills/hooks không.
- [x] T4: Viết `13_analysis_rea_cleanroom_gap.md` + 1 phương án khắc phục.
- [x] T5: DG3 `laya_gate.py run --stage dg3` → PASS (claims UNVALIDATED, kết luận bằng code + runtime).
- [x] T6: User APPROVE (2026-10-21T09:46).
- [x] T7 (G-08): lesson `ID/Trigger/Check` → `tripwires.json` → H1 inject `Check` khớp prompt; backfill lesson README-only. Test Red→Green.
- [x] T8 (G-01): `CLEAN_ROOM_START` neo nguyên dòng cho task `mode: clean-room`; APPROVE thường neo đầu dòng.
- [x] T9 (G-02/G-04): `contaminated_clean_room_reads` xét `run_command`, `read_url_content`, `invoke_subagent`.
- [x] T10 (G-03): Spec `R<n>: ... — REF:` bắt buộc trong H4 + `verify_governance.py` cho clean-room.
- [x] T11 (G-05): `run-recursion` bỏ proposer/scorer giả viết cứng; bắt buộc `--proposer-cmd`/`--rollout-cmd` thật, thiếu → exit 2.
- [ ] T12: unittest hooks + tooling PASS; `mutation_gate.py` ≥3 mutant KILLED; DG3; audit_report; verify_governance S6.
- [x] T13 (AF-1): tripwire evidence chỉ từ tool đọc/lệnh, không tính tool ghi.
- [x] T14 (AF-2): G-04 resolve path cấm tương đối theo Cwd của run_command.

DoD: Mọi finding có `file:line` hoặc output runtime; DG3 PASS/UNVALIDATED được ghi rõ.
