# 01_requirements — Cleanup `_gpay_source_id` + `_gpay_deleted`

## Functional requirements
- **R-1** Audit chi tiết mọi reference của `_gpay_source_id` và `_gpay_deleted` trong codebase, phân loại theo path (Shadow-FE-A / Shadow-FE-B / Master-V2 / Test / UI).
- **R-2** Đánh giá liệu `source_id` thực sự thay thế được `_gpay_source_id` ở mỗi path (cùng semantic + data type).
- **R-3** Đánh giá liệu `_deleted` thực sự thay thế được `_gpay_deleted` ở mỗi path.
- **R-4** Trình ≥3 cleanup option (Conservative / Mid-Scope / Full) với pros/cons + code demo.
- **R-5** Recommend option an toàn nhất căn cứ lesson "anti-pattern over-correct" + lesson "Verify ở destination".
- **R-6** Đợi user pick option → Muscle apply.

## Non-functional
- **N-1** APPEND-only `05_progress.md` (§11 GEMINI).
- **N-2** Brain code prohibition (§12): phase này chỉ document, KHÔNG sửa source code.
- **N-3** Full doc set 00..10 + report (§7).
- **N-4** Lesson cross-check (§13).

## Acceptance criteria (Definition of Done — phase audit)
- [ ] 104 references đã phân loại theo path.
- [ ] Có 3 cleanup option với code demo cụ thể (file + line + before/after).
- [ ] Có decision matrix Pros/Cons/Risk/Reversibility/LOC cho từng option.
- [ ] Có recommend rõ ràng + lý do.
- [ ] Có verify plan (build + test) cho mỗi option.
- [ ] Workspace doc set 00..10 + report đầy đủ.
- [ ] **KHÔNG sửa source code** cho đến khi user pick option và verb "làm đi".

## User constraints (echo)
- Đọc lesson trước.
- Core /agent + GEMINI.md.
- Chỉ làm đúng yêu cầu.
- Không cheat DB hay đổi config.
- Plan rõ ràng + code demo chi tiết.
- Report dựa trên kết quả tính toán thực tế.
- Verify build/test trước khi báo done.
- Luôn có `report_*.md`.
