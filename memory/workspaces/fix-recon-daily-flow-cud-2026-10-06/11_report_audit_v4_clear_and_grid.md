# 11 — Report: Audit V4 P1 (Clear Activity Log + Daily-flow Grid)

Ngày: 2026-10-12 · Trạng thái: P1 DONE (đã verify trên data thật), P2 chờ quyết định

## Files thay đổi trong P1 (phạm vi phiên này)
| Repo | File | Thay đổi |
|---|---|---|
| centralized-data-service | `internal/handler/shadow/batch_buffer.go` | Ghi `shadow_schema` vào details lúc Complete; nhánh lỗi chuyển sang `FailWithDetails` (shadow_schema, buffer_type, is_snapshot) |
| centralized-data-service | `internal/handler/shadow/batch_buffer_fanout.go` | `triggered_by="snapshot"` khi là snapshot buffer |
| centralized-data-service | `internal/handler/master/transmute_handler.go` | Forward `snapshot-hook`; debounce gom theo (binding, trigger) qua `groupDebouncedTasks` + `runDebouncedGroup`; `master_binding_id` ở mọi Fail/Complete; details đầy đủ cho `transmute_sub` |
| centralized-data-service | `internal/handler/master/transmute_debounce_group_test.go` (mới) | 2 unit test cho phần gom nhóm |
| cdc-cms-service | `internal/app/queries/system/list_activity_logs.go` | `ClearActivityParams{ShadowSchema, ShadowTable, SourceDatabase, MasterTable, Before *time.Time}` |
| cdc-cms-service | `internal/infra/persistence/system/activity_log_read_repo_gorm.go` | LATERAL ưu tiên `shadow_schema`/`master_binding_id`; viết lại `ClearActivity` dùng lại `enrichmentFromClause`, bỏ qua dòng ambiguous, bound `before` cho partition pruning |
| cdc-cms-service | `internal/api/system/activity_log_handler.go` | Validate scope (400), `before` dạng RFC3339 |
| cdc-cms-service | `internal/router/router.go` | DELETE `/activity-log` chuyển từ nhóm `shared` sang `destructiveChain` |
| cdc-cms-web | `src/hooks/useReconStatus.ts` | Payload mới; `reason` chỉ gửi qua header |
| cdc-cms-web | `src/components/ReconPipelineGrid.tsx` | Chia ngày theo giờ local (dayjs); thêm op `transmute_sub`; nhận diện snapshot-hook; `status==='error'`; bỏ fallback `'cdc_shadow'`; guard trong modal; **masterTable truyền dạng bare (BUG-P1-FE)** |

`git diff --stat` (lưu ý: CMS/Web/CDS đang gộp diff chưa commit của nhiều phiên từ 30/09, nên con số lớn hơn phạm vi P1):
- CDS (3 file handler): 582 dòng thêm, 112 dòng xoá, cộng 1 file test mới
- CMS: 5 file, 135 dòng thêm, 5 dòng xoá
- Web: 2 file, 980 dòng thêm, 96 dòng xoá (gộp nhiều phiên)

## Bằng chứng
Xem `06_validation_audit_v4_clear_and_grid.md` → mục "KẾT QUẢ THỰC THI": S1–S6 PASS, leak=0, EXPLAIN xác nhận pruning.

## Hạn chế còn tồn tại (đã biết, chưa sửa)
1. **H5 / P2**: grid tổng hợp từ tối đa 200 log → số ngày cũ có thể thiếu. Cần làm query `daily-flow` phía server.
2. Số DLQ có thể đếm thừa: khi transmute bị split, log lỗi của batch cha và log lỗi của từng sub cùng bị cộng.
3. Khối KAFKA được suy ra từ log shadow, nên phép so kafka==shadow là hiển nhiên đúng, chưa đối soát thật (GAP-V4-6).
4. Log cũ (chưa có `shadow_schema`) bị ambiguous khi nhiều connector trùng tên bảng → Clear bỏ qua, để partition retention tự dọn.
5. Rủi ro có từ trước: `chain()` trong router dùng `make(0,len)` và bỏ qua middleware nil → có thể `append` chung backing array. Chưa sửa (ngoài scope).

## Deploy
Phải redeploy **cả CDS worker lẫn CMS** (và web). Chỉ log ghi sau khi deploy mới mang identity.
