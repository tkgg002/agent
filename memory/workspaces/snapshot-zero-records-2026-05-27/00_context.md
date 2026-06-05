# 00_context — Audit Snapshot Zero Records

## Trigger
User report 2026-05-27 17:33:55 ICT:

> "rồi ko snapshot đc luôn. kiểm tra lại xem. http://localhost:5173/snapshot-monitor 17:33:55 27/5/2026 centrallized-export-service export-jobs done 161 161/161 100.00% 79523aa4d94ba5fa04cf6090 fe-snapshot-74a0bb97-... — báo 161/161 nhung ko có 1 record nào trong export_jobs_2 rất tào lao."

## Symptom
| Layer | Trạng thái |
|---|---|
| FE snapshot-monitor | `done` 161/161 — 100.00% |
| Activity log + snapshot_progress | `status=success, rows_processed=161` |
| Shadow table `export_jobs_2` | **0 rows** |
| Failed sync log | (chưa kiểm tra runtime — kỳ vọng có row do silent fallback) |

→ Counter ở source layer = `161` (= EstimatedDocumentCount từ Mongo Find).
→ Destination layer = 0 rows trong Postgres shadow.
→ Mismatch nghiêm trọng: **status ko phản ánh persistence reality**.

## Service liên quan
- `centralized-data-service` (chứa SnapshotRunner + EventHandler + BatchBuffer + SchemaAdapter).
- `cdc-cms-web` snapshot-monitor (chỉ display, không phải gốc bug).
- `cdc-cms-service` (publish `cdc.cmd.snapshot.v2`, không phải gốc bug).

## In-scope
- Trace toàn bộ chain: `cdc.cmd.snapshot.v2` → `SnapshotRunner.runSnapshot` → Mongo Find → `HandleRaw` → `processEvent` → `batchBuffer.Add` → `Flush` → `batchUpsert` → PG insert.
- Tìm bước nào báo "success" nhưng thực tế không persist.
- Đưa ra fix tối thiểu cho **observability + correctness** (counter đo persist, không đo enqueue).

## Out-of-scope
- Sửa data đã mất (ko cheat DB).
- Re-architect snapshot.v2.
- Migration mới.

## Liên hệ workspace trước
- `bug-first-snapshot-no-write-2026-05-26/` — đã fix 1 lớp tương tự: `HandleRaw` trả `(rows, err)` + route-empty trip CB. Fix đó CÒN nguyên (đã grep verify trong session trước). Bug hôm nay là LỚP MỚI nằm SAU `HandleRaw`: `BatchBuffer.Flush()` swallow error, không propagate về snapshot_runner.
- `audit-shadow-create-bugs-2026-05-27/` — vừa hoàn tất fix CREATE TABLE DDL ở cdc-cms-service. Có thể đây là nguồn lỗi side-channel (table thiếu cột, INSERT fail) — nhưng bug ROOT là **silent-swallow** che giấu mọi lỗi insert.

## Constraint từ user
- Đọc lesson trước.
- Đọc `agent/GEMINI.md` + `agent/memory/global/lessons.md` trước khi planning.
- Plan rõ ràng, có code demo chi tiết.
- Report cuối: file thay đổi + LOC delta + verify build/test.
- KHÔNG cheat DB. KHÔNG đổi config.
- KHÔNG báo done nếu chưa verify.

## Cross-reference Lesson
- `lessons.md` 2026-05-26 dòng 3417-3421 (workspace `bug-first-snapshot-no-write-2026-05-26`):
  > **Define DoD at the destination**: counter must measure persistence at destination, not dispatch/enqueue. Otherwise success metric will lie when intermediate layer fails silently.
- Bug hôm nay là **case study trực tiếp** của lesson này. Lớp `BatchBuffer.Flush()` chính là layer "enqueue thành công, persist fail nhưng counter báo OK".
