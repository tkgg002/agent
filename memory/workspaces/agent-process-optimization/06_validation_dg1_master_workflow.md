# 06 — Validation: DG1 cho `09_tasks_solution_master_workflow.md`

> 2026-10-16 10:09 | Laya `127.0.0.1:8200` (PID 13360) | Client: `laya_claim_check.py` (r1) + `laya_debate_rounds.py` (r2, r3), là client thật đã dùng từ 2026-10-04
> Claims: `06_laya_claims_dg1_master_workflow.json` | Raw: `06_laya_raw_dg1_r1_master_workflow.json`, `06_laya_raw_dg1_r2_r3_master_workflow.json`

## Hiệu chỉnh round bằng control

| ID | Loại | r1 | r2 | r3 | Đánh giá |
|---|---|---|---|---|---|
| K1 | control_false | INCORRECT 0.10 ✓ (yếu) | CORRECT 0.76 ✗ | YES 0.91 ✗ | r2, r3 sai mà tự tin |
| K2 | control_true | CORRECT 0.31 ✓ (yếu) | CORRECT 0.73 ✓ | YES 0.96 ✓ | |

**Trusted rounds: không có** (r1 không có control đạt ≥ 0.5; r2/r3 sai K1 với ≥ 0.5) → batch = **UNVALIDATED**.
*Batch này chỉ có 1 control_false (thiết kế mới yêu cầu ≥ 2) → đây là lần chạy thăm dò trước khi có `laya_gate.py`.*

## Claims

| ID | Claim | r1 | r2 | r3 | Đã kiểm trong tài liệu gốc | Kết luận |
|---|---|---|---|---|---|---|
| C1 | Output của PostToolUse là `{}`, không bơm thông điệp được | CORRECT 0.18 | CORRECT 0.10 | YES 0.12 | `agy-customizations/docs/hooks.md:235` | **VERIFIED (doc)** — UNVALIDATED |
| C2 | PostInvocation bơm được thông điệp + force_continue | INCORRECT 0.12 | CORRECT 0.03 | YES 0.56 | `docs/hooks.md:273-291` | **VERIFIED (doc)** — UNVALIDATED |
| C3 | Stop `decision=continue` chặn dừng + bơm reason | CORRECT 0.10 | CORRECT 0.87 | YES 0.98 | `docs/hooks.md:320-323` | **VERIFIED (doc)** — UNVALIDATED |
| C4 | Rule file > 24.000 byte bị cắt | INCORRECT 0.05 | CORRECT 0.17 | YES 0.58 | `docs/rules.md:31-33` + `wc -c` = 30949 + `<truncated` trong context | **VERIFIED (doc + runtime)** — UNVALIDATED |
| C5 | `skills.json` entries chỉ quét 1 cấp | CORRECT 0.45 | CORRECT 0.27 | NO 0.32 | `docs/json_configs.md:48-50` | **VERIFIED (doc)** — UNVALIDATED |
| C6 | muscle-execute có `git push` | CORRECT 0.84 | INCORRECT 0.30 | NO 0.002 | `workflows/muscle-execute.md:112-126` | **VERIFIED (code)** — UNVALIDATED |

## Kết luận
- Cả 6 claim nền tảng đã được kiểm trực tiếp trong tài liệu/code → giữ nguyên trong thiết kế.
- Laya **không phân biệt được** ở batch này (K1 bị r2/r3 xác nhận sai với 0.76/0.91). Đây chính là bằng chứng cho quy tắc "hiệu chỉnh round bằng control" trong mục 3.2 của 09.
