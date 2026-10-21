# Audit Report `anti_fake_evidence`

## 1. Từng dòng git diff
`git diff HEAD -- hooks tooling`: `agy_guard.py` (+CLAIM_RE/EV_RE, `claim_issues`, `current_fingerprint`, H4c), `mutation_gate.py` (`diff_anchor_issues`, `--base`, CRASHED, dòng MUTATION_GATE), `evidence_run.py` (mới), 2 file test mới.
- H4c chỉ lint dòng mới (trừ dòng có trong TargetContent) — không chặn tài liệu cũ.
- Khớp biên lai ↔ transcript đổi từ "nguyên văn cmd" sang "`evidence_run.py --kind <kind>` + không ghi tay 06_evidence" sau khi runtime cho false deny (biến shell/nháy).
- `CRASHED`: bổ sung khi thấy mutation_gate tính mọi exit≠0 là KILLED.

## 2. Chống mockup / smoke — trace runtime
Test dùng repo git thật trong tmp, file thật, subprocess thật. H4c chạy qua đúng lệnh hook với transcript thật (EV-20261021T103738-aa1a): deny/ask/deny đúng 3 ca.

## 3. Concurrency / timeout / continuity
`mutation_gate.py` sửa file tại chỗ: hook H6 chạy test song song đã báo fail giả 2 lần khi gate đang chạy (đã xác minh: chạy tuần tự sau đó đều xanh). Rủi ro cho phiên song song; chưa sửa trong scope này. Fingerprint không đổi khi commit, đổi khi sửa code.

## 4. Suy diễn / báo cáo láo
Mọi dòng khẳng định trong 06_validation trích biên lai EV; ca chưa kiểm được ghi riêng. Lần đầu H4c runtime cho false deny — đã sửa, không che.

## 5. Self-Improvement Loop
S5 → S3 → S4 hai lần (plan mục 4, 5). Lesson: biên lai chỉ giữ đuôi output nên verdict phải in ở dòng cuối (đã thêm `MUTATION_GATE ...`).

## 6. laya_router DG3
Xem `05_progress_anti_fake_evidence.md`. Vòng 2: `06_laya_summary_dg3_20261021T104018.json` gate PASS, không FLAG.

## Vòng 2 — H7 stop-audit (10:39)
- Diff critique tìm ra: `current_fingerprint` lấy Cwd của run_command cuối phiên ⇒ chạy lệnh ở repo khác là mọi biên lai bị báo "cũ" giả. Sửa: `_fp_for(rec["cwd"])` theo từng biên lai; test hồi quy trên repo git thật trong tmp; mutant M8 thay M2.
- Diff-anchor tự bắt lỗi của chính em: chạy `--base HEAD` sau khi đã commit thì mutant cũ không chạm dòng đổi → INVALID_INPUT (EV-20261021T103942-2f79, exit 3). Base đúng là commit trước task: 74840c8.
- Số liệu sau sửa: xem 06_validation mục Vòng 2; DG3 chốt: `06_laya_summary_dg3_20261021T104512.json`.

## Vòng 3 — Tối ưu Quy trình & Tiết kiệm Token A–D (11:18)
- (A) `GEMINI.md`: bỏ đọc `SKILL.md` cưỡng bức ở từng chặng S1–S6, giữ đủ trigger token cho `validate-surface`.
- (B) `tooling/laya_gate.py` + `hooks/agy/agy_guard.py`: `--stage dg1` có claim `D-*`/`demo` (hoặc `--stage dg12`) ghi luôn cả `06_laya_summary_dg1_*.json` và `06_laya_summary_dg2_*.json`.
- (C) `.agent-project.json` + `hooks/agy/agy_guard.py`: tắt `h6_auto_checks` ngầm tránh chạy test lặp và tránh race với `mutation_gate.py`; chỉ bơm `FINAL_AUDIT_MSG` ở lần sửa đầu tiên trong lượt; H7 `n == 0` chỉ chặn khi `miss` khác rỗng.
- (D) `tooling/verify_governance.py` + `hooks/agy/agy_guard.py`: gộp báo cáo Tier T2 vào `audit_report_<suffix>.md`.
- Hook unit tests: Ran 62 tests, OK (EV-20261021T111821-6aa2).
- Tooling unit tests: Ran 49 tests, OK (EV-20261021T111821-88b8).
- Runtime validate-surface: VALIDATED, OK (EV-20261021T111826-d20e).
- Mutation gate `--base 74840c8`: gate=PASS killed=9/9 (EV-20261021T111826-f68b).
- DG3 chốt: `06_laya_summary_dg3_20261021T111838.json`.

