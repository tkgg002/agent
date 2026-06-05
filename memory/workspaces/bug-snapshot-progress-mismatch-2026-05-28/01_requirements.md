# 01_requirements — Bug Snapshot Progress Mismatch

## Mục tiêu fix
Đảm bảo `snapshot_progress.status = 'done'` IFF `rows_processed >= total_rows * threshold` (completeness invariant). Không bao giờ mark `done` khi cursor exit sớm hoặc khi user pause.

## Functional Requirements

| ID | Yêu cầu | Acceptance |
|---|---|---|
| FR-1 | Cursor exhaustion phải dựa trên `len(batch) == 0` (đã sẵn ở line 383-385), không dùng `len(batch) < BatchSize` để quyết định break. | Bỏ block `if len(batch) < p.BatchSize { break }` (line 553-555). |
| FR-2 | Khi `isPaused.Load() == true` ở giữa loop → set status=paused → return ngay, KHÔNG fall-through xuống `markProgressDone`. | Sau `break` pause (line 356) phải `return nil` thay vì để cursor loop kết thúc và chạy tiếp final flush + markProgressDone. |
| FR-3 | `markProgressDone` phải guard completeness: nếu `rowsTotal < totalRows * threshold (default 0.99)` → gọi `markProgressError` với reason `incomplete: rows_processed=X expected>=Y`. | Signature mới: `markProgressDone(ctx, progressID, rowsTotal, totalRows int64) error`. |
| FR-4 | Thêm Prometheus counter `cdc_snapshot_partial_done_total{reason="..."}` ghi nhận mỗi lần guard FR-3 trip. | Metric expose ở `/metrics`; tag `reason` ∈ `{cursor_short, pause_fallthrough, persist_mismatch}`. |
| FR-5 | Snapshot Monitor FE phải chỉ hiển thị `done` khi `progress=100%`. Snapshot Monitor không phải gốc bug nhưng cần align nhãn: khi BE trả `status=error` với reason `incomplete` → FE hiển thị badge `incomplete`. | Out-of-scope code change FE; chỉ assert behaviour FE đã đúng (đã render từ DB column). |

## Non-Functional Requirements

| ID | Yêu cầu |
|---|---|
| NFR-1 | Patch tối thiểu (Simplicity First §6). Không re-architect cursor loop. |
| NFR-2 | Không thêm dependency mới. Prometheus client đã có trong project. |
| NFR-3 | Backward compatible với `snapshot_progress` schema hiện tại. Không migration mới. |
| NFR-4 | Test integration phải synthetic được scenario "cursor partial mid-stream" để verify FR-1 + FR-3. |

## Definition of Done

| DoD | Mô tả | Verify |
|---|---|---|
| DoD-1 | Bug Root cause A fixed | Test `TestSnapshot_CursorPartialMidStream_ContinuesUntilEmpty` PASS. |
| DoD-2 | Bug Root cause B fixed | Test `TestSnapshot_PauseDoesNotFallThroughToDone` PASS — DB row `status=paused`, không `done`. |
| DoD-3 | Bug Root cause C fixed | Test `TestSnapshot_MarkDoneGuardsCompleteness` PASS — `rowsTotal < totalRows * 0.99` → status=error. |
| DoD-4 | Metric expose | `curl :PORT/metrics \| grep snapshot_partial_done_total` xuất hiện cả 3 reason. |
| DoD-5 | Governance §1+§12 | Brain plan-only; Muscle apply sau verb. |
| DoD-6 | Report có files thay đổi + LOC delta | `report_bug_snapshot_progress_mismatch_2026-05-28.md` có bảng table. |
| DoD-7 | Verify build + test PASS | `go build ./... && go test ./internal/handler/... -count=1` exit 0. |

## Out-of-scope
- Đổi Mongo ReadPreference từ `SecondaryPreferred` sang `Primary` (sẽ tăng tải primary; defer ADR-002).
- Sửa data đã mất (cheat DB).
- Re-architect snapshot.v2 toàn diện.
- Thêm UI control "Retry incomplete snapshot" (defer roadmap).

## Constraints từ user
- ✓ Đọc lesson trước (đã đọc `snapshot-zero-records-2026-05-27/`).
- ✓ Theo `agent/GEMINI.md` + `agent/memory/global/lessons.md`.
- ✓ Plan rõ ràng + code demo chi tiết.
- ✓ KHÔNG cheat DB, KHÔNG đổi config.
- ✓ Report cuối có **files thay đổi** + **số lượng dòng code thay đổi**.
- ✓ Verify service work trước khi báo done.
- ✓ Luôn có file `report_*.md`.
