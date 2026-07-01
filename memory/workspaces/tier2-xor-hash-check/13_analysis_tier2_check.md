# Phân Tích Kỹ Thuết Chi Tiết: Luồng Đối Soát Tier 2 XOR-Hash

Tài liệu này ghi lại kết quả phân tích mã nguồn chi tiết về cơ chế đối soát Tier 2 (cửa sổ thời gian dựa trên XOR-hash) của hệ thống `centralized-data-service`.

---

## 1. Cơ Chế Hoạt Động của Thuật Toán XOR-Hash (Tier 2)

Đối soát dữ liệu quy mô lớn đòi hỏi kiểm tra sự đồng nhất giữa hai cơ sở dữ liệu (Source & Destination) một cách nhanh chóng mà không gây quá tải mạng và tài nguyên máy chủ. Tier 2 giải quyết bài toán này bằng thuật toán **cửa sổ thời gian trượt tích lũy XOR-hash**.

### Phép Toán XOR và Tính Chất Giao Hoán/Kết Hợp
Thuật toán tính toán mã băm cho từng bản ghi bằng hàm `hashIDPlusTsMs(ID, Timestamp)`. Sau đó, tất cả các mã băm trong một cửa sổ thời gian trượt `[lo, hi)` được tích lũy bằng phép toán XOR (`^`):
$$\text{XOR\_Hash} = \text{hash}(ID_1, TS_1) \oplus \text{hash}(ID_2, TS_2) \oplus \dots \oplus \text{hash}(ID_n, TS_n)$$

* **Tính chất giao hoán (Commutative)**: $A \oplus B = B \oplus A$
* **Tính chất kết hợp (Associative)**: $(A \oplus B) \oplus C = A \oplus (B \oplus C)$
* **Ý nghĩa thực tiễn**: Nhờ hai tính chất này, thứ tự đọc các dòng dữ liệu từ cơ sở dữ liệu không làm ảnh hưởng đến mã băm tích lũy cuối cùng của cửa sổ. Điều này giúp:
  1. Loại bỏ mệnh đề `ORDER BY` trong các câu lệnh truy vấn SQL.
  2. Cho phép tính toán song song hoặc stream dữ liệu mà không cần sắp xếp trước, giảm đáng kể tải CPU/RAM trên cả database và ứng dụng.
* **Tính tự nghịch đảo (Self-inverse)**: $A \oplus A = 0$. Tính chất này được kiểm chứng qua các bài test (`TestXORSelfInverse` tại `recon_hash_test.go`), giúp phát hiện các bản ghi bị lệch một cách chính xác.

---

## 2. Truy Vết Luồng Kiểm Soát (Control Flow Tracing)

Luồng thực thi của hàm `RunTier2` tại [recon_tier_a.go:L846](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_tier_a.go#L846) diễn ra như sau:

```mermaid
graph TD
    A[Bắt đầu RunTier2] --> B[withTableLock: Khóa bảng chống chạy trùng lặp]
    B --> C[beginRun: Khởi tạo handle đối soát Tier 2]
    C --> D[pickScanRangeWithLag: Lấy khoảng [lo, hi] an toàn dựa trên replication lag]
    D --> E[buildWindows: Chia nhỏ khoảng thành các cửa sổ trượt]
    E --> F[Vòng lặp qua từng Window]
    F --> G[sourceAgent.HashWindow: Lấy Count & XorHash đầu Source]
    F --> H[destAgent.HashWindow: Lấy Count & XorHash đầu Dest]
    G --> I{srcRes == dstRes?}
    H --> I
    I -- Đúng --> J[Tiếp tục sang window tiếp theo]
    I -- Sai --> K[Drifted! Gọi ListIDTsInWindow cả 2 bên]
    K --> L[diffIDTsSegmentA: Tìm concrete ID lệch/thiếu]
    L --> M[Gộp danh sách mismatches]
    J --> N{Hết các window?}
    M --> N
    N -- Đúng --> O[Post-processing: Kiểm tra existence trực tiếp trong Shadow DB]
    O --> P[stampA & finishRun: Tạo report và giải phóng khóa]
```

### Chi tiết các bước:
1. **Khóa bảng (`withTableLock`)**: Tránh trường hợp có 2 luồng đối soát Tier 2 cùng chạy đồng thời trên một table registry.
2. **Xác định Scan Range**: dynamic watermarks được xác định thông qua `pickScanRangeWithLag`, tự động trừ đi thời gian lag thích ứng (`adaptiveFreeze`) để đảm bảo không đối soát các dữ liệu realtime đang được đồng bộ, tránh phát hiện sai lệch giả (false drift).
3. **Đối chiếu XOR-Hash**:
   - `srcRes, err := rc.sourceAgent.HashWindow(...)`
   - `dstRes, err := rc.destAgent.HashWindow(...)`
   - Nếu `Count` và `XorHash` của hai bên trùng khớp hoàn hảo -> Window đó đồng nhất dữ liệu, chuyển sang window tiếp theo.
4. **Trích xuất sự sai lệch (Drill-down)**:
   - Nếu phát hiện lệch hash/count, hệ thống gọi `ListIDTsInWindow` ở cả Source và Destination để tải về danh sách cụ thể các cặp `(ID, Timestamp)`.
   - Hàm `diffIDTsSegmentA` đối chiếu hai danh sách này để phân loại ra:
     - `missingFromDest`: Có ở Source nhưng thiếu ở Destination.
     - `missingFromSrc` (Orphan): Có ở Destination nhưng không có ở Source.
     - `mismatched`: Trùng ID nhưng lệch Timestamp (dữ liệu cũ/mới bị lệch).
5. **Đồng bộ hóa biên (Post-processing Cross-Check)**:
   - Những ID thuộc `missingFromDest` được truy vấn trực tiếp lại trong Shadow DB bằng câu lệnh `SELECT ... WHERE pk IN (?)` theo chunk 1000 để kiểm tra thực tế có tồn tại hay không. Nếu có tồn tại (do lệch biên cửa sổ thời gian) -> chuyển sang trạng thái `mismatched` để nâng cao độ chính xác.

---

## 3. Xác Minh Tính Chất Chỉ Đọc (Read-Only Verification)

Một yêu cầu bắt buộc đối với luồng đối soát Tier 2 là **không được ghi dữ liệu dưới mọi hình thức**. Chúng tôi đã xác minh điều này ở cả 3 tầng:

### Tầng 1: Cô lập Transaction ở Cơ sở dữ liệu (PostgreSQL replica)
Mọi truy vấn từ phía destination agent (`ReconDestAgent`) đều đi qua helper `readOnlyDB(ctx)` ([recon_dest_agent.go:L62](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_dest_agent.go#L62)):
```go
func (da *ReconDestAgent) readOnlyDB(ctx context.Context) *gorm.DB {
	tx := da.replica.WithContext(ctx).Begin()
	tx.Exec("SET TRANSACTION READ ONLY")
	return tx
}
```
* **Bảo vệ phần cứng**: Lệnh `SET TRANSACTION READ ONLY` thiết lập quyền chỉ đọc cho transaction này trong PostgreSQL. Nếu có bất kỳ lệnh sửa đổi dữ liệu nào (`INSERT`, `UPDATE`, `DELETE`) vô tình hay cố ý được thực thi, database engine sẽ lập tức báo lỗi và chặn đứng.
* **Không Commit**: Tất cả các hàm gọi `readOnlyDB` đều sử dụng cấu trúc:
  ```go
  tx := da.readOnlyDB(ctx)
  defer tx.Rollback()
  ```
  Rollback transaction đảm bảo các thay đổi tạm thời (nếu có ở tầng application) sẽ không bao giờ được ghi xuống ổ đĩa, giải phóng kết nối một cách an toàn.

### Tầng 2: Truy vấn Phía Source Agent
* **MongoDB Source** ([recon_source_agent.go:L184](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_source_agent.go#L184)):
  - Chỉ sử dụng phương thức `Find` chỉ đọc.
  - Sử dụng read preference là `Secondary` (`options.Collection().SetReadPreference(readpref.Secondary())`), đảm bảo chỉ truy vấn các bản sao MongoDB secondary, loại bỏ tải cho primary cluster và cam kết không ghi dữ liệu.
* **PostgreSQL Source** ([recon_hash.go:L258](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_hash.go#L258)):
  - Sử dụng câu lệnh `SELECT ... FROM ... WHERE ...` phẳng để stream các dòng dữ liệu, không sử dụng các thủ tục hay câu lệnh có thể thay đổi dữ liệu.

### Tầng 3: Tầng logic ứng dụng
- Không có bất kỳ logic heal dữ liệu hay lời gọi API ghi nào được nhúng bên trong `RunTier2`.
- Kết quả đối soát sai lệch chỉ được tổng hợp và ghi nhận vào báo cáo `recon.ReconciliationReport` và đẩy ra Prometheus gauge metrics (`metrics.ReconDrift`) để phục vụ cảnh báo giám sát. Luồng Healing (sửa đổi đồng bộ dữ liệu) là một tiến trình hoàn toàn tách biệt, được gọi độc lập bằng tay hoặc qua admin controller chứ không tự động trigger.

---

## 4. Kết Luận
Thuật toán Tier 2 window-based XOR-hash được thiết kế cực kỳ tối ưu, an toàn, sử dụng tài nguyên hiệu quả và tuân thủ nghiêm ngặt nguyên lý **Strictly Read-Only** ở cả cấp độ ứng dụng lẫn cơ sở dữ liệu.

---

## 5. Phân Tích Chuyên Sâu: Cơ Chế Khóa Bảng (withTableLock) Dưới Quy Quy Mô 200 Bảng * 50 Triệu Records

### 1. Khóa bảng nào? Bản chất của Khóa là gì?
- Hàm `withTableLock` tạo khóa bằng cách băm chuỗi `"recon_" + table` thành một số nguyên `int64` thông qua thuật toán CRC32 Checksum (`crc32.ChecksumIEEE`).
- Sau đó, nó thực hiện gọi PostgreSQL Advisory Lock ở mức session:
  ```sql
  SELECT pg_try_advisory_lock($1)
  ```
- **Bản chất**: Đây hoàn toàn là một **khóa logic/ứng dụng (application-level cooperative lock)**, không phải là khóa vật lý trên bảng dữ liệu (không có các lệnh lock table của Postgres như row share/access exclusive).
- **Kết luận**: Khóa này **KHÔNG block** bất kỳ luồng ghi/đọc dữ liệu chính nào từ hệ thống nghiệp vụ (luồng core transaction vẫn ghi INSERT/UPDATE bình thường). Nó chỉ có tác dụng ngăn cản 2 luồng đối soát Tier 2 (hoặc luồng đối soát và luồng heal) chạy trùng lặp trên cùng một bảng logic tại cùng một thời điểm.

### 2. Rủi ro & Tác động dưới quy mô lớn (200 bảng * 50tr records)
Khi hệ thống có số lượng bảng lớn (200 bảng) và lượng dữ liệu khổng lồ (50 triệu dòng/bảng), cơ chế khóa Advisory Lock này đối mặt với 2 rủi ro kỹ thuật nghiêm trọng:
1. **Cạn kiệt Connection Pool của Database (Connection Starvation)**:
   - Cơ chế Advisory Lock ở mức session yêu cầu ứng dụng phải ghim (pin) duy nhất một kết nối DB liên tục (`conn, err := sqlDB.Conn(ctx)`) để giữ lock cho đến khi đối soát bảng hoàn tất.
   - Với bảng 50 triệu record, thời gian đối soát qua các window có thể kéo dài (từ vài chục giây đến vài phút nếu có sai lệch lớn cần list ID).
   - Nếu chạy đối soát song song cho nhiều bảng (ví dụ 10-20 bảng cùng lúc), hệ thống sẽ chiếm dụng và ghim cứng 10-20 slots connection. Nếu connection pool tối đa của service nhỏ (ví dụ 50 hoặc 100), điều này có thể dẫn đến việc cạn kiệt connection pool, làm treo các API nghiệp vụ chính hoặc các tiến trình worker khác đang chờ kết nối trống.
2. **Xác suất đụng độ khóa (CRC32 Lock Key Collision)**:
   - Hàm `advisoryLockKey` sử dụng CRC32 Checksum (32-bit uint32 cast sang int64). Với 200 bảng, tuy xác suất đụng độ lý thuyết là rất nhỏ nhưng vẫn tồn tại khả năng 2 bảng khác tên băm ra cùng một ID khóa. Khi đó, bảng này đang chạy đối soát sẽ khiến bảng kia bị skip nhầm (vì nghĩ rằng bảng đó đang được đối soát).

### 3. Đề xuất Khắc phục & Tối ưu hóa
- **Giới hạn số luồng chạy song song (Concurrency Limit)**: Tuyệt đối không chạy đối soát song song cả 200 bảng cùng một lúc. Cần điều tiết bằng hàng đợi (queue) hoặc worker pool để giới hạn tối đa 3-5 bảng được đối soát đồng thời nhằm bảo toàn connection pool.
- **Chuyển dịch sang Distributed Lock (Redis Lock)**: Thay vì ghim connection DB để giữ Postgres Advisory Lock, khuyến nghị chuyển sang sử dụng Redis Distributed Lock (sử dụng cluster Redis hiện có qua `rc.redis`). Cách tiếp cận này giúp giải phóng hoàn toàn connection DB bị ghim, tăng khả năng chịu tải và khả năng mở rộng (scalability) của hệ thống lên vô hạn.

---

## 6. Xác Thực Việc Ánh Xạ ID và Timestamp (ID & Timestamp Mapping)

Chúng tôi đã tiến hành rà soát chi tiết cách thức hệ thống phân giải và ánh xạ cặp thuộc tính `(ID, Timestamp)` ở cả hai đầu Source và Destination đối với cấu hình shadow mới (ID = PK Field, Timestamp = Timestamp Field):

### 1. Phía Source (MongoDB Source)
- **Cột ID**: MongoDB luôn sử dụng khóa chính là trường `_id`. Hàm `extractMongoID` ([recon_hash.go:L191](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_hash.go#L191)) tự động kiểm tra:
  - Nếu là kiểu `ObjectID`, hàm convert sang chuỗi Hex thông qua `oid.Hex()`.
  - Nếu là kiểu khác, chuyển sang string bằng `fmt.Sprintf("%v", v)`.
- **Cột Timestamp**: Được cấu hình thông qua `TimestampField` trong TableRegistry (ví dụ: `lastUpdatedAt`).
  - Hàm `extractTimestampMs` ([recon_hash.go:L198](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_hash.go#L198)) tự động đọc trường cấu hình này và convert sang Epoch Milliseconds (`int64`).
  - **Cơ chế Fallback thông minh**: Nếu document bị thiếu trường timestamp này (hoặc nil), và ID là một ObjectID hợp lệ, hệ thống sẽ tự động trích xuất timestamp từ 4 byte đầu tiên của ObjectID đó. Việc này đảm bảo tính toàn vẹn và không gây nghẽn (unblock) cho quá trình băm XOR.

### 2. Phía Destination (Shadow DB PostgreSQL)
- **Cột ID**: Được truyền động từ `entry.PrimaryKeyField` của TableRegistry (ví dụ: `_id` hoặc cột PK thật).
  - Lệnh SQL đối soát: `SELECT %s::text AS id` -> Bắt buộc ép kiểu sang chuỗi văn bản (`text`). Điều này đảm bảo kết quả scan ở Go luôn khớp byte-for-byte với chuỗi hex ID ở phía MongoDB Source.
- **Cột Timestamp**: Được phân giải động qua hàm `resolveSourceTSField` ([recon_tier_a.go:L214](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_tier_a.go#L214)):
  - **Phân giải kiểu chữ (Camel vs Snake Case)**: MongoDB sử dụng camelCase (`lastUpdatedAt`) nhưng shadow table trong Postgres thường được CDC sync sang dạng snake_case (`last_updated_at`). 
  - Hàm `resolveSourceTSField` tự động băm variant snake_case của cột timestamp, thử kiểm tra sự tồn tại thực tế của cột trong Shadow DB bằng `ColumnExists` theo thứ tự ưu tiên: `lastUpdatedAt` -> `last_updated_at` -> candidates khác.
  - Cột thực tế tồn tại trong DB vật lý sẽ được chọn làm `ts_field`. Nếu không cột nào khớp, hệ thống fallback về `_source_ts`.
  - Lệnh SQL đối soát sử dụng đúng cột đã được phân giải này bọc trong nháy kép qua `quoteIdent(tsCol)`.
  - Giá trị timestamp trong Postgres (dạng `TIMESTAMP` hoặc `TIMESTAMPTZ`) được scan và tự động chuyển đổi sang Epoch Milliseconds bằng `sourceTs.UnixMilli()`.

### 3. Nguyên Nhân Gốc Rễ Phát Hiện Mismatched Giả (Timezone Skew)
Sau khi kiểm tra sâu mã nguồn, chúng tôi phát hiện lỗi logic nghiêm trọng dẫn đến việc đối soát báo **mismatched giả** (báo lệch timestamp mặc dù dữ liệu khớp 100%):
1. **MongoDB Source**: Lưu thời gian dạng UTC DateTime chuẩn. Hàm `extractTimestampMs` lấy chính xác giá trị Epoch Milliseconds tính theo UTC.
2. **Postgres Destination (Shadow DB)**: Cột timestamp (ví dụ `last_updated_at` hoặc `lastUpdatedAt`) có kiểu dữ liệu là `TIMESTAMP` (without time zone). Debezium đồng bộ dữ liệu thô dạng UTC vào cột này.
3. **Lỗi scan driver (GORM/pgx)**:
   - Khi hàm `ListIDTsInWindow` ([recon_dest_query.go:L360](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_dest_query.go#L360)) và `HashWindow` scan giá trị timestamp không có múi giờ này vào biến Go kiểu `time.Time`:
     ```go
     var tsVal *time.Time
     rows.Scan(&it.ID, &tsVal)
     ```
   - Mặc định, Go runtime driver (pgx) sẽ tự động áp múi giờ **`time.Local` của hệ điều hành hiện tại** (đang là múi giờ **+07:00** của hệ thống chạy cdc-worker) lên đối tượng `time.Time` đó.
   - Ví dụ: Dữ liệu thô trong DB là `2026-07-01 08:29:13.459` (thực chất là UTC). Driver scan ra đối tượng Go `2026-07-01 08:29:13.459 +0700` (hiểu nhầm là giờ Việt Nam).
   - Khi gọi `tsVal.UnixMilli()`, Go tự động trừ đi 7 tiếng để convert về UTC Epoch Ms, khiến giá trị trả về bị **hụt đi đúng 7 tiếng (25,200,000 ms)** so với MongoDB Source.
4. **Hậu quả**:
   - XOR-hash của cửa sổ bị lệch (do Epoch Ms lệch).
   - Khi drill-down đối chiếu trong `diffIDTsSegmentA` ([recon_tier_a.go:L1147](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_tier_a.go#L1147)):
     ```go
     } else if (dstTs / 1000) != (it.Ts / 1000) {
         mismatched = append(mismatched, it.ID)
     }
     ```
     Hai thương số này lệch nhau đúng `25200` đơn vị. Do đó, hệ thống **báo mismatched giả cho toàn bộ các bản ghi trong window**, mặc dù dữ liệu trong DB khớp nhau hoàn toàn!

### 4. Đề Xuất Giải Pháp Sửa Đổi Kỹ Thuật (Remediation)
Để triệt tiêu hoàn toàn mismatched giả do lệch múi giờ, cần thực hiện một trong các cách sau:
- **Cách 1 (Khuyên Dùng)**: Ép kiểu timezone sang UTC trực tiếp trong truy vấn SQL ở đầu Postgres:
  ```sql
  SELECT id, last_updated_at AT TIME ZONE 'UTC' AS ts ...
  ```
  Hoặc ép kiểu sang Epoch Milliseconds trực tiếp từ SQL Postgres bằng phép toán:
  ```sql
  SELECT id, EXTRACT(EPOCH FROM last_updated_at)*1000 AS ts ...
  ```
  Điều này giúp trả về một số nguyên Epoch Ms chuẩn UTC, tránh việc scan qua `time.Time` của driver.
- **Cách 2**: Chuyển đổi timezone thủ công trong Go code:
  ```go
  // Nếu biết chắc dữ liệu thô trong Postgres là UTC
  utcTime := time.Date(tsVal.Year(), tsVal.Month(), tsVal.Day(), tsVal.Hour(), tsVal.Minute(), tsVal.Second(), tsVal.Nanosecond(), time.UTC)
  it.Ts = utcTime.UnixMilli()
  ```

---

## 7. Phân Tích Kết Quả Đầu Ra & Luồng Chữa Lành Dữ Liệu (Output & Healing Flow Audit)

Sau khi hoàn thành tiến trình đối soát Tier 2 (Segment A) hoặc Segment B, kết quả được lưu giữ và xử lý thông qua cơ chế chữa lành (heal) đồng bộ như sau:

### 1. Kết quả đối soát được lưu ở đâu?
- Kết quả đối soát được đóng gói vào đối tượng struct `ReconciliationReport` và lưu trữ trực tiếp vào cơ sở dữ liệu PostgreSQL của dịch vụ tại bảng:
  **`cdc_system.cdc_reconciliation_report`**
- **Cấu trúc dữ liệu báo cáo**:
  - `missing_ids` (JSONB): Chứa danh sách các ID thực sự bị thiếu ở đầu Destination (Shadow DB).
  - `stale_ids` (JSONB): Chứa cấu trúc gom 3 loại ID: `missing_from_dest` (thiếu ở shadow), `missing_from_src` (dư thừa ở shadow / orphans), và `mismatched` (trùng ID nhưng lệch timestamp).
  - Các trường đếm số lượng: `missing_count`, `stale_count`, `orphan_count`.
  - Trạng thái `status`: `"drift"` (nếu phát hiện sai lệch) hoặc `"ok"` (nếu khớp hoàn hảo).

### 2. Luồng Chữa Lành Segment A (Source -> Shadow DB)
Khi tiến hành trigger heal cho Segment A (qua NATs cmd `cdc.cmd.heal`), hàm `healSegmentA` ([recon_heal_v4.go:L273](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_heal_v4.go#L273)) thực hiện:
1. **Fresh Scan**: Chạy đối soát Tier 2 tươi mới với `cold_lookback=true` để lấy báo cáo drift mới nhất.
2. **Gom ID sai lệch**: Hệ thống gộp tất cả các ID từ `missing_ids`, `mismatched` và `missing_from_src`.
3. **Trigger CDC Re-sync (Debezium Signal path)**:
   - Hệ thống chia nhỏ danh sách ID thành các chunk (kích thước 1000) và bắn tín hiệu re-sync qua NATS topic `"cdc.cmd.debezium-signal"`:
     ```json
     {
       "type": "incremental",
       "table": "<table_name>",
       "filter": "<snapshot_id_filter>",
       "action": "recon-heal-a",
       "origin": "recon"
     }
     ```
   - Debezium connector tiêu thụ signal này, tự động kích hoạt **Incremental Snapshot** trên danh sách ID lỗi, kéo dữ liệu mới nhất từ MongoDB/Postgres Source và đẩy sự kiện CDC mới vào Kafka topic. Dữ liệu sẽ chảy qua toàn bộ pipeline CDC chuẩn (masking/mapping) để cập nhật đè xuống Shadow DB.
4. **Safety Net & Direct Write (FetchAndWriteByIDs)**:
   - Nếu Tier 2 báo sạch drift, hệ thống chạy thêm Full ID diff để quét toàn bộ bảng.
   - Nếu phát hiện IDs bị thiếu nằm ngoài cửa sổ thời gian (7 ngày), hệ thống gọi **Direct path**: fetch trực tiếp từ MongoDB Source và thực hiện **OCC upsert** thẳng vào Shadow DB bằng `applyOne` -> gọi `UpsertRecord`.
   - Cơ chế OCC với mệnh đề `WHERE` kiểm soát chặt chẽ timestamp `_source_ts`, đảm bảo chỉ ghi đè khi dữ liệu mới hơn, bảo vệ dữ liệu realtime không bị ghi đè bởi dữ liệu cũ.

### 3. Luồng Chữa Lành Segment B (Shadow DB -> Master DB)
Khi trigger heal cho Segment B (qua NATs cmd `cdc.cmd.heal` với segment B), hàm `healSegmentB` ([recon_heal_v4.go:L102](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_heal_v4.go#L102)) thực hiện:
1. **Fresh Segment B Scan**: Chạy kiểm tra Segment B tươi mới.
2. **Map ID**: Hệ thống lấy danh sách `_gpay_id` sai lệch, truy vấn shadow table để map ngược lại thành `_source_id`.
3. **Trigger Reprocess (Transmute path)**:
   - Bắn yêu cầu re-transmute cho danh sách `sourceIDs` qua NATS topic `"cdc.cmd.transmute"`:
     ```json
     {
       "master_table": "<table_name>",
       "_source_ids": ["<id_1>", "<id_2>", ...],
       "triggered_by": "recon-heal-b"
     }
     ```
   - Pipeline Transmute sẽ đọc dữ liệu từ Shadow DB, thực hiện các logic biến đổi nghiệp vụ và ghi đè xuống Master DB, có OCC bảo vệ để tránh xung đột dữ liệu realtime.

---

## 8. Luồng Trigger từ Giao Diện Quản Trị & Giải Thích Hành Vi Chạy Thực Tế (User Flow & Live Heal Behavior)

Dưới đây là phân tích chi tiết về luồng trải nghiệm của người dùng trên giao diện quản trị (CMS-Web) và giải thích kỹ thuật cho hiện tượng chuyển đổi trạng thái báo cáo giữa hai lần chạy heal của bạn:

### 1. Luồng Người Dùng trên CMS-Web (User Flow)
- **Đọc trạng thái**: CMS-Web hoặc API Gateway liên tục đọc bảng `cdc_system.cdc_reconciliation_report`.
- **Hiển thị nút Heal**: Nếu báo cáo của một bảng có trạng thái `status = 'drift'`, giao diện CMS-Web sẽ tự động hiển thị nút **Chữa lành (Heal)** cho bảng đó.
- **Trigger hành động**: Khi người dùng click nút Heal:
  - CMS-Web gửi request qua admin API gateway.
  - API gateway bắn một tin nhắn bất đồng bộ qua **NATS broker** đến chủ đề (subject) `cdc.cmd.heal` với payload định dạng:
    ```json
    {
      "table": "<tên_bảng>",
      "segment": "source_shadow",
      "legacy": false
    }
    ```
  - Dịch vụ `cdc-worker` (thông qua `ReconHandler.HandleReconHeal` tại [recon_handler_run.go:L200](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_handler_run.go#L200)) nhận tin nhắn và điều phối chạy luồng heal tương ứng.

### 2. Giải thích Hiện tượng Chạy Heal Thực Tế (Phân tích So sánh Lần 1 và Lần 2)

#### 🔸 Lần 1: Chạy Heal từ Report ID 7 (`status = 'drift'`)
- **Bối cảnh đối soát**: Report 7 phát hiện 1 bản ghi bị thiếu (`missing_from_dest`) và 3 bản ghi bị lệch timestamp (`mismatched`) trong cửa sổ quét.
- **Thực thi heal**: Healer gộp tất cả 4 ID lỗi này và bắn Debezium incremental snapshot signal qua NATS topic `"cdc.cmd.debezium-signal"`.
- **Hành vi thực tế**: Luồng re-sync này **không bao giờ chạy/hoàn tất** vì Debezium incremental snapshot signal qua NATS topic `"cdc.cmd.debezium-signal"` đã bị ngắt cấu hình (ngắt luồng trên Debezium). Dữ liệu lệch ở Shadow DB vẫn hoàn toàn giữ nguyên, chưa được sửa.

#### 🔸 Lần 2: Chạy Heal tiếp ngay sau đó - Report ID 8 (`status = 'healed'`)
- **Quét Window Tier 2 Sạch Drift (Cơ chế Trôi Cửa Sổ 2h Hot Lookback)**:
  - Khi User trigger heal lần 2, hệ thống tự động chạy một lượt quét fresh `RunTier2`.
  - Do hệ thống chạy ở **Hot mode (mặc định)**, hàm `effectiveLookback` chỉ sử dụng cấu hình **`HotWindowLookback` = 2 giờ** (thay vì 7 ngày của Cold mode).
  - Vì thời gian trôi qua và sự dịch chuyển của upper watermark (theo lag), các bản ghi bị lệch của Report 7 **đã trôi hoàn toàn ra ngoài cửa sổ quét 2 giờ gần nhất** của Report 8.
  - Do đó, lượt quét window-based XOR-hash của Report 8 báo kết quả **sạch hoàn toàn** (`MissingCount = 0`, `StaleCount = 0`, `OrphanCount = 0`). Đây là lý do các mảng `missing_ids` và `stale_ids` trong Report 8 ban đầu đều là `null`.
- **Kích hoạt Safety Net (Full ID Diff check) ngoài Window**:
  - Vì quét window Tier 2 báo sạch, healer kích hoạt cơ chế an toàn **Full ID Diff check** để so sánh danh sách ID toàn bộ bảng.
  - Kết quả Full ID check phát hiện ra **2 bản ghi bị thiếu** ở Shadow DB ngoài cửa sổ quét.
  - Do `eventHandler` đã được wired (`eventHandler != nil`), healer không dùng NATS signal nữa mà đi thẳng vào **Direct path**: Gọi hàm `FetchAndWriteByIDs`.
  - Hàm này thực hiện query trực tiếp MongoDB Source (`MongoDB.Find`), chuyển đổi thành heal envelope, chuyển qua `EventHandler.HandleRaw` và thực hiện OCC upsert ghi đè thẳng xuống Shadow DB thông qua `BatchBuffer` rồi Flush.
  - Trạng thái Report 8 lập tức được cập nhật đè: đổi `status` thành `"healed"`, cập nhật `missing_count = 2`, `healed_count = 2`. Các mảng ID window-based của Report 8 (đã quét trước đó) vẫn giữ nguyên là `null`.

### 3. Audit Chi Tiết Đường Chạy Heal Thực Tế (FetchAndWriteByIDs vs NATS Debezium Signal)
Chúng tôi đã thực hiện audit mã nguồn cấu hình runtime để xác định chính xác đường đi của lệnh heal trên thực tế:
- **Đăng ký Dependency (Wiring)**: Tại [server_setup.go:L336](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/server/server_setup.go#L336), `reconHandler` được khởi tạo với cấu hình `WithEventHandler(eventHandler)`. Vì vậy `h.eventHandler` trên live server **thực sự khác nil**.
- **Khi đối soát phát hiện drift (Report 7)**:
  - Đi theo nhánh đối soát window tiêu chuẩn.
  - Nhánh này **LUÔN LUÔN** thực hiện bắn Debezium incremental snapshot signal qua NATS topic `"cdc.cmd.debezium-signal"`.
- **Khi đối soát window báo sạch drift, chạy Safety Net (Report 8)**:
  - Do `h.eventHandler != nil`, hệ thống đi thẳng vào **Direct path**: Gọi hàm `FetchAndWriteByIDs`.
  - Quy trình chi tiết của `FetchAndWriteByIDs`:
    1. Query trực tiếp MongoDB Source: `MongoDB.Find({_id: {$in: [id]}})` lấy dữ liệu tươi mới nhất từ replica primary.
    2. Gói document thành "heal envelope" (tương tự định dạng snapshot runner).
    3. Chuyển cho `EventHandler.HandleRaw` xử lý.
    4. Ghi đè vào Shadow DB qua `BatchBuffer` (soft/hard shadow upsert) có OCC bảo vệ timestamp.
    5. Gọi `FlushBatchBuffer` đồng bộ xuống database.
    6. Shadow DB tự động trigger `transmute hook` để đồng bộ master table.
    7. Cập nhật báo cáo đối soát: `missing_count=1, healed_count=1, status=healed`.

---

## 9. Phân Tích Gốc Rễ Type Mapping DDL Core (TIMESTAMP vs TIMESTAMPTZ)

Để giải quyết triệt để lỗi từ gốc rễ hệ thống (Core Systems) theo phản hồi của bạn, chúng tôi đã tiến hành rà soát kỹ thuật sâu vào module DDL generation và kiểu dữ liệu core của hệ thống shadow:

### 1. Sự Không Đồng Nhất Trong Core Column Types
Qua kiểm tra mã nguồn, chúng tôi phát hiện sự thiếu đồng nhất về thiết kế kiểu dữ liệu date/time ở tầng core của Shadow DB:
- **Cột Metadata Hệ thống**: Tại [schema_adapter.go:L24](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/shadow/schema_adapter.go#L24), các cột metadata như `_synced_at`, `_created_at`, `_updated_at` được hardcode là:
  ```go
  {"_synced_at", "TIMESTAMP DEFAULT NOW()"},
  {"_created_at", "TIMESTAMP DEFAULT NOW()"},
  {"_updated_at", "TIMESTAMP DEFAULT NOW()"},
  ```
  Tất cả đều dùng kiểu **`TIMESTAMP` (without time zone)**.
- **Cột Trạng Thái Explode**: Trái lại, trong [child_explode.go:L191](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/shadow/child_explode.go#L191), cột metadata `_synced_at` lại được khai báo là:
  ```go
  "_synced_at" TIMESTAMPTZ NOT NULL DEFAULT NOW()
  ```
  Tức là sử dụng **`TIMESTAMPTZ` (with time zone)**.

### 2. Nguyên Nhân Suy Diễn Kiểu Cột Business Thành TIMESTAMP
Khi người dùng đăng ký hoặc quét schema (Schema Discovery), hệ thống tự động suy diễn kiểu dữ liệu cho Shadow DB từ MongoDB document mẫu.
Tại [source_router.go:L38](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/source/source_router.go#L38), hàm `InferTypeFromRawData` thực hiện logic map:
```go
func InferTypeFromRawData(jsonValue interface{}) string {
    ...
    switch v := jsonValue.(type) {
    ...
    case string:
        if isRFC3339Like(v) {
            return "TIMESTAMP" // <--- ĐÂY CHÍNH LÀ GỐC RỄ LỖI!
        }
        return "TEXT"
    ...
```
* **Lỗi thiết kế**: Khi phát hiện chuỗi có định dạng ngày tháng (RFC3339), core system mặc định suy diễn và trả về kiểu dữ liệu là **`TIMESTAMP`** (without time zone).
* **Hệ quả**:
  - DDL sinh ra khởi tạo cột `last_updated_at` trong Shadow DB là kiểu `TIMESTAMP`.
  - MongoDB lưu thời gian ở UTC, Debezium đồng bộ đúng chuỗi UTC đó vào Postgres. Nhưng do kiểu cột là `TIMESTAMP` (without timezone), Postgres lưu trữ thô và làm mất thông tin múi giờ.
  - Khi Go driver (pgx) scan cột này vào `time.Time`, do không có timezone, nó tự động áp múi giờ hệ thống của máy chạy ứng dụng (`time.Local`, hiện tại là múi giờ **+07:00** của bạn).
  - Khi tính toán XOR-hash và drill-down so sánh `UnixMilli()`, giá trị bị hụt đi 7 tiếng, tạo ra các cảnh báo **mismatched giả** hàng loạt.

### 3. Phương Án Sửa Đổi Core Hợp Lệ (DoD Alignment)
Để triệt tiêu lỗi này từ gốc rễ kiến trúc và tuân thủ đúng quy tắc hệ thống, cần tiến hành sửa đổi logic sinh kiểu dữ liệu của core DDL:
1. **Sửa đổi Type Inference**: Cập nhật hàm `InferTypeFromRawData` tại [source_router.go:L54](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/source/source_router.go#L54) để trả về **`TIMESTAMPTZ`** thay vì `TIMESTAMP`:
   ```diff
   -		if isRFC3339Like(v) {
   -			return "TIMESTAMP"
   -		}
   +		if isRFC3339Like(v) {
   +			return "TIMESTAMPTZ"
   +		}
   ```
2. **Sửa đổi Metadata Columns**: Đồng bộ hóa toàn bộ các cột metadata thời gian hệ thống trong [schema_adapter.go:L24](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/shadow/schema_adapter.go#L24) sang **`TIMESTAMPTZ`** để đồng nhất thiết kế:
   ```diff
   -	{"_synced_at", "TIMESTAMP DEFAULT NOW()"},
   +	{"_synced_at", "TIMESTAMPTZ DEFAULT NOW()"},
   ```
Khi tất cả các cột thời gian được lưu dưới dạng `TIMESTAMPTZ`, Postgres engine sẽ lưu trữ UTC chuẩn và driver `pgx` của Go luôn scan ra đối tượng `time.Time` có múi giờ **UTC** chuẩn xác (không bị ảnh hưởng bởi múi giờ local của container/server), giải quyết triệt để bài toán XOR-hash lệch múi giờ.

---

## 10. Phân Tích Logic Phân Nhánh Heal & Đánh Giá Rủi Ro Safety Net

Dựa trên phản hồi về 3 lần chạy heal vô vọng ở các Report 11, 12, 13 (vẫn báo drift, không update lại shadow) và cảnh báo rủi ro về Safety Net, chúng tôi đã tiến hành phân tích chi tiết logic phân nhánh trong `healSegmentA` ([recon_heal_v4.go:L273](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_heal_v4.go#L273)):

### 1. Phân Tích Bản Chất Hai Phân Nhánh Chạy Thực Tế

Trong logic `healSegmentA`, hệ thống phân chia luồng heal thành 2 nhánh hoàn toàn riêng biệt dựa trên kết quả của lượt quét fresh đối soát Tier 2:

```go
newReport := h.reconCore.RunTier2(coldCtx, *entry)
if newReport.MissingCount == 0 && newReport.StaleCount == 0 && newReport.OrphanCount == 0 {
    // 🔸 NHÁNH 2: Safety Net (Khi quét window báo sạch drift)
    fullMissing, ... := h.reconCore.FullIDDiffMissingFromShadow(ctx, *entry)
    ...
} else {
    // 🔸 NHÁNH 1: Heal Window Drift (Khi quét window phát hiện drift)
    healIDs := append(...)
    ...
}
```

* **Lý do kẹt ở Nhánh 1 (Report 11, 12, 13)**: 
  - Trong Report 11, 12, 13, đối soát window Tier 2 phát hiện ra 1 record missing (`6a44867951c80c9c38556f50`) và 1 record mismatched (`6a43650fec5b9378333d6362`).
  - Do có số lượng drift khác 0, hệ thống **bắt buộc rẽ vào Nhánh 1**.
  - Tại Nhánh 1, healer **không hề có logic fallback** sang direct path `FetchAndWriteByIDs`. Nó hardcode 100% việc bắn tin nhắn NATS tới topic `"cdc.cmd.debezium-signal"`.
  - Vì Debezium incremental snapshot signal đã bị ngắt cấu hình, các bản ghi drift này không bao giờ được re-sync.
  - Sau mỗi lượt heal, dữ liệu vẫn lệch nguyên vẹn. Các lần chạy tiếp theo vẫn phát hiện drift và tiếp tục rơi vào vòng lặp kẹt ở Nhánh 1, không bao giờ được chữa lành.

* **Lý do nhảy qua Nhánh 2 ở Report 8 trước đó**:
  - Tại Report 8, do dữ liệu lệch đã nằm ngoài cửa sổ quét hot lookback (2 giờ), lượt quét Tier 2 trả về kết quả sạch drift tuyệt đối (`MissingCount == 0 && StaleCount == 0 && OrphanCount == 0`).
  - Hệ thống thỏa mãn điều kiện `MissingCount == 0 ...` nên chuyển tiếp thành công sang **Nhánh 2 (Safety Net)**.
  - Ở Nhánh 2, do kiểm tra thấy `eventHandler != nil`, hệ thống đi theo **Direct Path** gọi `FetchAndWriteByIDs` và ghi trực tiếp thành công vào Shadow DB.

---

## 11. Tài Liệu Thiết Kế Chi Tiết FE/BE & Cải Tiến Logic Routing Luồng Heal

Nhằm khắc phục triệt để sự phụ thuộc vào Debezium NATS signal, kiểm soát tốt tải hệ thống quy mô lớn, và bàn giao luồng kích hoạt an toàn cho CMS-Web, chúng tôi thiết kế lại toàn diện hai đầu Frontend (FE) và Backend (BE):

### 1. Thiết Kế Frontend (FE Modal kích hoạt Heal trên CMS-Web)
Giao diện Modal kích hoạt heal trên CMS-Web được bổ sung các controls chọn chế độ quét và bộ lọc thời gian:

- **Bộ chọn chế độ quét (Radio buttons)**:
  - `[Radio] Chế độ Window (Quét cửa sổ thời gian)` (Mặc định)
  - `[Radio] Chế độ Full-diff (Quét so sánh toàn bảng)`
- **Bộ lọc thời gian (Date/Time Picker Inputs)**:
  - Input `from` (StartTime)
  - Input `to` (EndTime)
- **Logic kiểm soát trạng thái giao diện (Javascript / React UI)**:
  - **Khi chọn `Chế độ Window`**:
    - Tự động **Disable** (vô hiệu hóa) cả 2 ô nhập liệu `from` và `to`.
    - Tự động tính toán và hiển thị khoảng thời gian **7 ngày gần nhất** (`from = now - 7 days`, `to = now`) trong trạng thái readonly để người dùng dễ quan sát cửa sổ đối soát.
  - **Khi chọn `Chế độ Full-diff`**:
    - **Enable** (kích hoạt) 2 ô nhập liệu `from` và `to` cho phép người dùng tự do lựa chọn khoảng thời gian cần heal.
    - **Validate nghiệp vụ**: Hệ thống liên tục kiểm tra giá trị của 2 input. Nếu `from` hoặc `to` bị bỏ trống, hoặc hiệu số thời gian giữa chúng vượt quá **30 ngày** ($\Delta t > 30 \text{ ngày}$):
      - Hiển thị thông báo cảnh báo đỏ bên dưới: *"Khoảng thời gian quét Full-diff không được vượt quá 30 ngày để bảo vệ DB!"*.
      - **Disable** nút **"Submit Heal / Thực hiện"** trên Modal.
- **NATS Payload gửi lên Backend**:
  ```json
  {
    "table": "shadow_testexp.export_jobs",
    "segment": "source_shadow",
    "mode": "window", 
    "start_time": "2026-06-24T00:00:00Z",
    "end_time": "2026-07-01T00:00:00Z"
  }
  ```

### 2. Thiết Kế Backend (BE) Payload & Validate
- **Cập nhật Struct Payload**: Handler NATS tại Backend unmarshal thêm 3 trường cấu hình:
  ```go
  type HealPayload struct {
      Table     string `json:"table"`
      Segment   string `json:"segment"` // "source_shadow" (A) / "shadow_master" (B)
      Legacy    bool   `json:"legacy"`
      Mode      string `json:"mode"`       // "window" hoặc "full_diff"
      StartTime string `json:"start_time"` // RFC3339 string
      EndTime   string `json:"end_time"`   // RFC3339 string
  }
  ```
- **Validate Khoảng Thời Gian tại Backend (Safety Net Protection)**:
  - Khi handler nhận tin nhắn NATS với `Mode == "full_diff"`, BE bắt đầu parse `StartTime` và `EndTime` sang `time.Time`.
  - Nếu parse lỗi, hoặc giá trị trống, hoặc hiệu số thời gian vượt quá 30 ngày, BE trả về message lỗi lập tức và dừng xử lý để tránh rủi ro OOM/DB spike do client cố tình bypass FE:
    ```go
    if payload.Mode == "full_diff" {
        start, err1 := time.Parse(time.RFC3339, payload.StartTime)
        end, err2 := time.Parse(time.RFC3339, payload.EndTime)
        if err1 != nil || err2 != nil || end.Before(start) || end.Sub(start) > 30*24*time.Hour {
            return fmt.Errorf("invalid time range for full-diff: must be bounded within 30 days")
        }
    }
    ```

### 3. Cải Tiến Logic Phân Nhánh & Routing Tại BE (`healSegmentA`)
Hệ thống chuyển đổi hoàn toàn cơ chế phân nhánh tự động (dựa trên drift của RunTier2) sang **chủ động phân nhánh theo tham số `Mode` từ FE**:

#### 🔸 Nhánh A. Chạy ở `Mode == "window"` (Window Mode)
- **Hành vi**: Gọi `RunTier2` fresh scan với khoảng thời gian mặc định 7 ngày (hoặc lấy từ payload start_time/end_time).
- **Heal Path**: Khi phát hiện có drift (`newReport.MissingCount > 0 || newReport.StaleCount > 0`):
  - **Loại bỏ hoàn toàn việc bắn Debezium NATS signal** tới topic `"cdc.cmd.debezium-signal"`.
  - Thay thế bằng việc **gọi trực tiếp `FetchAndWriteByIDs`** (direct write) để fetch document tươi mới nhất từ MongoDB và OCC upsert ghi đè thẳng xuống Shadow DB. Việc này giúp dữ liệu drift được chữa lành tức thì mà không phụ thuộc vào connector Debezium!

#### 🔸 Nhánh B. Chạy ở `Mode == "full_diff"` (Full-diff Mode)
- **Hành vi**:
  - Không chạy quét window XOR-hash Tier 2.
  - Chạy hàm quét an toàn có bộ lọc thời gian: `TimeBoundedDiffMissingFromShadow(ctx, entry, startTime, endTime)`.
    - **PostgreSQL Shadow Query**: Lọc theo index timestamp của shadow table:
      ```sql
      SELECT "_source_id"::text FROM shadow_table 
      WHERE NOT "_deleted" AND "_source_id" IS NOT NULL
        AND "last_updated_at" >= :start_time AND "last_updated_at" < :end_time
      ```
    - **MongoDB Source Query**: Stream sử dụng cursor filter có range:
      ```json
      {
        "lastUpdatedAt": {
          "$gte": ISODate("start_time"),
          "$lt": ISODate("end_time")
        }
      }
      ```
  - So khớp tập ID ở RAM (giới hạn tối đa 30 ngày nên tập ID nhỏ, bảo toàn RAM < 100MB).
  - **Heal Path**: Gọi trực tiếp `FetchAndWriteByIDs` cho danh sách ID missing tìm thấy để heal trực tiếp xuống Shadow DB.
