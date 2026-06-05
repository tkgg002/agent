# 01_requirements — Audit Snapshot Zero Records

## Functional requirements
- **R-1** Trace toàn bộ chain `cdc.cmd.snapshot.v2` → PG insert; xác định layer nào báo success nhưng KHÔNG persist.
- **R-2** Root-cause: tìm exact file/line nơi error bị silent-swallowed, đối chiếu với counter mà `markProgressDone(rowsTotal)` sử dụng.
- **R-3** Đề xuất fix minimal-impact (Plan A) — chỉ plumb `(written, err)` qua chain Flush; không refactor BatchBuffer.
- **R-4** Code demo chi tiết tới từng patch site (file, line range, before/after snippet).
- **R-5** Verify build PASS sau khi apply (chờ user verb).

## Non-functional
- **N-1** APPEND-only `05_progress.md` (§11 GEMINI).
- **N-2** Brain code prohibition (§12 GEMINI): chỉ document, KHÔNG sửa source code trong phase audit.
- **N-3** Full doc set 00..10 + report (§7).
- **N-4** Lesson cross-check (§13): bug là case study của "Define DoD at the destination" — không cần lesson mới, chỉ APPEND evidence.

## Acceptance criteria (Definition of Done)
- [ ] Đã xác định CHÍNH XÁC file:line nơi `Flush()` swallow error.
- [ ] Đã xác định CHÍNH XÁC file:line nơi `rowsTotal += written` đếm enqueue thay vì persist.
- [ ] Đã đề xuất signature change cho `Flush() → (int, error)` và `FlushBatchBuffer() → (int, error)`.
- [ ] Đã viết code demo cho ≥4 patch site (Flush, batchUpsert, FlushBatchBuffer, runSnapshot lines 516+550).
- [ ] Đã ghi report audit phase.
- [ ] User approve "làm đi" → Muscle apply → build + test PASS.

## User constraints (echo lại)
- Đọc lesson trước.
- Theo core `/agent` + `agent/GEMINI.md`.
- Chỉ làm đúng yêu cầu.
- Không cheat DB, không đổi config.
- Plan rõ ràng, code demo chi tiết.
- Report dựa trên kết quả tính toán thực tế.
- Verify build trước khi báo done.
- Luôn có file `report_*.md`.
