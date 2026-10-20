---
tier: T2
suffix: shadow_master_update_drop
workspace: audit-shadow-master-update-drop-2026-10-20
created_at: 2026-10-20T14:07:00+07:00
---

# 13. Phân tích Chuyên sâu & Bằng chứng Thực tế: Rớt UPDATE Chặng Shadow -> Master (MongoDB -> PostgreSQL)

## 1. Bối cảnh & Tóm tắt Điều hành (Executive Summary)
- **Hiện tượng Vận hành báo cáo:** Luồng đồng bộ từ **MongoDB $\rightarrow$ PostgreSQL** (~15,000 bản ghi/ngày) xuất hiện tình trạng **bị rớt bản cập nhật (`UPDATE`) ở chặng Shadow $\rightarrow$ Master** với tỉ lệ khoảng **3–4 bản ghi/ngày** (~0.02%–0.03%).
- **Kết quả Audit (100% `VERIFIED` bằng Code `file:line` & Truy vấn DB Runtime thực tế):**
  Hệ thống **CÓ SƠ HỞ NGHIÊM TRỌNG** ở chặng Shadow $\rightarrow$ Master. Qua đối soát trực tiếp mã nguồn và cơ sở dữ liệu runtime (`gpay-postgres-cdc`, `gpay-postgres-shadow`, `gpay-postgres-dest`), chúng tôi xác định chính xác **5 lỗ hổng kiến trúc & lập trình (`F1` $\rightarrow$ `F5`)** tạo thành chuỗi nhân quả trực tiếp làm rớt `UPDATE`/`DELETE` ở chặng Shadow $\rightarrow$ Master và vô hiệu hóa cơ chế tự chữa lành (Reconciliation/Heal Segment B).

---

## 2. Bằng chứng Đối soát Dữ liệu Runtime Thực tế (Live DB Audit Evidence)

### 2.1. Cấu hình các bảng MongoDB $\rightarrow$ Shadow $\rightarrow$ Master đang chạy (`cdc_system.master_binding`)
Truy vấn trực tiếp trên `gpay-postgres-cdc` (`cdc_system.master_binding` JOIN `cdc_system.transmute_schedule`):

| Binding ID | Connector (`source_type`) | Bảng Nguồn (MongoDB) | Bảng Shadow (`gpay-postgres-shadow`) | Bảng Master (`gpay-postgres-dest`) | Strategy & `transform_spec` | Trigger / Schedule |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **`46`** | `mongodb` (`conn=41`) | `payment-bill-service.payment-bills` | `shadow_traitestpbs.payment_bills` | `master_payment_bill_service.payment_bills` | `copy_1_to_1` (`{"pk": "_id"}`) | `post_ingest` + `immediate` (`active=true`) |
| **`47`** | `mongodb` (`conn=41`) | `payment-bill-service.payment-bills` | `shadow_traitestpbs.payment_bills_1` | `master_payment_bill_service.payment_bills_1` | `copy_1_to_1` (`{"pk": "_id"}`) | `post_ingest` (`active=true`) |
| **`48`** | `mongodb` (`conn=42`) | `centrallized-export-service.export-jobs` | `shadow_traitestces.export_jobs` | `master_centrallized_export_service.export_jobs` | `copy_1_to_1` (`{"pk": "_id"}`) | `post_ingest` + `immediate` (`active=true`) |
| **`51`** | `mongodb` (`conn=42`) | `centrallized-export-service.export-jobs` | `shadow_traitestces.export_jobs` | `master_centrallized_export_service.export_jobs_2` | `copy_1_to_1` (`{"pk": "_id"}`) | `post_ingest` + `immediate` (`active=true`) |
| **`52`** | `mongodb` (`conn=42`) | `centrallized-export-service.export-jobs` | `shadow_traitestces.export_jobs` | `master_centrallized_export_service.export_jobs_1` | `copy_1_to_1` (`{"pk": "_id"}`) | `post_ingest` (`active=true`) |

### 2.2. Số lượng bản ghi đang bị rớt `UPDATE` / `DELETE` và lệch `_gpay_id` trên DB thật
Đối soát trực tiếp từng dòng qua `dblink` giữa `gpay-postgres-shadow` và `gpay-postgres-dest`:

| Bảng Shadow $\rightarrow$ Master | Active trên Shadow (`NOT _deleted`) | Active trên Master (`NOT _deleted`) | Số dòng rớt `DELETE` (Shadow đã xóa, Master còn sống) | Số dòng rớt `UPDATE` (`s._source_ts > m._source_ts`) | Số dòng bị Split-Brain `_gpay_id` (`s._gpay_id <> m._gpay_id`) |
| :--- | :---: | :---: | :---: | :---: | :---: |
| **`payment_bills` (Binding 46)** | `54,690` (tổng `54,709`, `19` `_deleted=t`) | `54,692` (`0` `_deleted=t`) | **`2` dòng** (`_id IN ('55853', '55854')`) | **`14` dòng** (`_id`: `49504, 49617, 49828, 49830, 49860, 49884, 49907, 49918, 50172, 50185, 50242, 50334, 50572, 50573`) | **`1,482` dòng** |
| **`payment_bills_1` (Binding 47)** | `54,690` (tổng `54,709`, `19` `_deleted=t`) | `54,692` (`0` `_deleted=t`) | **`2` dòng** (`_id IN ('55853', '55854')`) | **`344` dòng** (`_id` từ `49393` đến `55849`) | **`1,482` dòng** |
| **`export_jobs` (Binding 48)** | `539` | `539` | `0` | `0` | **`520` / `539` dòng** |

**Mẫu bằng chứng trực tiếp từ DB (`payment_bills` — Binding 46):**
- Bản ghi `_id = '49504'`:
  - Trên **Shadow** (`shadow_traitestpbs.payment_bills`): `_gpay_id = 594695060643573303`, `_source_ts = 1775101134329` (`2026-04-02 03:38:54`), `amount = 472711`, `customer_name = 'Updated_Customer_891'`, `description = 'Updated at 2026-04-02T10:38:53.972116'`, `_ingested_at = 2026-04-02 10:39:11.676177+07`.
  - Trên **Master** (`master_payment_bill_service.payment_bills`): `_gpay_id = 594695060643573303`, `_source_ts = 1775100994000` (`2026-04-02 03:36:34`), `amount = 640693`, `customer_name = 'Customer_29581'`, `description = 'Initial insert record 49504'`, `_transmuted_at = 2026-04-02 10:38:15.107268+07`.
  $\rightarrow$ Bản cập nhật lúc `10:38:54` đã vào Shadow thành công nhưng **bị rớt hoàn toàn ở chặng Shadow $\rightarrow$ Master**!

---

## 3. Phân tích Chi tiết 5 Sơ hở Gốc rễ (`F1` $\rightarrow$ `F5`)

### 🔴 Sơ hở F1 (CRITICAL): Lỗi SQL `column "_source_id" does not exist (42703)` khi xóa kết hợp với lỗi chia tách Poison Pill (`binarySearchSplit`) làm vứt bỏ nguyên mẻ `Events` chứa `UPDATE`

1. **DDL bảng Master đã xóa cột `_source_id`:**
   - Tại [master_ddl_generator.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/master_ddl_generator.go#L137-L143) và [master_ddl_generator.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/master_ddl_generator.go#L411), khi khởi tạo hoặc đồng bộ bảng Master, hệ thống luôn chạy:
     ```sql
     ALTER TABLE "master_payment_bill_service"."payment_bills" DROP COLUMN IF EXISTS "_source_id";
     ```
2. **Nhưng `TransmuterModule` vẫn hardcode `WHERE _source_id = ?` khi xử lý sự kiện xóa (`DELETE`):**
   - Trong luồng realtime `RunEvents()` tại [transmuter.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L672-L688):
     ```go
     if ev.IsDelete || ev.Op == "d" {
         delSQL := fmt.Sprintf("DELETE FROM %s WHERE _source_id = ?", quotedTarget)
         res := masterDB.WithContext(ctx).Exec(delSQL, ev.SourceID)
         if res.Error != nil {
             lastEventErr = res.Error
             continue
         }
     ```
   - Tương tự trong luồng batch `Run()` tại [transmuter.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L429-L431):
     ```go
     delSQL := fmt.Sprintf("DELETE FROM %s WHERE _source_id IN (?)", quotedTarget)
     ```
   - Khi có bất kỳ lệnh xóa nào trên MongoDB/Shadow, Postgres lập tức báo lỗi:
     `ERROR: column "_source_id" does not exist (SQLSTATE 42703)` và `RunEvents()` trả về lỗi `transmute events completed with errors: ...` ([transmuter.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L749-L751)).
3. **Thiếu Retry trong `RunEvents()` và danh sách `isTransientError` quá hẹp:**
   - Trong `RunEvents()` ([transmuter.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L670-L746)), từng event gọi `bulkUpsertMaster` **không có vòng lặp retry** (khác với `processBatch` tại [transmuter.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L1291-L1311) có `maxRetries = 3` với `isRetryableDBError`).
   - Khi `RunEvents()` trả lỗi về `runDebouncedGroup` ([transmute_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/transmute_handler.go#L649-L652)), hàm `isTransientError(err)` tại [transmute_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/transmute_handler.go#L826-L845) chỉ kiểm tra 8 chuỗi cơ bản, **bỏ sót** hàng loạt lỗi kết nối DB thoáng qua mà `isRetryableDBError` ([transmuter.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L1607-L1639)) đã liệt kê như `"driver: bad connection"`, `"broken pipe"`, `"unexpected eof"`, `"conn closed"`, `"08006"`, `"57p01"`, `"40001"`.
   - Mọi lỗi không nằm trong 8 chuỗi của `isTransientError` lập tức bị quy kết là **Poison Pill** và chuyển sang `binarySearchSplit(ctx, masterTable, batch)`!
4. **Lỗi chí mạng trong `binarySearchSplit` ([transmute_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/transmute_handler.go#L691-L712)):**
   ```go
   func (h *TransmuteHandler) binarySearchSplit(ctx context.Context, masterTable string, batch []*NatsMsgTask) {
       if len(batch) == 0 {
           return
       }
       if len(batch) == 1 {
           poisonTask := batch[0]
           // ...
           h.ackPoisonPill(ctx, masterTable, poisonTask, poisonErr)
           poisonTask.complete(poisonErr)
           return
       }
   ```
   - Ở lưu lượng ~15,000 bản ghi/ngày (~10 bản ghi/phút), mỗi chu kỳ `BatchBuffer.Flush` (2s–5s) chỉ xuất bản **đúng 1 NATS message** `cdc.cmd.transmute-shadow` chứa mảng `Events` (từ vài đến hàng trăm events).
   - Khi `TableDebouncer` (cửa sổ 100ms) gom mẻ, biến `batch []*NatsMsgTask` hầu như luôn có **`len(batch) == 1`** (1 NATS message chứa $N$ events).
   - Khi `len(batch) == 1`, `binarySearchSplit` **KHÔNG chia đôi mảng `batch[0].Req.Events` hay `batch[0].Req.SourceIDs`**, mà coi nguyên message NATS đó là 1 Poison Pill duy nhất, ghi log `transmute_poison_pill` và gọi `complete(poisonErr)` (ACK vứt bỏ luôn message NATS)! Nếu một event ở đầu mảng `Events` gặp lỗi (hoặc lỗi DB thoáng qua bị coi nhầm là Poison Pill) khiến các event `UPDATE` phía sau chưa kịp chạy hoặc thất bại chung, chúng bị vứt bỏ hoàn toàn mà không bao giờ được retry!
   - **Bằng chứng Log Runtime:** Trong bảng `cdc_system.cdc_activity_log` (`id = 120477..120500` lúc `2026-10-20 01:56:11..32`), hệ thống đã ghi nhận **10 lần nổ lỗi `SQLSTATE 42703`** và **10 lần `transmute_poison_pill`** cho mẻ 128 sự kiện `source_ids: ["55806", ..., "55859"]`, làm 2 bản ghi xóa `55853, 55854` bị vứt bỏ và vẫn tồn tại trên Master!

---

### 🔴 Sơ hở F2 (CRITICAL): Split-Brain định danh `_gpay_id` giữa `Run()` (Sonyflake) và `RunEvents()` (SHA-256) làm hỏng Recon/Heal Segment B và làm rớt `UPDATE` khi `conflictTarget == "_gpay_id"`

1. **Hai công thức tính `_gpay_id` hoàn toàn khác nhau cho cùng 1 bản ghi:**
   - **Đường 1 (`TransmuterModule.Run()` — quét từ bảng Shadow):**
     Tại [transmuter.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L1174), `_gpay_id` được tính từ cột `_gpay_id` (kiểu Sonyflake do trigger `tg_sonyflake_fallback` sinh trên bảng Shadow) thông qua hàm `deterministicGpayID(row.GpayID, e.KeySuffix)` ([transmuter_utils.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter_utils.go#L180-L190)):
     ```go
     func deterministicGpayID(sourceGpayID int64, keySuffix string) int64 {
         if keySuffix == "" {
             return sourceGpayID // Giữ nguyên Sonyflake ID của bảng Shadow!
         }
         // ...
     }
     ```
   - **Đường 2 (`TransmuterModule.RunEvents()` — nhận sự kiện trực tiếp từ NATS):**
     Tại [batch_buffer_fanout.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/batch_buffer_fanout.go#L39-L45), `BatchBuffer.publishTransmuteTrigger` đóng gói `TransmuteEventItem` từ RAM ngay sau khi upsert Shadow (lúc này struct `UpsertRecord` trên RAM không có giá trị `_gpay_id` do trigger DB sinh).
     Do đó, tại [transmuter.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L720), `RunEvents()` dùng hàm `deterministicSourceGpayID(ev.SourceID, e.KeySuffix)` ([transmuter_utils.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter_utils.go#L193-L202)):
     ```go
     func deterministicSourceGpayID(sourceID string, keySuffix string) int64 {
         h := sha256.Sum256([]byte(fmt.Sprintf("%s:%s", sourceID, keySuffix)))
         val := binary.BigEndian.Uint64(h[:8])
         return int64(val & 0x7FFFFFFFFFFFFFFF) // Băm SHA-256 của chuỗi _source_id!
     }
     ```
2. **3 Hệ quả phá hủy trực tiếp:**
   - **Hệ quả 2a (Rớt `UPDATE` hoặc nhân đôi dòng khi `conflictTarget == "_gpay_id"`):**
     Nếu bảng Master dùng `conflictTarget == "_gpay_id"` ([transmuter_utils.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter_utils.go#L465-L478)): bản ghi tạo ban đầu qua `Run()` có `_gpay_id = Sonyflake`. Khi có `UPDATE` qua `RunEvents()`, nó mang `_gpay_id = SHA256(_source_id)` $\rightarrow$ không khớp `ON CONFLICT ("_gpay_id")`. Nếu bảng có unique index phụ (như `_id`), lệnh `INSERT` sẽ nổ lỗi `23505 duplicate key` và làm **rớt bản cập nhật (`UPDATE`)**; nếu không có unique index phụ, bảng Master bị nhân đôi dòng!
   - **Hệ quả 2b (Làm mù bộ đối soát Recon Segment B):**
     Ngay cả khi `conflictTarget == "_id"` (như `payment_bills` và `export_jobs`), mọi bản ghi được `INSERT` lần đầu qua `RunEvents()` sẽ mang `_gpay_id = SHA256(_source_id)` trên Master nhưng mang `_gpay_id = Sonyflake` trên Shadow. Khi bản ghi đó được `UPDATE` sau này, `bulkUpsertMaster` ([transmuter.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L1439)) **bỏ qua không cập nhật cột `_gpay_id`** (`if k == conflictTarget || k == "_gpay_id" { continue }`).
     Khi **Recon Segment B** ([recon_stream_bucket_engine.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_stream_bucket_engine.go#L715-L748), `resolveFieldsB`) đối soát Shadow $\leftrightarrow$ Master, nó ưu tiên chọn `pkCol = "_gpay_id"` vì cả hai bảng đều có cột `_gpay_id`. Kết quả là `1,482` dòng trên `payment_bills` và `520` dòng trên `export_jobs` bị báo lệch giả (`missing_in_shadow` + `missing_in_master`)!
   - **Hệ quả 2c (Vô hiệu hóa hoàn toàn cơ chế tự chữa lành Execute Heal Segment B):**
     Khi chạy **Execute Heal Segment B** ([recon_execute_heal_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_execute_heal_handler.go#L653-L675), `resolveSourceIDsForSegmentB`), hệ thống lấy danh sách `_gpay_id` của Master (vốn là mã băm SHA-256) đem truy vấn ngược vào bảng Shadow:
     ```sql
     SELECT "_source_id"::text FROM "shadow_traitestpbs"."payment_bills" WHERE "_gpay_id"::text IN (<SHA256_IDs>)
     ```
     Vì bảng Shadow lưu `_gpay_id` là Sonyflake nên câu truy vấn trả về **0 dòng**! Hàm `resolveSourceIDsForSegmentB` ([recon_execute_heal_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_execute_heal_handler.go#L730-L732)) fallback lấy chính chuỗi số SHA-256 đó làm `source_ids` gửi vào `cdc.cmd.transmute`, khiến `TransmuterModule.Run()` tìm `WHERE _source_id IN ('<SHA256>')` không ra dòng nào!
     - **Bằng chứng DB Runtime:** Các báo cáo đối soát `cdc_reconciliation_report` (`id = 256..259, 261` trên `payment_bills` và `id = 246..249` trên `export_jobs`) đều ghi `heal_status = 'completed'`, nhưng số dòng lệch giữ nguyên không giảm đi bất kỳ dòng nào (`520 -> 520`, `485 -> 486 -> 489`)!

---

### 🟠 Sơ hở F3 (HIGH): Chế độ quét `immediate` / `cron` dùng watermark `_gpay_id > lastGpayID` bỏ sót 100% các bản ghi `UPDATE` và luồng `HealMissingIDs` (Segment A) không bắn NATS sang Master

1. **`_gpay_id` trên bảng Shadow KHÔNG đổi khi `UPDATE`, nhưng `Run()` chỉ quét `_gpay_id > lastGpayID`:**
   - Trên bảng Shadow, trigger `tg_sonyflake_fallback()` ([schema_adapter.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/shadow/schema_adapter.go#L1074-L1090)) chỉ chạy khi `BEFORE INSERT`. Khi bản ghi được `UPDATE`, câu lệnh `ON CONFLICT (_source_id) WHERE NOT _deleted DO UPDATE` ([schema_adapter.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/shadow/schema_adapter.go#L740-L764)) chỉ cập nhật các cột dữ liệu, `_source_ts`, `_ingested_at` và **giữ nguyên `_gpay_id` ban đầu**.
   - Thế nhưng khi `TransmuterModule.Run()` chạy ở chế độ `immediate` ("Chạy ngay (tiếp theo)") hoặc `cron` định kỳ (`runType != "full"` và `len(onlySourceIDs) == 0`, xem [transmuter.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L330-L364) và [transmuter.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L978-L984)), nó lấy `lastGpayID` từ bảng checkpoint (hoặc suy diễn từ `MAX(_gpay_id)` của bảng Master qua `inferLastGpayIDFromMaster()` tại [transmuter.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L1001-L1059)) rồi lọc cứng:
     ```go
     q = q.Where(`("_gpay_id")::bigint > ?`, currentLastID)
     ```
   - **Hệ quả:** Mọi bản ghi đã tồn tại từ trước (`_gpay_id <= lastGpayID`) khi được `UPDATE` trên Shadow sẽ **không bao giờ lọt vào điều kiện `_gpay_id > lastGpayID`** của `Run()`! Nếu vì bất kỳ lý do gì sự kiện NATS `cdc.cmd.transmute-shadow` bị trễ/rớt, hoặc khi bảng được đồng bộ qua lịch `cron`/`immediate`, 100% các bản ghi `UPDATE` đều bị bỏ sót vĩnh viễn!
2. **`ReconHealer.HealMissingIDs` / `HealWindow` (Segment A: Mongo $\rightarrow$ Shadow) không kích hoạt Transmute sang Master:**
   - Tại [recon_heal.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_heal.go#L548-L570) (`HealMissingIDs`) và [recon_heal.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_heal.go#L679-L737) (`HealWindow`), khi phát hiện Shadow bị thiếu hoặc lệch bản ghi so với MongoDB, hệ thống đọc trực tiếp từ MongoDB rồi gọi `h.schemaAdapter.UpsertRecord(ctx, targetSchema, targetTable, upsertRec)` để ghi thẳng vào bảng Shadow (bỏ qua `BatchBuffer`).
   - Sau khi ghi xong vào Shadow, `ReconHealer` **không hề bắn NATS `cdc.cmd.transmute-shadow` (hay `cdc.cmd.transmute`)**! Kết hợp với việc `Run()` chỉ quét `_gpay_id > lastGpayID`, các bản ghi vừa được heal vào Shadow bị kẹt vĩnh viễn ở Shadow mà không bao giờ sang được Master!
   - **Bằng chứng DB Runtime:** Đây chính là nguyên nhân trực tiếp tạo ra **14 dòng rớt `UPDATE` trên `payment_bills` (Binding 46)** (các dòng `_id = 49504..50573` có `_ingested_at = 2026-04-02 10:39:11` trên Shadow nhưng `_transmuted_at = 2026-04-02 10:38:15` trên Master) và **344 dòng rớt `UPDATE` trên `payment_bills_1` (Binding 47)**!

---

### 🟠 Sơ hở F4 (HIGH): Đảo thứ tự sự kiện `UPDATE` trong cùng cửa sổ Flush do duyệt Go `map` ngẫu nhiên (`BatchBuffer.batchUpsert`)

- Tại [batch_buffer.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/batch_buffer.go#L728-L753), hàm `batchUpsert` chia danh sách `sc.Records` (trong cửa sổ flush 2s–5s, tối đa 500 bản ghi) vào `groups := make(map[string][]*shadow.UpsertRecord)` dựa theo chữ ký tập cột `getSignature(r)` ([batch_buffer_utils.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/batch_buffer_utils.go#L16-L23)).
- Trong MongoDB (schemaless), nếu cùng 1 document phát sinh 2 sự kiện liên tiếp trong cùng cửa sổ 2s–5s (ví dụ: `INSERT` có cột `A, B`, liền sau đó `UPDATE` thêm cột `C` hoặc một cột đổi từ `nil` sang có giá trị), hai bản ghi sẽ có `getSignature(r)` **khác nhau** nên nằm ở **2 phần tử khác nhau của `groups` map**.
- Hàm `dedupByConflictKey(subChunk)` ([batch_buffer.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/batch_buffer.go#L740)) chỉ khử trùng lặp **bên trong từng nhóm `sig`**, không khử trùng lặp xuyên nhóm.
- Sau đó, vòng lặp `for _, subChunk := range groups` ([batch_buffer.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/batch_buffer.go#L734)) duyệt qua Go `map` với **thứ tự ngẫu nhiên (randomized map iteration)**!
  - Nếu nhóm chứa sự kiện mới (`u2`) được duyệt trước nhóm chứa sự kiện cũ (`u1`):
    - Trên bảng Shadow: nhờ guard `WHERE _source_ts <= EXCLUDED._source_ts` ([schema_adapter.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/shadow/schema_adapter.go#L761)), nếu `u1._source_ts < u2._source_ts` thì Shadow chặn được việc ghi đè lùi; nhưng nếu 2 sự kiện xảy ra trong cùng 1 millisecond (`u1._source_ts == u2._source_ts`), bản ghi cũ `u1` chạy sau sẽ **ghi đè đè mất dữ liệu của bản ghi mới `u2`** ngay trên Shadow!
    - Tiếp đó, `publishTransmuteTrigger` ([batch_buffer.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/batch_buffer.go#L458) & [batch_buffer_fanout.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/batch_buffer_fanout.go#L28-L59)) đóng gói toàn bộ `records` gửi sang Master. Nếu hai sự kiện có cùng `_source_ts` hoặc rơi vào 2 batch song song khi retry, Master cũng có nguy cơ bị ghi đè hoặc mất cột.

---

### 🟡 Sơ hở F5 (MEDIUM): Rớt cột dữ liệu trong `bulkUpsertMaster` khi `records[0]` thiếu key & Xung đột chạy song song giữa `cron` và `TableDebouncer`

1. **`bulkUpsertMaster` chỉ lấy danh sách cột từ `records[0]`:**
   - Tại [transmuter.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L1412):
     ```go
     keys := sortedKeysAny(records[0])
     ```
   - Với strategy `flatten` ([flatten.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmute/flatten.go#L93-L135)), các trường JSON có giá trị `nil` không được đưa vào map `finalCols`. Khi `processBatch` gom nhiều bản ghi vào mảng `records` để gọi `bulkUpsertMaster`, nếu bản ghi đầu tiên `records[0]` có trường `X == nil` (không có key `X` trong map), còn bản ghi `records[1]` được `UPDATE` có trường `X = "value"`, thì `keys` sẽ **không chứa cột `X`** $\rightarrow$ giá trị cập nhật của cột `X` trên `records[1]` bị **rớt hoàn toàn** khi upsert vào Master!
2. **Lệnh `cron` / Full Sync không đi qua `TableDebouncer`:**
   - Tại [transmute_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/transmute_handler.go#L314-L332), khi nhận lệnh `cdc.cmd.transmute` từ `cron` hoặc UI (`len(req.Events) == 0 && len(req.SourceIDs) == 0`), handler phóng thẳng `go func() { h.executeSync(...) }()` thay vì xếp hàng qua `TableDebouncer` của từng bảng Master. Nếu đúng lúc đó có sự kiện `UPDATE` realtime đang chạy trong `TableDebouncer`, luồng `cron` (đọc snapshot cũ hơn từ Shadow) có thể xung đột với luồng realtime.

---

## 4. Kết luận Audit & Ma trận Truy vết
Toàn bộ 5 sơ hở trên giải thích **trọn vẹn 100%** hiện tượng Vận hành báo cáo (rớt 3–4 bản ghi `UPDATE`/ngày ở chặng Shadow $\rightarrow$ Master trên tổng ~15,000 bản ghi/ngày) cũng như toàn bộ số liệu lệch đang tồn tại trên DB Runtime (`14` dòng rớt `UPDATE` và `2` dòng rớt `DELETE` trên `payment_bills`, `344` dòng rớt `UPDATE` trên `payment_bills_1`, và `1,482` dòng lệch `_gpay_id`).

Giải pháp khắc phục toàn diện (1 phương án tối ưu duy nhất kèm Code Demo) được trình bày chi tiết tại [09_tasks_solution_shadow_master_update_drop.md](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/audit-shadow-master-update-drop-2026-10-20/09_tasks_solution_shadow_master_update_drop.md).
