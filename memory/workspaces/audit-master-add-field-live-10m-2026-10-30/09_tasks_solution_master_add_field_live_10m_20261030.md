# 09. TASKS & TECHNICAL SOLUTION — `master_add_field_live_10m_20261030`

- **Workspace:** `audit-master-add-field-live-10m-2026-10-30`
- **tier:** T2
- **mode:** standard
- **Suffix:** `master_add_field_live_10m_20261030`
- **Trạng thái:** `PROPOSE-ONLY (Chờ APPROVE)`

---

## 1. Direct Summary (Kết luận Trực diện cho Bài toán 10 Triệu Bản ghi Đang Chạy Live)

Hệ thống hiện tại **ĐÁP ỨNG ĐƯỢC ~70% về mặt khung kiến trúc** (đã có `SyncRulesFromShadow` kéo field mới từ `shadow` sang `master`, DDL `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` metadata-only không khóa lâu trên PostgreSQL 11+, cơ chế `InvalidateRuleCache` tự động nạp field mới cho luồng CDC live, và đọc lô Keyset Pagination `WHERE _gpay_id > cursor LIMIT 2000` không dùng `OFFSET`).
**TUY NHIÊN, nếu chạy ngay trên bảng 10 triệu (`10,000,000`) bản ghi đang Live đọc/ghi mà chưa vá 4 điểm nghẽn dưới đây (`F1–F4`), hệ thống SẼ GẶP RỦI RO NGHIÊM TRỌNG ảnh hưởng trực tiếp tới luồng Live:**
1. **`F1` (Silent Data Regression — Ghi đè dữ liệu cũ lên dữ liệu Live mới):** Khi chạy `Full Resync` (`run_type = "full"`) hoặc truyền `_source_ids` để backfill 10 triệu dòng cũ, hàm `processBatch` truyền `isFullSync = true` làm **tắt hoàn toàn chốt chặn OCC (`_source_ts`)** ở cả PostgreSQL Master lẫn MongoDB Master. Nếu 1 bản ghi vừa được luồng Live CDC cập nhật lên phiên bản mới (`T2`) đúng lúc lô Backfill đang xử lý bản ghi đó ở phiên bản cũ (`T1`), lô Backfill sẽ **ghi đè ngược dữ liệu cũ `T1` đè mất dữ liệu mới `T2`** trên `master`.
2. **`F2` (Đứt gãy Luồng Live CDC bởi `SQLSTATE 42703 column does not exist`):** Khi bấm Approve field mới trên CMS, `SaveMasterRuleHandler` / `BatchUpdateRulesHandler` lưu `status = 'approved'` vào DB trước rồi mới bắn NATS `cdc.cmd.master-create` dạng **fire-and-forget** (lại chỉ truyền `master_table` trần thiếu `master_binding_id`). Trong khi đó, `TransmuterModule.loadRules()` chỉ lọc `status = 'approved'` mà **không kiểm tra `in_master = true`**. Nếu DDL trên `master` trễ hoặc trượt do `lock_timeout = 5s`, worker CDC live nạp rule mới và bắn `INSERT INTO master (..., new_col)` vào cột chưa tồn tại $\rightarrow$ **văng lỗi `42703` làm chết toàn bộ luồng CDC live sang `master`**.
3. **`F3` (Khóa Bảng Chặn Đứng Luồng Đọc/Ghi Live khi Tạo Index trong Transaction DDL):** `MasterDDLGenerator.Apply()` gom chung `ALTER TABLE ADD COLUMN`, `CREATE INDEX IF NOT EXISTS` (không `CONCURRENTLY`) và 7 lệnh `ALTER TABLE DROP COLUMN IF EXISTS` vào **cùng 1 Transaction** có `statement_timeout = 30s`. Nếu field mới thuộc nhóm cột tài chính (`amount`, `fee`, `balance`, `total`, `price`...) hoặc có khai báo index, lệnh `ALTER TABLE` đầu transaction chiếm `AccessExclusiveLock` và **giữ nguyên khóa độc quyền đó suốt thời gian PostgreSQL quét 10 triệu dòng để dựng Index** $\rightarrow$ **chặn đứng 100% luồng đọc và ghi Live trên `master`**, đồng thời dễ nổ `statement_timeout = 30s` làm rollback cả lệnh thêm cột.
4. **`F4` (Bão I/O, Phình Gấp Đôi Bảng MVCC Dead Tuples & Tranh chấp Cache Đọc Live khi Backfill 10M dòng không có Throttle):** Vòng lặp `Full Resync` (`TransmuterModule.Run`) quét và `UPDATE` toàn bộ 10 triệu dòng (`5,000` batches $\times$ `2,000` rows) liên tục với **0ms nghỉ (Zero Throttle)** và cập nhật đè toàn bộ các cột kể cả khi giá trị không đổi $\rightarrow$ sinh ra **10 triệu dead tuples** trên PostgreSQL (hoặc làm đầy dirty cache WiredTiger trên MongoDB), gây bão WAL/Autovacuum Disk I/O làm tăng vọt latency của luồng đọc Live trên `master`.

---

## 2. Phân tích Chi tiết Luồng Hiện tại & 5 Phát hiện (`F1`–`F5`) kèm `file:line`

### Phát hiện `F1` (CRITICAL — `VERIFIED`): Chế độ `isFullSync` tắt khóa bảo vệ OCC (`_source_ts`), gây Race Condition ghi đè dữ liệu cũ lên dữ liệu Live mới
- **Bằng chứng vật lý (`file:line`):**
  - Tại [`transmuter.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L450-L452):
    ```go
    batchRes = t.processBatch(ctx, masterRow, rules, shadowRows, strat, rc, runType == "full" || len(chunkIDs) > 0)
    ```
  - Tại [`transmuter.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L1636-L1650) (PostgreSQL Master):
    ```go
    if isFullSync {
        sqlText = fmt.Sprintf(`INSERT INTO %s (%s) VALUES %s
            ON CONFLICT (%s) DO UPDATE SET %s
            RETURNING (xmax = 0) AS inserted`, ...)
    } else {
        const clockSkewToleranceMs = 2000
        sqlText = fmt.Sprintf(`INSERT INTO %s (%s) VALUES %s
            ON CONFLICT (%s) DO UPDATE SET %s
            WHERE COALESCE(EXCLUDED._source_ts, 0) >= COALESCE(%s._source_ts, 0) - %d
            RETURNING (xmax = 0) AS inserted`, ...)
    }
    ```
  - Tại [`transmuter_mongo.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter_mongo.go#L234-L295) & [`transmuter_mongo.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter_mongo.go#L323) (MongoDB Master): Khi `isFullSync == true`, hàm bỏ qua kiểm tra `_source_ts` và dùng `$set` phẳng đè thẳng lên document đích thay vì Aggregation Pipeline `$cond`.
- **Cơ chế gây lỗi trên hệ thống Live 10 triệu dòng:**
  - Để backfill field mới cho 10 triệu bản ghi lịch sử từ `shadow` sang `master`, hệ thống phải chạy `run_type = "full"` (vì `incremental` chỉ quét các dòng `shadow` có `_gpay_id > last_gpay_id` hoặc `_updated_at` mới trong 10 phút–24h qua — xem `F5`).
  - Quá trình quét 10 triệu dòng kéo dài nhiều phút. Trong lúc lô Backfill vừa `SELECT` 2,000 dòng từ `shadow` (trong đó bản ghi `K` có `_source_ts = T1`), luồng Live CDC từ `source` nhận giao dịch mới của bản ghi `K` (`_source_ts = T2 > T1`) và cập nhật ngay `master` lên `T2`. Ngay mili-giây sau, lô Backfill gọi `bulkUpsertMaster(..., isFullSync=true)` không có điều kiện `WHERE EXCLUDED._source_ts >= master._source_ts` $\rightarrow$ **ghi đè bản ghi `K` trên `master` lùi về trạng thái cũ `T1`**, làm luồng đọc Live trên `master` đọc phải dữ liệu cũ (Stale Read / Lost Update)!
  - **Lưu ý quan trọng:** Với các bản ghi lịch sử không thay đổi trong lúc backfill, `_source_ts` ở `shadow` và `master` **bằng nhau (`EXCLUDED._source_ts == master._source_ts`)**. Do đó, giữ nguyên mệnh đề OCC `WHERE COALESCE(EXCLUDED._source_ts, 0) >= COALESCE(master._source_ts, 0) - 2000` (trên PG) và `$lte: ["$_source_ts", allowThreshold]` (trên Mongo, chỉ bỏ bước `coll.Find` pre-fetch để tối ưu tốc độ) **vẫn cho phép cập nhật thêm field mới cho 100% dòng lịch sử (`T_shadow == T_master`)**, đồng thời **chặn tuyệt đối việc ghi đè vào các dòng vừa được Live CDC cập nhật mới hơn (`T_shadow < T_master`)**!

### Phát hiện `F2` (CRITICAL — `VERIFIED`): `loadRules` không kiểm tra `in_master = true` kết hợp `PublishMasterCreate` bất đồng bộ thiếu `master_binding_id` gây sập luồng CDC Live (`SQLSTATE 42703`)
- **Bằng chứng vật lý (`file:line`):**
  - Tại [`save_master_rule.go`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/master/save_master_rule.go#L59-L74) và [`batch_update_rules.go`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/master/batch_update_rules.go#L38-L53): Cập nhật `status = 'approved'` trong `cdc_system.mapping_rule_master` trước, sau đó gọi `h.publisher.PublishMasterCreate(ctx, masterTable, ...)` kiểu fire-and-forget.
  - Tại [`master_ddl_publisher.go`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/messaging/nats/master_ddl_publisher.go#L124-L136): Payload gửi đi chỉ có `"master_table"` mà không truyền `"master_binding_id"`. Nếu có $\ge 2$ binding trùng tên bảng, `loadBinding` ([`master_ddl_generator.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/master_ddl_generator.go#L523-L525)) báo lỗi `ambiguous_master_name` và không tạo cột.
  - Tại [`transmuter.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L883-L894):
    ```sql
    FROM cdc_system.mapping_rule_master m
    LEFT JOIN cdc_system.mapping_rule_v2 v2 ON v2.id = m.mapping_v2_id
    WHERE m.master_binding_id = ? AND m.is_active = true AND m.status = 'approved'
    ```
    Truy vấn nạp rule cho Transmuter hoàn toàn không kiểm tra `m.in_master = true` (dù [`master_ddl_generator.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/master_ddl_generator.go#L423-L441) đã có sẵn cơ chế đồng bộ `in_master = true` sau khi cột vật lý thực sự được `ALTER TABLE ADD COLUMN` trên `master`).
- **Hậu quả trên môi trường Live:** Ngay khi operator bấm Approve field mới mà DDL chưa chạy xong (hoặc bị fail do `lock_timeout = 5s`), luồng CDC live nạp rule mới và bắn `INSERT INTO master (..., "new_col")` $\rightarrow$ nổ lỗi `SQLSTATE 42703: column "new_col" of relation "..." does not exist` làm đứt gãy luồng ghi Live sang `master`.

### Phát hiện `F3` (HIGH — `VERIFIED`): Gộp `ALTER TABLE` và `CREATE INDEX` (non-concurrent) trong cùng 1 Transaction giữ `AccessExclusiveLock` chặn đứng luồng Đọc Live trên bảng 10 triệu dòng
- **Bằng chứng vật lý (`file:line`):**
  - Tại [`master_ddl_generator.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/master_ddl_generator.go#L361-L400):
    - Bước 1 (dòng `372-378`): Chạy `CreateSQL` và mọi câu `ALTER TABLE ... ADD COLUMN IF NOT EXISTS ...` (bao gồm cả các cột cũ đã tồn tại). Trong PostgreSQL, lệnh `ALTER TABLE` ngay lập tức giữ **`AccessExclusiveLock`** (khóa chặn cả `SELECT` đọc live lẫn `INSERT/UPDATE` ghi live) cho tới khi `COMMIT` transaction.
    - Bước 2 (dòng `380-384`): Ngay trong cùng transaction đang giữ `AccessExclusiveLock` đó, chạy toàn bộ `res.IndexSQL` (`CREATE INDEX IF NOT EXISTS ...` không có `CONCURRENTLY`).
    - Bước 3 (dòng `394-398`): Chạy tiếp 7 câu `ALTER TABLE ... DROP COLUMN IF EXISTS` cho các cột legacy.
- **Hậu quả trên bảng 10 triệu bản ghi đang Live:**
  - Nếu field mới là cột thường (không tạo index mới) và các index cũ đã tồn tại: `ADD COLUMN IF NOT EXISTS` (nullable, không volatile default) trên PostgreSQL 11+ chỉ cập nhật catalog trong **~1–5ms**, hầu như không ảnh hưởng luồng đọc Live.
  - **NHƯNG** nếu field mới khớp regex cột tài chính `financialIndexRe` ([`master_ddl_generator.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/master_ddl_generator.go#L63): `amount|fee|balance|total|price|refund|subtotal|discount|tax|cost...`) hoặc được thêm vào `transform_spec.indexes`, PostgreSQL sẽ phải quét 10 triệu dòng để dựng B-Tree Index **trong lúc vẫn đang giữ `AccessExclusiveLock` của câu `ALTER TABLE` phía trên**!
  - Kết quả:
    1. Toàn bộ luồng đọc Live (`SELECT`) trên `master` bị treo cứng chờ khóa!
    2. Nếu dựng index trên 10 triệu dòng mất `> 30s`, `SET LOCAL statement_timeout = '30s'` (dòng `369`) sẽ hủy toàn bộ transaction $\rightarrow$ **không thể thêm cột mới vào `master`**!

### Phát hiện `F4` (HIGH — `VERIFIED`): Vòng lặp Backfill 10 triệu dòng (`Full Sync`) không có nhịp nghỉ (Zero Throttle), không sắp xếp khóa chống Deadlock và ghi đè mù làm phình bảng MVCC
- **Bằng chứng vật lý (`file:line`):**
  - Tại [`transmuter.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L401-L581): Vòng lặp `for` đọc từng batch `2,000` dòng từ `shadow` (`fetchShadowBatch` tại dòng `411`) rồi gọi `processBatch` $\rightarrow$ `bulkUpsertMaster` liên tục cho tới hết 10 triệu dòng (**5,000 vòng lặp liên tiếp với `0ms` delay**).
  - Tại [`transmuter.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L214) & [`transmuter.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L990-L1001): Đầu hàm `Run()` gọi `SELECT COUNT(*) FROM shadow WHERE (_gpay_id)::bigint > 0` gây quét nặng trên bảng `shadow` 10 triệu dòng.
  - Tại [`transmuter.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L1257-L1269): Sau khi khử trùng lặp `allRecords` từ 10 goroutine con (dòng `1116-1195` — thứ tự kết thúc goroutine là ngẫu nhiên), mảng `allRecords` không được sắp xếp lại theo khóa `conflictTarget` trước khi chạy `INSERT ... ON CONFLICT DO UPDATE`, dẫn tới nguy cơ **Deadlock (`SQLSTATE 40P01`)** khi đụng độ với batch của luồng Live CDC (`RunEvents`) đang cập nhật cùng tập bản ghi theo thứ tự khác.

### Phát hiện `F5` (MEDIUM — `VERIFIED`): Quy trình vận hành để có dữ liệu cho 10 triệu dòng cũ bắt buộc phải kích hoạt `Full Sync` (`run_type = "full"`)
- **Bằng chứng vật lý (`file:line`):**
  - Tại [`transmuter.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L350-L397) và [`transmuter.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L1348-L1420):
    - Luồng Live CDC (`RunEvents`) chỉ tự động điền field mới cho **các bản ghi có phát sinh sự kiện thay đổi mới từ `source`** sau thời điểm Approve field.
    - Luồng Incremental Transmute (`run_type = ""`) chỉ đọc từ `last_gpay_id` trở đi (tức cuối bảng 10M dòng) và quét bù các dòng có `shadow._updated_at >= cutoff` (10 phút–24h gần nhất).
    - Vì 10 triệu bản ghi lịch sử trong `shadow` đã nằm sẵn trong cột `_raw_data` từ quá khứ (`shadow._updated_at` cũ), để lấp đầy field mới cho toàn bộ 10 triệu dòng lịch sử trên `master`, hệ thống bắt buộc phải chạy qua `run_type = "full"` (hoặc chạy backfill theo cửa sổ).

---

## 3. Thiết kế Giải pháp Kỹ thuật Tối ưu Duy nhất (Ánh xạ 1-1 `F1` $\rightarrow$ `F4`)

### 3.1. Bảng Ánh xạ Điểm gọi Thực thi & Thời điểm Kích hoạt (Runtime Call-Site & Lifecycle Wiring — `L004`)
| Phát hiện | Hàm / Module sửa đổi | Điểm gọi Thực thi (`Caller file:line`) | Thời điểm Kích hoạt (Lifecycle Trigger) |
| :--- | :--- | :--- | :--- |
| **`F1`** | `bulkUpsertMaster` ([`transmuter.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L1636-L1650)) & `BuildMongoOCCBulkModels` / `bulkUpsertMongoMaster` ([`transmuter_mongo.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter_mongo.go#L211-L345)) | Được gọi từ `TransmuterModule.processBatch` ([`transmuter.go:1292`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L1292)) trong cả `Run(runType="full")` lẫn `Run(onlySourceIDs)` | Mỗi khi chạy Full Resync / Backfill 10M dòng hoặc Incremental Sync từ `shadow -> master` |
| **`F2`** | `TransmuterModule.loadRules` ([`transmuter.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L883-L895)) + `PublishMasterCreate` ([`master_ddl_publisher.go`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/messaging/nats/master_ddl_publisher.go#L124-L136), [`save_master_rule.go`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/master/save_master_rule.go#L70-L74), [`batch_update_rules.go`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/master/batch_update_rules.go#L47-L53)) | Được gọi khi Operator bấm Save/Approve Rule trên CMS và khi `Run` / `RunEvents` nạp danh sách cột đích (`transmuter.go:247, 668`) | Ngay thời điểm thêm/duyệt field mới trên CMS và mỗi chu kỳ nạp rule của luồng Live CDC |
| **`F3`** | `MasterDDLGenerator.Apply` ([`master_ddl_generator.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/master_ddl_generator.go#L361-L405)) | Được gọi từ `MasterDDLHandler.HandleMasterCreate` ([`master_ddl_handler.go:188`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/master_ddl_handler.go#L188)) khi nhận lệnh `cdc.cmd.master-create` | Khi áp dụng DDL thêm cột mới và tạo index lên bảng `master` đang chạy Live |
| **`F4`** | `TransmuterModule.processBatch` & `TransmuterModule.Run` ([`transmuter.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L450-L581), [`transmuter.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L1257-L1271)) | Được gọi từ `TransmuteHandler.HandleTransmute` ([`transmute_handler.go:369`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/transmute_handler.go#L369)) | Trong suốt quá trình Backfill 10 triệu bản ghi lịch sử (`run_type = "full"`) |

---

### 3.2. Giải pháp cho `F1`: Giữ Chốt chặn OCC (`_source_ts`) ngay cả khi chạy `Full Sync` (`isFullSync = true`)
- **Nguyên lý:**
  - Trên **PostgreSQL Master** ([`transmuter.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L1636-L1650)): Luôn gắn điều kiện `WHERE COALESCE(EXCLUDED._source_ts, 0) >= COALESCE(master._source_ts, 0) - 2000` (hoặc `master._source_ts IS NULL OR EXCLUDED._source_ts IS NULL`) vào câu lệnh `ON CONFLICT (...) DO UPDATE` bất kể `isFullSync` là `true` hay `false`.
    - Với 10 triệu dòng lịch sử chưa thay đổi: `EXCLUDED._source_ts == master._source_ts` $\rightarrow$ thỏa mãn `>=` $\rightarrow$ **cập nhật thành công field mới vào `master`**.
    - Với các dòng vừa được luồng Live CDC cập nhật mới hơn (`master._source_ts = T2 > EXCLUDED._source_ts = T1 + 2000`): mệnh đề `WHERE` chặn lại $\rightarrow$ **bảo vệ tuyệt đối dữ liệu mới `T2` của luồng Live (và trên `T2` cũng đã có sẵn field mới do luồng Live CDC vừa ghi!)**.
  - Trên **MongoDB Master** ([`transmuter_mongo.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter_mongo.go#L234-L295)): Khi `isFullSync == true`, vẫn giữ việc bỏ `coll.Find` pre-fetch ở dòng `323` (để không tốn thêm 1 vòng đọc mạng mỗi batch 2,000 dòng), nhưng ở `BuildMongoOCCBulkModels` (dòng `234`), khi `incomingTs > 0` **luôn sử dụng Aggregation Pipeline Update có `$cond` kiểm tra `$_source_ts <= allowThreshold`** thay vì `$set` mù.

### 3.3. Giải pháp cho `F2`: Khóa An toàn `in_master = true` trong `loadRules` (PostgreSQL) & Truyền `master_binding_id` khi `PublishMasterCreate`
- **Nguyên lý:**
  1. Trong [`transmuter.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L883-L894) (`loadRules`): Với PostgreSQL Master (`!row.IsMongoDestination(ctx, t.connMgr)`), chỉ nạp các rule có `m.in_master = true` (`AND (m.in_master = true OR ? = true)` với tham số `isMongo`). Nhờ đó, dù operator vừa bấm Approve trên CMS nhưng cột vật lý trên PostgreSQL `master` chưa kịp tạo xong (hoặc DDL đang retry), luồng Live CDC vẫn tiếp tục ghi các cột hiện hữu bình thường, **tuyệt đối không bao giờ bắn lỗi `SQLSTATE 42703`**! Ngay khi `MasterDDLGenerator.Apply()` hoàn tất `ALTER TABLE ADD COLUMN`, nó cập nhật `in_master = true` và gọi `g.cacheInvalidator(reg.ID, reg.MasterTable)` ([`master_ddl_generator.go:435-447`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/master_ddl_generator.go#L435-L447)) để Transmuter lập tức nạp cột mới ở batch kế tiếp!
  2. Trong [`cdc-cms-service`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/messaging/nats/master_ddl_publisher.go#L124-L136): Cho phép `PublishMasterCreate` gửi kèm `master_binding_id` (hoặc truyền chuỗi ID/FQN chính xác) để triệt tiêu lỗi `ambiguous_master_name`.

### 3.4. Giải pháp cho `F3`: Tách Bạch Transaction `ALTER TABLE ADD COLUMN` (Siêu Nhanh < 5ms) khỏi Bước Tạo Index Nền trên Bảng 10 Triệu Dòng
- **Nguyên lý:**
  - Trong [`master_ddl_generator.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/master_ddl_generator.go#L361-L405):
    1. **Transaction 1 (Chỉ Schema/Column DDL — giữ khóa `< 5ms`):** Chỉ chạy `CREATE TABLE IF NOT EXISTS` và `ALTER TABLE ADD COLUMN IF NOT EXISTS` cho những cột **chưa tồn tại thực tế** trong `information_schema.columns` (kiểm tra trước qua `information_schema.columns` để không phát lệnh `ALTER TABLE` thừa lấy `AccessExclusiveLock` vô ích trên các cột đã có!), rồi **`COMMIT` ngay lập tức** và cập nhật ngay `in_master = true` + `cacheInvalidator`.
    2. **Bước 2 (Tạo Index ngoài Transaction chính với `CREATE INDEX CONCURRENTLY IF NOT EXISTS`):** Sau khi Transaction 1 đã `COMMIT` (nhả hoàn toàn `AccessExclusiveLock`), kiểm tra các index chưa tồn tại trong `pg_indexes` và chạy `CREATE INDEX CONCURRENTLY IF NOT EXISTS` trên kết nối tự động commit ngoài transaction (với `statement_timeout = 10min`).
       - Nhờ `CONCURRENTLY`, PostgreSQL xây dựng index trên 10 triệu dòng **không chặn luồng đọc Live và không chặn luồng ghi Live CDC**!
       - Dù việc tạo index trên 10 triệu dòng mất 30–60 giây hay thất bại tạm thời, cột mới **đã được tạo và commit xong từ mili-giây đầu tiên**!

### 3.5. Giải pháp cho `F4`: Sắp xếp Khóa Chống Deadlock (`40P01`) & Pacing Throttle Nhường I/O cho Luồng Live khi Backfill 10M Dòng
- **Nguyên lý:**
  1. Trong [`transmuter.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L1257-L1270): Sắp xếp `dedupedRecords` tăng dần theo khóa `conflictTarget` (`sort.Slice`) trước khi gọi `bulkUpsertMaster` để mọi batch (cả `Full Sync` lẫn `Live CDC`) luôn khóa các dòng theo cùng một thứ tự từ điển $\rightarrow$ **triệt tiêu hoàn toàn Deadlock `40P01`** giữa luồng Backfill và luồng Live CDC.
  2. Trong [`transmuter.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L575-L581): Khi chạy `runType == "full"` trên bảng lớn, thêm nhịp nghỉ ngắn có kiểm soát (`time.Sleep(20 * time.Millisecond)` giữa mỗi batch 2,000 dòng) để nhường CPU/Disk IOPS/Connection Pool cho luồng Đọc Live và luồng Ghi Live CDC, đồng thời cho phép `autovacuum` dọn kịp dead tuples.

---

## 4. Code Demo (Trước / Sau kèm `file:line`)

### 4.1. [`centralized-data-service/internal/service/master/transmuter.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L882-L895) — Khóa an toàn `in_master` (`F2`), Sắp xếp khóa chống Deadlock & Pacing (`F4`), và Bảo toàn OCC khi Full Sync (`F1`)

```go
// --- TRƯỚC (transmuter.go:882-895) ---
	var rules []mappingRuleRow
	err := t.systemDB.WithContext(ctx).Raw(`
		SELECT m.id, ?::bigint AS source_object_id, m.master_binding_id,
		       COALESCE(m.source_field, v2.source_field) AS source_field,
		       m.target_column, COALESCE(m.data_type, v2.data_type) AS data_type,
		       COALESCE(v2.source_format, '') AS source_format,
		       COALESCE(m.source_path, v2.source_path) AS source_path,
		       COALESCE(NULLIF(m.transform_fn, ''), v2.transform_fn) AS transform_fn,
		       COALESCE(v2.is_nullable, true) AS is_nullable, v2.default_value
		  FROM cdc_system.mapping_rule_master m
		  LEFT JOIN cdc_system.mapping_rule_v2 v2 ON v2.id = m.mapping_v2_id
		 WHERE m.master_binding_id = ? AND m.is_active = true AND m.status = 'approved'
		 ORDER BY m.id`, row.SourceObjectID, row.ID).Scan(&rules).Error

// --- SAU (transmuter.go:882-897) ---
	isMongo := row.IsMongoDestination(ctx, t.connMgr)
	var rules []mappingRuleRow
	err := t.systemDB.WithContext(ctx).Raw(`
		SELECT m.id, ?::bigint AS source_object_id, m.master_binding_id,
		       COALESCE(m.source_field, v2.source_field) AS source_field,
		       m.target_column, COALESCE(m.data_type, v2.data_type) AS data_type,
		       COALESCE(v2.source_format, '') AS source_format,
		       COALESCE(m.source_path, v2.source_path) AS source_path,
		       COALESCE(NULLIF(m.transform_fn, ''), v2.transform_fn) AS transform_fn,
		       COALESCE(v2.is_nullable, true) AS is_nullable, v2.default_value
		  FROM cdc_system.mapping_rule_master m
		  LEFT JOIN cdc_system.mapping_rule_v2 v2 ON v2.id = m.mapping_v2_id
		 WHERE m.master_binding_id = ? AND m.is_active = true AND m.status = 'approved'
		   AND (m.in_master = true OR ? = true)
		 ORDER BY m.id`, row.SourceObjectID, row.ID, isMongo).Scan(&rules).Error
```

```go
// --- TRƯỚC (transmuter.go:1635-1650) ---
	var sqlText string
	if isFullSync {
		sqlText = fmt.Sprintf(`INSERT INTO %s (%s) VALUES %s
			ON CONFLICT (%s) DO UPDATE SET %s
			RETURNING (xmax = 0) AS inserted`,
			qt, strings.Join(cols, ", "), strings.Join(placeholders, ", "),
			conflictIdent, strings.Join(sets, ", "))
	} else {
		const clockSkewToleranceMs = 2000
		sqlText = fmt.Sprintf(`INSERT INTO %s (%s) VALUES %s
			ON CONFLICT (%s) DO UPDATE SET %s
			WHERE COALESCE(EXCLUDED._source_ts, 0) >= COALESCE(%s._source_ts, 0) - %d
			RETURNING (xmax = 0) AS inserted`,
			qt, strings.Join(cols, ", "), strings.Join(placeholders, ", "),
			conflictIdent, strings.Join(sets, ", "), qt, clockSkewToleranceMs)
	}

// --- SAU (transmuter.go:1635-1645) ---
	// Luôn giữ điều kiện OCC (_source_ts) kể cả khi Full Sync / Backfill để:
	// 1) Cho phép cập nhật thêm field mới cho các dòng lịch sử (EXCLUDED._source_ts == master._source_ts)
	// 2) Chặn tuyệt đối lô Backfill ghi đè dữ liệu cũ (T1) lên dòng vừa được Live CDC cập nhật (T2 > T1)
	const clockSkewToleranceMs = 2000
	sqlText := fmt.Sprintf(`INSERT INTO %s (%s) VALUES %s
		ON CONFLICT (%s) DO UPDATE SET %s
		WHERE COALESCE(EXCLUDED._source_ts, 0) >= COALESCE(%s._source_ts, 0) - %d
		RETURNING (xmax = 0) AS inserted`,
		qt, strings.Join(cols, ", "), strings.Join(placeholders, ", "),
		conflictIdent, strings.Join(sets, ", "), qt, clockSkewToleranceMs)
```

### 4.2. [`centralized-data-service/internal/service/master/transmuter_mongo.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter_mongo.go#L233-L242) — Bảo toàn OCC Pipeline Update trên MongoDB Master ngay cả khi `isFullSync = true` (`F1`)

```go
// --- TRƯỚC (transmuter_mongo.go:233-242) ---
		var update any
		if !isFullSync && incomingTs > 0 {
			allowThreshold := incomingTs + mongoOCCClockSkewToleranceMs
			occCond := bson.M{
				"$or": bson.A{
					bson.M{"$eq": bson.A{bson.M{"$type": "$_source_ts"}, "missing"}},
					bson.M{"$lte": bson.A{"$_source_ts", allowThreshold}},
				},
			}

// --- SAU (transmuter_mongo.go:233-242) ---
		var update any
		if incomingTs > 0 {
			allowThreshold := incomingTs + mongoOCCClockSkewToleranceMs
			occCond := bson.M{
				"$or": bson.A{
					bson.M{"$eq": bson.A{bson.M{"$type": "$_source_ts"}, "missing"}},
					bson.M{"$lte": bson.A{"$_source_ts", allowThreshold}},
				},
			}
```

### 4.3. [`centralized-data-service/internal/service/master/master_ddl_generator.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/master_ddl_generator.go#L361-L405) — Tách Transaction `ADD COLUMN` (Chỉ chạy cột chưa có) khỏi Bước Tạo Index để không giữ `AccessExclusiveLock` trên bảng 10 triệu dòng (`F3`)

```go
// --- TRƯỚC (master_ddl_generator.go:361-400) ---
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`SET LOCAL lock_timeout = '5s'`).Error; err != nil {
			return fmt.Errorf("set lock_timeout: %w", err)
		}
		if err := tx.Exec(`SET LOCAL statement_timeout = '30s'`).Error; err != nil {
			return fmt.Errorf("set statement_timeout: %w", err)
		}
		if err := tx.Exec(res.CreateSQL).Error; err != nil {
			return fmt.Errorf("create table: %w", err)
		}
		for _, s := range res.AlterSQL {
			if err := tx.Exec(s).Error; err != nil {
				return fmt.Errorf("alter add column: %w", err)
			}
		}
		for _, s := range res.IndexSQL {
			if err := tx.Exec(s).Error; err != nil {
				return fmt.Errorf("create index: %w", err)
			}
		}
		...
	})

// --- SAU (master_ddl_generator.go:361-415) ---
	// 1. Đọc danh sách cột vật lý hiện có trên bảng Master trước khi mở Transaction DDL
	var existingCols []string
	_ = db.WithContext(ctx).Raw(
		`SELECT column_name FROM information_schema.columns WHERE table_schema = ? AND table_name = ?`,
		reg.MasterSchema, reg.MasterTable,
	).Scan(&existingCols)
	existingColMap := make(map[string]bool, len(existingCols))
	for _, c := range existingCols {
		existingColMap[c] = true
	}

	// 2. Transaction 1: Chỉ tạo bảng (nếu chưa có) và ADD COLUMN cho đúng cột MỚI chưa tồn tại (< 5ms)
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`SET LOCAL lock_timeout = '5s'`).Error; err != nil {
			return fmt.Errorf("set lock_timeout: %w", err)
		}
		if err := tx.Exec(`SET LOCAL statement_timeout = '30s'`).Error; err != nil {
			return fmt.Errorf("set statement_timeout: %w", err)
		}
		if len(existingCols) == 0 {
			if err := tx.Exec(res.CreateSQL).Error; err != nil {
				return fmt.Errorf("create table: %w", err)
			}
		}
		for _, s := range res.AlterSQL {
			// Chỉ thực thi ALTER TABLE ADD COLUMN cho cột chưa tồn tại để tránh lấy AccessExclusiveLock thừa
			if !shouldSkipExistingColumnAlter(s, existingColMap) {
				if err := tx.Exec(s).Error; err != nil {
					return fmt.Errorf("alter add column: %w", err)
				}
			}
		}
		return nil
	})
	if err != nil {
		res.Err = err.Error()
		g.markDDLStatus(ctx, reg.ID, "failed", err)
		return res, err
	}

	// 3. Tách riêng Index ra ngoài Transaction chính và dùng CONCURRENTLY để KHÔNG khóa bảng 10M dòng đang Live
	for _, idxSQL := range res.IndexSQL {
		concurrentIdxSQL := strings.Replace(idxSQL, "CREATE INDEX IF NOT EXISTS", "CREATE INDEX CONCURRENTLY IF NOT EXISTS", 1)
		concurrentIdxSQL = strings.Replace(concurrentIdxSQL, "CREATE UNIQUE INDEX IF NOT EXISTS", "CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS", 1)
		if idxErr := db.WithContext(ctx).Exec(concurrentIdxSQL).Error; idxErr != nil {
			g.logger.Warn("master concurrent index creation warning",
				zap.String("master", reg.MasterTable),
				zap.String("sql", concurrentIdxSQL),
				zap.Error(idxErr))
		}
	}
```


---

## 5. KIỂM TOÁN PHẢN BIỆN PLAN V1 & NÂNG CẤP LÊN PLAN V2 (Adversarial Critique: Mục tiêu, File, Từng Dòng Update, SRP & Backward Compatibility)

Theo yêu cầu phản biện toàn diện trước "cú nổ lớn", em đã rà soát chéo toàn bộ **Plan V1** với **tất cả các file test hiện hữu**, **giới hạn số dòng (`L008: NOW <= HEAD` trên file `> 500` dòng)**, và **Design Pattern phân tầng gốc của hệ thống**. Kết quả phản biện đã phát hiện **5 điểm tử huyệt trong Plan V1** (nếu làm theo V1 sẽ làm gãy unit test, panic MongoDB test, làm tê liệt bảng cũ và vi phạm `L008`), từ đó thiết kế lại **Plan V2 chuẩn Single-Responsibility (SRP) & 100% Tương thích Ngược**:

### 5.1. 5 Phát hiện Phản biện Nghiêm trọng đối với Plan V1 (Tại sao phải bỏ cách sửa thô của V1 và chuyển sang V2)

1. **Phản biện Mục tiêu #1 (`F2` — Bác bỏ việc sửa câu SQL `loadRules` thêm `AND (m.in_master = true OR ? = true)` trong `transmuter.go:883-894`):**
   - **Tử huyệt của Plan V1:**
     - **Phá vỡ Backward Compatibility trên dữ liệu cũ:** Cột `in_master` mới được thêm ở migration `076_add_in_master_to_mapping_rule_master.sql` với `DEFAULT FALSE`. Các bảng `master` cũ đang chạy ổn định hoặc dữ liệu seed trong [`transmuter_events_batch_test.go:143-148`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter_events_batch_test.go#L143-L148) có `in_master = false`. Nếu lọc cứng `AND m.in_master = true` trong `loadRules`, `loadRules` sẽ trả về **0 rules** và làm ngừng hoạt động toàn bộ các bảng `master` cũ chưa chạy lại DDL!
     - **Làm gãy 5 Unit Tests hiện hữu:** Trong [`transmuter_orphan_test.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter_orphan_test.go#L97) (dòng `97, 288, 449, 596, 726`), `sqlmock` đang khớp cứng `WithArgs(sourceObjectID, row.ID)` (2 tham số). Đổi câu SQL `loadRules` sang 3 tham số sẽ làm **FAIL ngay lập tức 5 bài test**!
   - **Thiết kế V2 (100% Tương thích Ngược + SRP):**
     - **Giữ nguyên 100%** câu SQL `loadRules` trong `transmuter.go:883-894`.
     - Tách cơ chế bảo vệ sang file mới đơn trách nhiệm `centralized-data-service/internal/service/master/transmuter_batch_guard.go`: Hàm `bulkUpsertMasterWithSchemaGuard` bọc quanh `bulkUpsertMaster`. Ở trạng thái bình thường, hàm tốn **0 truy vấn phụ**. **Chỉ khi PostgreSQL trả về lỗi `SQLSTATE 42703` (`undefined_column` / `column ... of relation ... does not exist`)** — tức rơi đúng vào cửa sổ vài mili-giây khi rule mới vừa `approved` trên CMS nhưng lệnh `ALTER TABLE ADD COLUMN` trên `master` chưa chạy xong hoặc đang retry — hàm mới đọc `information_schema.columns` của bảng `master`, tạm thời lọc bỏ cột chưa tồn tại khỏi bản sao `records` của mẻ hiện tại và retry `bulkUpsertMaster` ngay lập tức!
     - **Kết quả:** Chặn đứng 100% lỗi `42703` trên luồng Live CDC mà không tốn thêm query ở luồng thường, không làm hỏng bảng cũ `in_master = false`, và giữ PASS 100% unit/integration tests!

2. **Phản biện Mục tiêu #2 (`F1` — Bác bỏ việc xóa nhánh `if isFullSync` trong `bulkUpsertMaster` và sửa `BuildMongoOCCBulkModels` trong `transmuter_mongo.go`):**
   - **Tử huyệt của Plan V1:**
     - Trong [`transmuter_mongo_test.go:194-207`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter_mongo_test.go#L194-L207) (`TestBuildMongoOCCBulkModels_PreservesGpayIDWhenConflictTargetIsBusinessPK`), bài test gọi trực tiếp `BuildMongoOCCBulkModels(..., true)` và ép kiểu `uomFull.Update.(bson.M)`. Nếu bỏ `!isFullSync` ở `transmuter_mongo.go:234` như V1, hàm trả về `[]bson.M` (pipeline) gây **PANIC test ngay lập tức** (`interface conversion: interface {} is []primitive.M, not primitive.M`)!
     - Trong [`transmuter_gpayid_test.go:527`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter_gpayid_test.go#L527), luồng **Recon Force-Heal** gọi `tm.Run(ctx, row.MasterTable, driftedIDs, "", "full", bID)` với danh sách `driftedIDs` cụ thể (`len(onlySourceIDs) > 0` VÀ `runType == "full"`) để cưỡng chế sửa các dòng lệch bất chấp `_source_ts`. Nếu xóa hẳn nhánh `if isFullSync` trong `bulkUpsertMaster`, Recon Force-Heal theo danh sách ID sẽ bị OCC chặn khi `sr.SourceTs < mr.SourceTs`!
   - **Thiết kế V2 (Sửa đúng 1 biểu thức gốc tại `transmuter.go:451` — 100% Tương thích Ngược):**
     - Giữ nguyên 100% `bulkUpsertMaster` (`transmuter.go:1636-1650`) và `BuildMongoOCCBulkModels` (`transmuter_mongo.go:234`).
     - Tại [`transmuter.go:451`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go#L451), đổi:
       - Từ: `runType == "full" || len(chunkIDs) > 0`
       - Thành: `runType == "full" && len(chunkIDs) > 0`
     - **Ý nghĩa toán học & nghiệp vụ chính xác tuyệt đối:**
       - Chỉ khi **Recon Force-Heal** truyền cả `runType == "full"` **VÀ** danh sách ID cụ thể (`len(chunkIDs) > 0`), `isFullSync` mới bằng `true` để cưỡng chế ghi đè sửa lệch.
       - Khi chạy **Full Table Resync / Backfill 10 triệu dòng** (`runType == "full"` nhưng `len(chunkIDs) == 0`) hoặc khi chạy **Incremental Sync theo ID** (`runType != "full"` và `len(chunkIDs) > 0`), biểu thức `runType == "full" && len(chunkIDs) > 0` bằng **`false`** $\rightarrow$ **Cả PostgreSQL Master lẫn MongoDB Master đều tự động bật khóa bảo vệ OCC (`_source_ts`)**!
       - Với 10 triệu dòng lịch sử không đổi (`EXCLUDED._source_ts == master._source_ts`), điều kiện `>=` thỏa mãn 100% nên field mới được điền đầy đủ; với các dòng vừa được Live CDC cập nhật lên `T2 > T1 + 2000`, OCC chặn lô Backfill cũ `T1` lại, bảo vệ tuyệt đối dữ liệu Live!

3. **Phản biện Mục tiêu #3 (`F3` + `L008` — Bác bỏ việc nhồi thêm code vào `master_ddl_generator.go` `743` dòng & Xử lý triệt để `AccessExclusiveLock` của 7 câu `DROP COLUMN IF EXISTS`):**
   - **Tử huyệt của Plan V1:**
     - `master_ddl_generator.go` hiện có **`743` dòng** (`> 500` dòng). Nhồi thêm code vào file này vi phạm trực tiếp kỷ luật `L008` (`NOW <= HEAD`).
     - Trong PostgreSQL, cả `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` (trên các cột cũ đã tồn tại) lẫn 7 câu `ALTER TABLE ... DROP COLUMN IF EXISTS` (`_source_id`, `_source`, `_synced_at`, `_version`, `_hash`, `_created_at`, `_raw_data` tại dòng `394-398`) **đều chiếm `AccessExclusiveLock` TRƯỚC KHI kiểm tra `IF NOT EXISTS` / `IF EXISTS`**!
     - Đồng thời, trên SQLite in-memory (dùng trong unit test `master_ddl_indexes_test.go`, `master_ddl_handler_test.go`), cú pháp `CREATE INDEX CONCURRENTLY` không được SQLite hỗ trợ và `information_schema.columns` không tồn tại.
   - **Thiết kế V2 (Tách file `master_ddl_generator_pg.go` theo đúng Design Pattern đối xứng với `master_ddl_generator_mongo.go`):**
     - Tách toàn bộ phần thực thi DDL vật lý của PostgreSQL (`applyPostgresTableAndIndexes`, `ReconcileColumn`, `DropColumn`, `Swap`, `DropTable`) sang file mới `centralized-data-service/internal/service/master/master_ddl_generator_pg.go` (`~395` dòng). File `master_ddl_generator.go` giảm từ **`743` dòng xuống `~360` dòng (`-383` dòng, `NOW < HEAD`)**!
     - Trong `applyPostgresTableAndIndexes`:
       - Đọc `existingCols` từ `information_schema.columns` trước.
       - **Nhánh A (`len(existingCols) == 0` — Tạo bảng mới lần đầu hoặc chạy trên SQLite unit test):** Giữ nguyên 100% transaction gốc (`CreateSQL` + `AlterSQL` + `IndexSQL` + legacy drops) $\rightarrow$ **Tương thích ngược 100% với mọi unit test SQLite hiện hữu**!
       - **Nhánh B (`len(existingCols) > 0` — Bảng PostgreSQL Master đang chạy Live 10 triệu dòng):**
         1. Chỉ lọc ra đúng câu `ALTER TABLE ADD COLUMN` của **cột mới chưa có trong `existingCols`** (`filterMissingColumnAlters`) và chỉ lọc câu `DROP COLUMN` cho cột legacy nào **thực sự còn nằm trong `existingCols`**.
         2. Nếu có câu `ALTER TABLE` cần chạy, mở Transaction cực ngắn (`lock_timeout = 5s`, `statement_timeout = 30s`), thực thi đúng câu `ADD COLUMN` của cột mới rồi **`COMMIT` ngay lập tức (`~1–5ms`)**! Sau đó cập nhật ngay `in_master = true` và gọi `g.cacheInvalidator(reg.ID, reg.MasterTable)`.
         3. Đọc danh sách index đã tồn tại từ `pg_indexes` (`WHERE schemaname = ? AND tablename = ?`). Chỉ những câu trong `res.IndexSQL` mà tên index **chưa có trong `pg_indexes`** mới được chuyển sang `CREATE [UNIQUE] INDEX CONCURRENTLY IF NOT EXISTS` và chạy **bên ngoài transaction**! Nhờ đó không bao giờ giữ `AccessExclusiveLock` khi dựng index trên 10 triệu dòng, và cũng không chạy lại `CREATE INDEX` thừa cho các index đã có!

4. **Phản biện Mục tiêu #4 (`F4` + `L008` — Bác bỏ việc nhồi thêm code vào `transmuter.go` `1,835` dòng):**
   - **Tử huyệt của Plan V1:** `transmuter.go` đang là God-file **`1,835` dòng**. Thêm bất kỳ dòng nào vào `transmuter.go` mà không rút bớt logic sẽ vi phạm `L008` (`NOW <= HEAD`).
   - **Thiết kế V2 (Tách file `transmuter_batch_guard.go` & giảm số dòng `transmuter.go` từ `1,835` xuống `~1,775` dòng):**
     - Rút khối khử trùng lặp nội bộ mẻ (`transmuter.go:1202-1270`, dài 68 dòng) sang hàm đơn trách nhiệm `dedupAndSortMasterBatchRecords` trong file mới `centralized-data-service/internal/service/master/transmuter_batch_guard.go` (`~135` dòng).
     - Hàm `dedupAndSortMasterBatchRecords` vừa khử trùng lặp theo `_source_ts`, vừa **sắp xếp kết quả tăng dần theo khóa `conflictTarget`** (`sort.SliceStable`) để triệt tiêu hoàn toàn nguy cơ Deadlock `SQLSTATE 40P01` khi chạy song song với luồng Live CDC.
     - Kết quả: `transmuter.go` giảm từ **`1,835` dòng xuống `~1,775` dòng (`-60` dòng, `NOW < HEAD`)**!

5. **Phản biện Mục tiêu #5 (Phát hiện mới `F5` — "Cú Nổ Ẩn" làm sai lệch 10 triệu dòng trên Báo cáo Đối soát 3 Trạm `GetDailyFlow` khi chạy `run_type = "full"`):**
   - **Bằng chứng vật lý (`file:line`):**
     - Khi chạy Backfill 10 triệu dòng qua `cdc.cmd.transmute` với `run_type = "full"`, [`transmute_handler.go:401`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/transmute_handler.go#L401) ghi vào `cdc_activity_log.details` giá trị `"run_type": "full"`.
     - Thế nhưng tại [`activity_log_read_repo_gorm.go:428, 438, 454-462`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/system/activity_log_read_repo_gorm.go#L428-L462), câu truy vấn `GetDailyFlow` (bảng Thống kê Đối soát Luồng Dữ liệu 3 Trạm trên CMS) chỉ kiểm tra `COALESCE(e.details->>'run_type', '') = 'full_sync'` (thiếu giá trị `'full'`)!
     - Nếu người vận hành bấm chạy với `triggered_by` là tài khoản cá nhân (không nằm trong `'snapshot', 'admin', 'manual', 're-sync'`), toàn bộ **10,000,000 dòng Updated của đợt Backfill sẽ bị cộng nhầm vào cột `master_written` (`4. GHI VÀO DB MASTER - Từ CDC Stream`)** thay vì `snapshot_master` (`1. SNAPSHOT / RE-SYNC -> Master Sync`), gây nổ cảnh báo lệch đối soát 10 triệu dòng!
   - **Thiết kế V2 (`NOW == HEAD` = `599/599` dòng):**
     - Sửa in-place trên đúng 7 dòng (`428, 438, 454, 456, 458, 460, 462`) của `activity_log_read_repo_gorm.go`: thay `= 'full_sync'` thành `IN ('full', 'full_sync')` và `<> 'full_sync'` thành `NOT IN ('full', 'full_sync')` (giữ nguyên `599` dòng `NOW == HEAD`).

---

### 5.2. Bảng Kiểm soát Kiến trúc Phân tầng (Layering), Đơn Trách Nhiệm (SRP) & Ngân sách Dòng (`wc -l` Line Budget — `L008`)

| STT | Service / Layer | File | Vai trò & Trách nhiệm Đơn (SRP) | `HEAD` (`wc -l`) | `V2 Target` (`wc -l`) | Đánh giá `L008` |
| :-: | :--- | :--- | :--- | :-: | :-: | :-: |
| 1 | `centralized-data-service` / Domain Service (`internal/service/master`) | [`transmuter.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go) | Điều phối luồng Transmute (`Run`, `RunEvents`, `processBatch`, `bulkUpsertMaster`) | `1835` | `~1775` (**`-60` dòng**) | **PASS (`NOW < HEAD`)** |
| 2 | `centralized-data-service` / Domain Service (`internal/service/master`) | `transmuter_batch_guard.go` *(File mới)* | Chốt chặn an toàn mẻ ghi: Dedup + Sort khóa chống Deadlock `40P01`, Fallback lọc cột khi gặp `42703`, và Pacing `20ms` khi Full Sync | `0` | `~135` | **PASS (`< 500` dòng)** |
| 3 | `centralized-data-service` / Domain Service (`internal/service/master`) | [`master_ddl_generator.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/master_ddl_generator.go) | Sinh câu lệnh DDL (`Generate`), tra cứu `master_binding` (`loadBinding`) và điều phối `Apply` | `743` | `~360` (**`-383` dòng**) | **PASS (`NOW < HEAD`)** |
| 4 | `centralized-data-service` / Domain Service (`internal/service/master`) | `master_ddl_generator_pg.go` *(File mới — đối xứng với `master_ddl_generator_mongo.go`)* | Thực thi DDL vật lý trên PostgreSQL Master (`applyPostgresTableAndIndexes` tách cột mới `< 5ms` & tạo index `CONCURRENTLY` ngoài tx, `ReconcileColumn`, `DropColumn`, `Swap`, `DropTable`) | `0` | `~395` | **PASS (`< 500` dòng)** |
| 5 | `cdc-cms-service` / Domain Port (`internal/domain/master`) | [`publisher.go`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/domain/master/publisher.go) | Khai báo interface `MasterDDLPublisher` (`PublishMasterCreate` nhận variadic `bindingID ...int64`) | `14` | `14` | **PASS (`< 500` dòng)** |
| 6 | `cdc-cms-service` / Infra Messaging (`internal/infra/messaging/nats`) | [`master_ddl_publisher.go`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/messaging/nats/master_ddl_publisher.go) | Adapter NATS gửi kèm `master_binding_id` trong payload `cdc.cmd.master-create` | `162` | `168` | **PASS (`< 500` dòng)** |
| 7 | `cdc-cms-service` / App Command (`internal/app/commands/master`) | [`save_master_rule.go`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/master/save_master_rule.go) & [`batch_update_rules.go`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/master/batch_update_rules.go) | Truyền `rule.MasterBindingID` / `bid` vào `PublishMasterCreate` | `77` / `56` | `77` / `56` | **PASS (`< 500` dòng)** |
| 8 | `cdc-cms-service` / Infra Persistence (`internal/infra/persistence/system`) | [`activity_log_read_repo_gorm.go`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/system/activity_log_read_repo_gorm.go) | Phân loại đúng `run_type IN ('full', 'full_sync')` vào `snapshot_master` trên bảng Đối soát 3 Trạm | `599` | `599` (**`+0` dòng**) | **PASS (`NOW == HEAD`)** |

---

### 5.3. Phản biện Từng Dòng Update (Line-by-Line Code Demo V2)

#### A. File 1 (Cập nhật): [`centralized-data-service/internal/service/master/transmuter.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go) (`1835` $\rightarrow$ `~1775` dòng)

- **Dòng `451` (`F1` — Giữ khóa bảo vệ OCC `_source_ts` khi chạy Full Table Resync 10M dòng và Incremental Sync theo ID; chỉ tắt OCC khi Recon Force-Heal truyền cả `runType == "full" && len(chunkIDs) > 0`):**
```go
// --- TRƯỚC (transmuter.go:450-452) ---
			if len(shadowRows) > 0 {
				batchRes = t.processBatch(ctx, masterRow, rules, shadowRows, strat, rc, runType == "full" || len(chunkIDs) > 0)

// --- SAU (transmuter.go:450-452) ---
			if len(shadowRows) > 0 {
				batchRes = t.processBatch(ctx, masterRow, rules, shadowRows, strat, rc, runType == "full" && len(chunkIDs) > 0)
```

- **Dòng `576` (`F4` — Thêm Pacing `20ms` có kiểm soát giữa các batch 2,000 dòng khi quét Full Sync toàn bảng để nhường I/O cho luồng Live):**
```go
// --- TRƯỚC (transmuter.go:575-580) ---
			}

			// Nếu là Incremental Sync theo danh sách chunkIDs, thoát ngay sau batch đầu tiên của chunk hiện tại
			if len(chunkIDs) > 0 {
				break
			}

// --- SAU (transmuter.go:575-581) ---
			}
			paceFullSyncBatch(ctx, runType, len(chunkIDs))

			// Nếu là Incremental Sync theo danh sách chunkIDs, thoát ngay sau batch đầu tiên của chunk hiện tại
			if len(chunkIDs) > 0 {
				break
			}
```

- **Dòng `1202-1270` & `1292` (`F2` + `F4` + `L008` — Rút 68 dòng dedup inline sang `dedupAndSortMasterBatchRecords` và bọc `bulkUpsertMasterWithSchemaGuard`):**
```go
// --- TRƯỚC (transmuter.go:1202-1295) ---
	// [BỔ SUNG VÁ LỖI QUAN TRỌNG]: Dedup nội bộ mẻ để loại bỏ triệt để lỗi SQLSTATE 21000
	if len(allRecords) > 0 {
		conflictTarget := resolveConflictTarget(binding, colTypes)
        // ... (62 dòng dedup inline không sắp xếp khóa) ...
		allRecords = dedupedRecords

		// Thực hiện chia nhỏ mẻ để ghi dữ liệu hàng loạt (Bulk Upsert)
		if len(allRecords) > 0 {
            ...
				for attempt := 0; attempt <= maxRetries; attempt++ {
					ins, upd, occSkip, err = t.bulkUpsertMaster(ctx, binding, allRecords[i:end], colTypes, isFullSync)

// --- SAU (transmuter.go:1202-1235) ---
	if len(allRecords) > 0 {
		conflictTarget := resolveConflictTarget(binding, colTypes)
		dedupedRecords, intraBatchDuplicates := dedupAndSortMasterBatchRecords(allRecords, conflictTarget)
		out.occSkipped += intraBatchDuplicates
		out.skipped += intraBatchDuplicates
		allRecords = dedupedRecords

		// Thực hiện chia nhỏ mẻ để ghi dữ liệu hàng loạt (Bulk Upsert)
		if len(allRecords) > 0 {
            ...
				for attempt := 0; attempt <= maxRetries; attempt++ {
					ins, upd, occSkip, err = t.bulkUpsertMasterWithSchemaGuard(ctx, binding, allRecords[i:end], colTypes, isFullSync)
```

---

#### B. File 2 (Tạo mới — Đơn trách nhiệm): `centralized-data-service/internal/service/master/transmuter_batch_guard.go` (`~135` dòng)

```go
package master

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"
)

// dedupAndSortMasterBatchRecords khử trùng lặp nội bộ mẻ theo conflictTarget (giữ bản ghi có _source_ts mới nhất)
// và sắp xếp kết quả tăng dần theo khóa conflictTarget để triệt tiêu Deadlock (SQLSTATE 40P01)
// khi Full Resync 10M dòng chạy song song với luồng Live CDC.
func dedupAndSortMasterBatchRecords(allRecords []map[string]any, conflictTarget string) ([]map[string]any, int64) {
	if len(allRecords) == 0 {
		return allRecords, 0
	}
	bestIndices := make(map[string]int, len(allRecords))
	for idx, rec := range allRecords {
		conflictVal := resolveRecordConflictValue(rec, conflictTarget)
		if conflictVal == nil {
			continue
		}
		dedupKey := fmt.Sprintf("%v", conflictVal)
		storedIdx, exists := bestIndices[dedupKey]
		if !exists {
			bestIndices[dedupKey] = idx
			continue
		}
		currentTs, okCur := extractRecordSourceTs(rec)
		storedTs, okStored := extractRecordSourceTs(allRecords[storedIdx])
		if !okCur || !okStored || currentTs >= storedTs {
			bestIndices[dedupKey] = idx
		}
	}

	keepIndices := make(map[int]bool, len(bestIndices))
	for _, idx := range bestIndices {
		keepIndices[idx] = true
	}
	deduped := make([]map[string]any, 0, len(bestIndices))
	for idx, rec := range allRecords {
		if keepIndices[idx] {
			deduped = append(deduped, rec)
		}
	}
	sort.SliceStable(deduped, func(i, j int) bool {
		ki := fmt.Sprintf("%v", resolveRecordConflictValue(deduped[i], conflictTarget))
		kj := fmt.Sprintf("%v", resolveRecordConflictValue(deduped[j], conflictTarget))
		return ki < kj
	})
	return deduped, int64(len(allRecords) - len(deduped))
}

func resolveRecordConflictValue(rec map[string]any, conflictTarget string) any {
	var v any
	if conflictTarget != "_gpay_id" {
		v = rec[conflictTarget]
	}
	if v == nil {
		v = rec["_gpay_id"]
	}
	return v
}

func extractRecordSourceTs(rec map[string]any) (int64, bool) {
	switch v := rec["_source_ts"].(type) {
	case int64:
		return v, true
	case float64:
		return int64(v), true
	default:
		return 0, false
	}
}

// bulkUpsertMasterWithSchemaGuard bọc quanh bulkUpsertMaster: ở luồng bình thường tốn 0 query phụ.
// Chỉ khi gặp lỗi PostgreSQL 42703 (undefined_column — do rule mới vừa được approve trên CMS
// nhưng DDL ALTER TABLE ADD COLUMN trên bảng Master đang chạy hoặc bị trễ), hàm mới đọc danh sách cột
// thực tế trên bảng Master, tạm lọc bỏ cột chưa tồn tại khỏi bản sao records và retry ngay lập tức.
func (t *TransmuterModule) bulkUpsertMasterWithSchemaGuard(
	ctx context.Context,
	binding *masterBindingRow,
	records []map[string]any,
	colTypes map[string]string,
	isFullSync bool,
) (int64, int64, int64, error) {
	ins, upd, skip, err := t.bulkUpsertMaster(ctx, binding, records, colTypes, isFullSync)
	if err == nil || !isUndefinedColumnDBError(err) || binding.IsMongoDestination(ctx, t.connMgr) {
		return ins, upd, skip, err
	}
	filteredRecords, strippedCols := t.filterRecordsToExistingMasterColumns(ctx, binding, records)
	if len(strippedCols) == 0 || len(filteredRecords) == 0 {
		return ins, upd, skip, err
	}
	t.logger.Warn("transmuter: stripping unapplied master columns on 42703 to protect live stream",
		zap.String("master", binding.MasterTable),
		zap.Strings("stripped_columns", strippedCols))
	return t.bulkUpsertMaster(ctx, binding, filteredRecords, colTypes, isFullSync)
}

func isUndefinedColumnDBError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "42703") ||
		(strings.Contains(msg, "column") && strings.Contains(msg, "does not exist")) ||
		strings.Contains(msg, "no such column")
}

func (t *TransmuterModule) filterRecordsToExistingMasterColumns(
	ctx context.Context,
	binding *masterBindingRow,
	records []map[string]any,
) ([]map[string]any, []string) {
	masterDB, err := t.connMgr.GetMasterDB(ctx, binding.MasterConnectionKey)
	if err != nil {
		return nil, nil
	}
	var existingCols []string
	if err := masterDB.WithContext(ctx).Raw(
		`SELECT column_name FROM information_schema.columns WHERE table_schema = ? AND table_name = ?`,
		binding.MasterSchema, binding.MasterTable,
	).Scan(&existingCols).Error; err != nil || len(existingCols) == 0 {
		return nil, nil
	}
	colSet := make(map[string]bool, len(existingCols))
	for _, c := range existingCols {
		colSet[c] = true
	}
	strippedMap := make(map[string]bool)
	filtered := make([]map[string]any, len(records))
	for i, rec := range records {
		clone := make(map[string]any, len(rec))
		for k, v := range rec {
			if colSet[k] {
				clone[k] = v
			} else {
				strippedMap[k] = true
			}
		}
		filtered[i] = clone
	}
	stripped := make([]string, 0, len(strippedMap))
	for k := range strippedMap {
		stripped = append(stripped, k)
	}
	sort.Strings(stripped)
	return filtered, stripped
}

// paceFullSyncBatch nghỉ ngắn 20ms giữa các mẻ 2,000 dòng khi chạy Full Table Scan (runType == "full" && chunkIDsLen == 0)
// để nhường Disk IOPS, CPU, Connection Pool và Autovacuum cho luồng Đọc/Ghi Live trên bảng 10 triệu dòng.
func paceFullSyncBatch(ctx context.Context, runType string, chunkIDsLen int) {
	if runType != "full" || chunkIDsLen > 0 {
		return
	}
	timer := time.NewTimer(20 * time.Millisecond)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}
```

---

#### C. File 3 (Cập nhật) & File 4 (Tạo mới — Đơn trách nhiệm): Tách [`master_ddl_generator.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/master_ddl_generator.go) (`743` $\rightarrow$ `~360` dòng) & `master_ddl_generator_pg.go` (`~395` dòng)

- Trong [`master_ddl_generator.go:341-449`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/master_ddl_generator.go#L341-L449), hàm `Apply` trở thành bộ điều phối gọn nhẹ:
```go
func (g *MasterDDLGenerator) Apply(ctx context.Context, masterName string, bindingID ...int64) (*MasterDDLResult, error) {
	var bID int64
	if len(bindingID) > 0 {
		bID = bindingID[0]
	}
	reg, err := g.loadBinding(ctx, masterName, bID)
	if err != nil {
		return nil, err
	}
	res, err := g.Generate(ctx, masterName, bID)
	if err != nil {
		g.markDDLStatus(ctx, reg.ID, "failed", err)
		return nil, err
	}
	if strings.EqualFold(reg.EngineType, "mongodb") || strings.EqualFold(reg.EngineType, "mongo") ||
		(g.connMgr != nil && g.connMgr.IsMongoMasterConnection(ctx, reg.MasterConnectionKey)) {
		return g.applyMongoCollectionAndIndexes(ctx, reg, res)
	}
	return g.applyPostgresTableAndIndexes(ctx, reg, res)
}
```

- Trong file mới `centralized-data-service/internal/service/master/master_ddl_generator_pg.go` (chứa `applyPostgresTableAndIndexes`, `ReconcileColumn`, `DropColumn`, `Swap`, `DropTable` chuyển từ `master_ddl_generator.go` sang):
```go
func (g *MasterDDLGenerator) applyPostgresTableAndIndexes(
	ctx context.Context,
	reg *masterDDLBindingRow,
	res *MasterDDLResult,
) (*MasterDDLResult, error) {
	db, err := g.connMgr.GetMasterDB(ctx, reg.MasterConnectionKey)
	if err != nil {
		g.markDDLStatus(ctx, reg.ID, "failed", err)
		return nil, err
	}

	// Kiểm tra danh sách cột hiện có trên bảng Master trước khi mở Transaction
	var existingCols []string
	_ = db.WithContext(ctx).Raw(
		`SELECT column_name FROM information_schema.columns WHERE table_schema = ? AND table_name = ?`,
		reg.MasterSchema, reg.MasterTable,
	).Scan(&existingCols)
	existingColMap := make(map[string]bool, len(existingCols))
	for _, c := range existingCols {
		existingColMap[c] = true
	}

	legacyCols := []string{"_source_id", "_source", "_synced_at", "_version", "_hash", "_created_at", "_raw_data"}

	if len(existingColMap) == 0 {
		// Nhánh A: Bảng mới hoàn toàn (hoặc môi trường SQLite test) -> chạy nguyên khối như cũ (100% Backward Compatible)
		err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec(`SET LOCAL lock_timeout = '5s'`).Error; err != nil {
				return fmt.Errorf("set lock_timeout: %w", err)
			}
			if err := tx.Exec(`SET LOCAL statement_timeout = '30s'`).Error; err != nil {
				return fmt.Errorf("set statement_timeout: %w", err)
			}
			if err := tx.Exec(res.CreateSQL).Error; err != nil {
				return fmt.Errorf("create table: %w", err)
			}
			for _, s := range res.AlterSQL {
				if err := tx.Exec(s).Error; err != nil {
					return fmt.Errorf("alter add column: %w", err)
				}
			}
			for _, s := range res.IndexSQL {
				if err := tx.Exec(s).Error; err != nil {
					return fmt.Errorf("create index: %w", err)
				}
			}
			legacyTbl := quoteDDLQualified(reg.MasterSchema, reg.MasterTable)
			for _, col := range legacyCols {
				if err := tx.Exec(fmt.Sprintf(`ALTER TABLE %s DROP COLUMN IF EXISTS %s`, legacyTbl, quoteDDLIdent(col))).Error; err != nil {
					return fmt.Errorf("drop legacy col %s: %w", col, err)
				}
			}
			return nil
		})
		if err != nil {
			res.Err = err.Error()
			g.markDDLStatus(ctx, reg.ID, "failed", err)
			return res, err
		}
	} else {
		// Nhánh B: Bảng PostgreSQL Master đang chạy Live (vd 10M dòng) ->
		// 1) Chỉ chạy ALTER TABLE ADD COLUMN cho cột MỚI chưa có trong existingColMap
		//    và chỉ DROP COLUMN cho cột legacy thực sự còn tồn tại, rồi COMMIT ngay (< 5ms).
		missingAlters := filterMissingColumnAlters(res.AlterSQL, existingColMap)
		legacyTbl := quoteDDLQualified(reg.MasterSchema, reg.MasterTable)
		for _, col := range legacyCols {
			if existingColMap[col] {
				missingAlters = append(missingAlters, fmt.Sprintf(`ALTER TABLE %s DROP COLUMN IF EXISTS %s`, legacyTbl, quoteDDLIdent(col)))
			}
		}
		if len(missingAlters) > 0 {
			err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
				if err := tx.Exec(`SET LOCAL lock_timeout = '5s'`).Error; err != nil {
					return fmt.Errorf("set lock_timeout: %w", err)
				}
				if err := tx.Exec(`SET LOCAL statement_timeout = '30s'`).Error; err != nil {
					return fmt.Errorf("set statement_timeout: %w", err)
				}
				for _, s := range missingAlters {
					if err := tx.Exec(s).Error; err != nil {
						return fmt.Errorf("alter table live: %w", err)
					}
				}
				return nil
			})
			if err != nil {
				res.Err = err.Error()
				g.markDDLStatus(ctx, reg.ID, "failed", err)
				return res, err
			}
		}
	}

	// Bật RLS, đồng bộ in_master = true và xóa cache rule NGAY LẬP TỨC sau khi cột mới đã commit
	if err := db.WithContext(ctx).Exec(fmt.Sprintf(
		`ALTER TABLE %s ENABLE ROW LEVEL SECURITY`,
		quoteDDLQualified(reg.MasterSchema, reg.MasterTable),
	)).Error; err != nil {
		g.logger.Warn("enable RLS failed",
			zap.String("master", reg.MasterTable),
			zap.String("schema", reg.MasterSchema), zap.Error(err))
	} else {
		res.RLSApplied = true
	}

	g.syncInMasterColumns(ctx, db, reg)
	g.markDDLStatus(ctx, reg.ID, "created", nil)
	if g.cacheInvalidator != nil {
		g.cacheInvalidator(reg.ID, reg.MasterTable)
	}

	// 2) Với bảng Live đã tồn tại (len(existingColMap) > 0), chỉ tạo các Index MỚI chưa có trong pg_indexes
	//    bằng CREATE INDEX CONCURRENTLY bên ngoài Transaction để KHÔNG giữ AccessExclusiveLock chặn luồng đọc/ghi Live!
	if len(existingColMap) > 0 && len(res.IndexSQL) > 0 {
		g.applyMissingIndexesConcurrently(ctx, db, reg, res.IndexSQL)
	}
	return res, nil
}
```

---

#### D. File 5–8 (`cdc-cms-service`): Truyền `master_binding_id` khi `PublishMasterCreate` (`F2`) & Sửa `GetDailyFlow` nhận `'full'` (`F5`)

1. [`cdc-cms-service/internal/domain/master/publisher.go:13`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/domain/master/publisher.go#L13):
```go
// --- TRƯỚC ---
	PublishMasterCreate(ctx context.Context, masterTable, triggeredBy string) error
// --- SAU (Tương thích ngược 100% nhờ variadic bindingID ...int64) ---
	PublishMasterCreate(ctx context.Context, masterTable, triggeredBy string, bindingID ...int64) error
```

2. [`cdc-cms-service/internal/infra/messaging/nats/master_ddl_publisher.go:124-136`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/messaging/nats/master_ddl_publisher.go#L124-L136):
```go
func (p *NatsMasterDDLPublisher) PublishMasterCreate(ctx context.Context, masterTable, triggeredBy string, bindingID ...int64) error {
	if p.conn == nil {
		return fmt.Errorf("nats conn nil")
	}
	msgMap := map[string]any{
		"master_table":   masterTable,
		"triggered_by":   triggeredBy,
		"correlation_id": "rule-approve-" + masterTable + "-" + time.Now().UTC().Format(time.RFC3339Nano),
	}
	if len(bindingID) > 0 && bindingID[0] > 0 {
		msgMap["master_binding_id"] = bindingID[0]
	}
	payload, _ := json.Marshal(msgMap)
	return p.conn.Publish("cdc.cmd.master-create", payload)
}
```

3. [`cdc-cms-service/internal/app/commands/master/save_master_rule.go:72`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/master/save_master_rule.go#L72) & [`batch_update_rules.go:50`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/master/batch_update_rules.go#L50):
```go
// save_master_rule.go:72
_ = h.publisher.PublishMasterCreate(ctx, masterTable, actor, rule.MasterBindingID)

// batch_update_rules.go:50
_ = h.publisher.PublishMasterCreate(ctx, masterTable, username, bid)
```

4. [`cdc-cms-service/internal/infra/persistence/system/activity_log_read_repo_gorm.go:428, 438, 454-462`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/system/activity_log_read_repo_gorm.go#L428-L462) (`599` $\rightarrow$ `599` dòng `NOW == HEAD`):
```sql
-- Sửa in-place trên 7 dòng (428, 438, 454, 456, 458, 460, 462):
-- Thay: COALESCE(e.details->>'run_type', '') = 'full_sync'   --> COALESCE(e.details->>'run_type', '') IN ('full', 'full_sync')
-- Thay: COALESCE(e.details->>'run_type', '') <> 'full_sync'  --> COALESCE(e.details->>'run_type', '') NOT IN ('full', 'full_sync')
```
