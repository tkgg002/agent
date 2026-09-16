# 09_tasks_solution_fix_recon_istimestamptz.md: Kế hoạch Giải pháp Chuẩn xác (Dùng trực tiếp Payload Fields)

## 1. Phân tích Nguyên nhân Gốc rễ

1. **Payload NATS/API mang sẵn `shadow_schema` và `shadow_table`:**
   ```json
   {
     "shadow_schema": "shadow_traitestctphs",
     "shadow_table": "trans_his",
     "table": "trans_his"
   }
   ```
2. **Điểm đứt gãy trong `ReconJobWorker.HandleJobEvent` ([recon_job_worker.go:253](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_job_worker.go#L253)):**
   - Mặc dù event payload có `ShadowSchema = "shadow_traitestctphs"` và `ShadowTable = "trans_his"`, nhưng sau khi gọi `GetTableConfig`, worker lại truyền `*entry` cũ (nếu `entry.ShadowSchema` rỗng) sang `ExecuteSegment`.
   - Kết quả: `entry.ShadowSchema` rỗng $\rightarrow$ `entry.QualifiedTarget()` trả tên bảng trần `trans_his` $\rightarrow$ `ColumnExists(ctx, "trans_his", "updatedAt")` tìm ở `table_schema = 'public'` bị `false`.
   - `resolveTSFields` lầm tưởng `updatedAt` không tồn tại ở Dest $\rightarrow$ Fallback nhầm về `dstTS = "_source_ts"`.
   - So sánh nhầm `updatedAt` (Mongo) vs `_source_ts` (Postgres) $\rightarrow$ Sinh ra 47 ID `Mismatched` và 2 ID trôi sub-window.

---

## 2. Kế hoạch Giải quyết Tối giản (100% Không viết thêm code hack)

1. **KHÔNG viết thêm hàm Khử trùng lặp (`deduplicateStaleIDsPayload`).**
2. **KHÔNG viết thêm Fallback Query xuyên schema.**
3. **Dùng trực tiếp Payload Fields:** Trong `ReconJobWorker.HandleJobEvent`:
   ```go
   entryCopy := *entry
   if event.ShadowSchema != "" {
       entryCopy.ShadowSchema = event.ShadowSchema
   }
   if event.ShadowTable != "" {
       entryCopy.TargetTable = event.ShadowTable
   }
   res, engineErr := w.engine.ExecuteSegment(ctx, entryCopy, reqSegment, event.StartTime, event.EndTime, event.MasterSchema, event.MasterTable)
   ```
4. Khi `entryCopy` mang đầy đủ `ShadowSchema = "shadow_traitestctphs"` và `TargetTable = "trans_his"`:
   - `entryCopy.QualifiedTarget()` trả về đúng `"shadow_traitestctphs.trans_his"`.
   - `ColumnExists(ctx, "shadow_traitestctphs.trans_his", "updatedAt")` query trực tiếp đúng `table_schema = 'shadow_traitestctphs' AND table_name = 'trans_his'` $\rightarrow$ Trả về `true`.
   - `resolveTSFields` lấy đúng `srcTS = "updatedAt"` và `dstTS = "updatedAt"`.
   - Đối soát `updatedAt` vs `updatedAt` khớp 100%, xóa sổ hoàn toàn 47 Mismatches và 2 IDs lặp mà không cần viết bất kỳ hàm hack nào!
