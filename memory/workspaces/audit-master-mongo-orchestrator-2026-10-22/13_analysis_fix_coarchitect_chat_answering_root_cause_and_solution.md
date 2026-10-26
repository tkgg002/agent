---
tier: T2
suffix: fix_coarchitect_chat_answering_root_cause_and_solution
---

# 13. Phân tích Chuyên sâu (S2 Investigate): Nguyên Nhân Gốc Rễ Của `CDC-RECON-01` (`Shadow -> Master`) & Lỗi Không Phản Hồi Trong `💬 THẢO LUẬN KIẾN TRÚC` (`task_chat.go`)

## PHẦN I: VÌ SAO KHI CHẠY RECON SHADOW -> MASTER (`CDC-RECON-01`) KẾT QUẢ KHÔNG ĐÚNG? (`VERIFIED file:line`)

Qua điều tra trực tiếp mã nguồn trên 3 repositories (`centralized-data-service`, `cdc-cms-service`, `cdc-cms-web`) và dữ liệu runtime thực tế (đã ghi nhận tại [`13_analysis_shadow_master_update_drop.md`](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/audit-shadow-master-update-drop-2026-10-20/13_analysis_shadow_master_update_drop.md) và [`CDC-RECON-01/09_tasks_solution.md`](file:///Users/trainguyen/Documents/work/ai-orchestrator/.orchestrator/tasks/CDC-RECON-01/09_tasks_solution.md)), có **8 sơ hở gốc rễ (`F1` -> `F8`)** làm kết quả đồng bộ & đối soát `Shadow -> Master` bị sai lệch:

1. **🔴 Sơ hở `F1` (`centralized-data-service` — Rớt `DELETE` do lỗi SQL `42703` & `binarySearchSplit` vứt bỏ nguyên mẻ `UPDATE`):**
   - **Điểm chạm:** [`master_ddl_generator.go:137-143, 411`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/master_ddl_generator.go#L137-L143), [`transmuter.go:672-688, 429-431`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L672-L688), [`transmute_handler.go:691-712`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/transmute_handler.go#L691-L712).
   - **Cơ chế lỗi:** DDL bảng Master luôn chạy `DROP COLUMN IF EXISTS "_source_id"`, nhưng `TransmuterModule.RunEvents()` và `Run()` vẫn hardcode `DELETE FROM %s WHERE _source_id = ?`, gây nổ lỗi `ERROR: column "_source_id" does not exist (SQLSTATE 42703)`. Khi lỗi rơi vào `TransmuteHandler.binarySearchSplit`, vì mỗi chu kỳ flush thường chỉ gửi `len(batch) == 1` NATS message chứa mảng nhiều `Events`, hàm `binarySearchSplit` không chia đôi mảng `Events` bên trong message đó mà đánh dấu nguyên message là `transmute_poison_pill` rồi ACK bỏ luôn, làm **rớt cả các sự kiện `UPDATE` hợp lệ đi chung mẻ**!

2. **🔴 Sơ hở `F2` (`centralized-data-service` — Split-Brain `_gpay_id` giữa `Run()` dùng Sonyflake và `RunEvents()` dùng SHA-256 làm mù Recon & Heal Segment B):**
   - **Điểm chạm:** [`transmuter.go:720, 1174, 1439`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L720), [`recon_stream_bucket_engine.go:715-748`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_stream_bucket_engine.go#L715-L748), [`recon_execute_heal_handler.go:653-734`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_execute_heal_handler.go#L653-L734).
   - **Cơ chế lỗi:**
     - Đường batch `Run()` lấy `_gpay_id` là số **Sonyflake** từ bảng Shadow (`deterministicGpayID`), trong khi đường sự kiện NATS `RunEvents()` lại tính `_gpay_id` bằng **băm SHA-256 của `_source_id`** (`deterministicSourceGpayID`).
     - Khi `bulkUpsertMaster` chạy `UPDATE`, nó lại bỏ qua không cập nhật cột `_gpay_id` (`if k == conflictTarget || k == "_gpay_id" { continue }`).
     - **Hậu quả khi chạy Recon Shadow -> Master (Segment B):** Bộ đối soát `resolveFieldsB` ưu tiên so sánh theo `pkCol = "_gpay_id"`, nên dù cùng 1 bản ghi `_id` đã có trên cả Shadow và Master nhưng một bên mang `_gpay_id` Sonyflake, một bên mang `_gpay_id` SHA-256 -> báo lệch giả hàng nghìn dòng (`missing_in_shadow` + `missing_in_master`: `1,482` dòng trên `payment_bills`, `520` dòng trên `export_jobs`).
     - **Hậu quả khi chạy Execute Heal Segment B:** `resolveSourceIDsForSegmentB` lấy `_gpay_id` (SHA-256) trên Master đem truy vấn `SELECT "_source_id" FROM shadow WHERE "_gpay_id" IN (...)` (vốn lưu Sonyflake) -> trả về **0 dòng**, khiến Heal báo `completed` nhưng số dòng lệch không giảm!

3. **🟠 Sơ hở `F3` (`centralized-data-service` — Quét Incremental `_gpay_id > lastGpayID` bỏ sót 100% `UPDATE` của dòng cũ & Heal Segment A không kích hoạt Transmute sang Master):**
   - **Điểm chạm:** [`transmuter.go:330-364, 978-984`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L330-L364), [`recon_heal.go:548-570, 679-737`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_heal.go#L548-L570).
   - **Cơ chế lỗi:** Trên bảng Shadow, `_gpay_id` chỉ sinh khi `INSERT` và giữ nguyên khi `UPDATE`. Nhưng `TransmuterModule.Run()` ở chế độ incremental chỉ lọc `WHERE ("_gpay_id")::bigint > currentLastID`, nên mọi bản ghi cũ được `UPDATE` (`_gpay_id <= currentLastID`) đều bị bỏ sót. Đồng thời khi `ReconHealer.HealMissingIDs` / `HealWindow` (Segment A: Mongo -> Shadow) ghi trực tiếp vào Shadow, nó không bắn NATS `cdc.cmd.transmute-shadow` sang Master.

4. **🟠 Sơ hở `F4` & `F5` (`centralized-data-service` — Đảo thứ tự `UPDATE` trong cùng cửa sổ Flush & Rớt cột khi `records[0]` thiếu key):**
   - **Điểm chạm:** [`batch_buffer.go:728-753`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/batch_buffer.go#L728-L753), [`transmuter.go:1412`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L1412).
   - **Cơ chế lỗi:** `BatchBuffer.batchUpsert` chia nhóm theo `getSignature(r)` trước khi khử trùng lặp theo `PrimaryKeyValue` và duyệt Go `map` ngẫu nhiên; `bulkUpsertMaster` chỉ lấy danh sách cột từ `sortedKeysAny(records[0])` làm rớt cột của các bản ghi sau nếu `records[0]` không có cột đó.

5. **🟠 Sơ hở `F6`, `F7`, `F8` (`centralized-data-service`, `cdc-cms-service`, `cdc-cms-web` — Cộng nhầm `DELETE` vào `Updated`, thiếu cột `Delete(d)` của Master và bỏ sót `transmute_poison_pill` trên bảng Đối soát):**
   - **Điểm chạm:** [`transmuter.go:708-726, 449, 1409`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L708-L726), [`daily_flow.go:36-83`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/queries/system/daily_flow.go#L36-L83), [`activity_log_read_repo_gorm.go:420, 444-550`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/system/activity_log_read_repo_gorm.go#L420), [`ReconPipelineGrid.tsx:840-844, 1309-1361`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ReconPipelineGrid.tsx#L840-L844).
   - **Cơ chế lỗi:** Khi xóa dòng trên Master thành công, `TransmuterModule` lại tăng `res.Updated++` (không có trường `Deleted`), khối **4. GHI VÀO DB MASTER** trên CMS/UI không có cột `Delete(d)`, và câu SQL `GetDailyFlow` chỉ lọc `al.operation IN ('transmute', 'transmute_sub')` mà bỏ sót `'transmute_poison_pill'` khiến `dlq_count = 0` và báo `BALANCED` sai sự thật!

---

## PHẦN II: VÌ SAO `BRAIN · ARCHITECT` TRONG `💬 THẢO LUẬN KIẾN TRÚC` (`task_chat.go`) KHÔNG PHẢN HỒI ĐÚNG CÂU HỎI CỦA USER? (`VERIFIED file:line`)

Tại [`internal/webui/task_chat.go:1325-1399`](file:///Users/trainguyen/Documents/work/ai-orchestrator/internal/webui/task_chat.go#L1325-L1399), hàm `buildContextualDiscussReply` chỉ kiểm tra đúng 2 nhánh `if` rất hẹp:
- Nhánh 1 (`L1350`): `if strings.Contains(lowerMsg, "mày là ai") || strings.Contains(lowerMsg, "model nào") || strings.Contains(lowerMsg, "role nào")`
- Nhánh 2 (`L1365`): `if strings.Contains(lowerMsg, "cảm nhận") || strings.Contains(lowerMsg, "antigravity") || strings.Contains(lowerMsg, "so với")`

Với mọi câu hỏi kỹ thuật khác của User (như *"rồi vì sao chạy ko đúng, giải pháp là gì"*), hàm này hoàn toàn **không đọc** `09_tasks_solution.md`, `13_analysis.md`, `audit_report.md` hay workspace `/agent`, mà rơi thẳng xuống `L1388-1398` để in ra đoạn văn bản khuôn mẫu vô nghĩa (nhại lại câu hỏi của User + in đường dẫn file).
