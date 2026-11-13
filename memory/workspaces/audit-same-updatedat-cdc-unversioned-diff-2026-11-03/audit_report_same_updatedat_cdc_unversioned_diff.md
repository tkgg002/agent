---
tier: T2
workspace: audit-same-updatedat-cdc-unversioned-diff-2026-11-03
suffix: same_updatedat_cdc_unversioned_diff
mode: standard
---

# Audit & Implementation Report — Kiểm Định Phản Biện Toàn Trình (S5/S6) Cho Case `261029KHHDUH` (`MERCHANT_PAYMENT` — `unversioned_diff`)

## 1. Bảng Tổng Hợp Issues & Tổng Kết Hành Động (Executive Issues Summary Table — `L007`)

### 1.1. Các Hạng Mục Cần Phối Hợp Runtime Thật Cùng Anh (Pending Live Runtime Trace)
| Mã | Nhóm | Mức độ | Vị trí / Thành phần | Mô tả & Lý do Không Tự Chạy | Hành động Tiếp theo |
| :--- | :--- | :---: | :--- | :--- | :--- |
| **`RT-01`** | **Live Runtime Trace** | **MEDIUM** | `centralized-data-service` Worker (Kafka Debezium $\rightarrow$ Shadow PostgreSQL $\rightarrow$ Master) | Các bài test `EV-20261103T135646-5a6e`, `EV-20261103T135653-7bc0`, `EV-20261103T135657-b281`, `EV-20261103T132536-a432` đã kiểm chứng logic trích xuất `_source_ts` (`ts_ms + min(ord, 999)`), bộ sinh SQL PostgreSQL và thứ tự OCC trên SQLite in-memory (`setupTestDB` tại `schema_adapter_ordering_test.go:15-16`). Theo kỷ luật `L040`/`L041` (tuyệt đối không tự ý chạy lệnh ghi/DDL lên DB thật) và `L027` (không lấy unit test/SQLite in-memory để suy diễn runtime thật), cần phối hợp với anh khi nạp binary mới lên môi trường chạy thật. | Phối hợp cùng anh restart worker `centralized-data-service` trên môi trường staging/production và trace giá trị runtime `_source_ts`, `_hash`, `status` khi có bản tin Debezium MongoDB cùng giây. |

### 1.2. Các Vấn Đề Đã Phát Hiện & Tự Khắc Phục Dứt Điểm Trong Vòng Audit (`S5 -> S3 -> S4 -> S5`)
| Mã | Mức độ | Component & `file:line` | Phát hiện Phản biện (Adversarial Finding) | Giải pháp Đã Thực Thi & Biên lai `EV-...` | Trạng thái |
| :--- | :---: | :--- | :--- | :--- | :---: |
| **`F1-A`** | **CRITICAL (P0)** | `internal/handler/shadow/utils.go:72-112` (`112` dòng) | Debezium MongoDB xuất `source.ts_ms = seconds * 1000` (từ BSON Oplog `Timestamp(seconds, ord)`), làm 2 lệnh ghi trong cùng giây `05:17:04Z` có cùng `_source_ts = 1761715024000`. | Tách `parseAnyInt64` và nâng cấp `extractSourceTsMs` cộng `min(ord, 999)` vào `tsMs` khi `tsMs > 0 && tsMs%1000 == 0`. Biên lai: `EV-20261103T135646-5a6e` (`exit=0`). | **RESOLVED (`100%`)** |
| **`F1-B`** | **CRITICAL (P0)** | `internal/service/shadow/schema_adapter.go:768-789` (`1136` dòng, giảm 1 dòng so với trước S4 và giảm 22 dòng so với `HEAD: 1158`) | `buildOCCWhereClause` trước đây dùng `<` ngặt (`_source_ts < EXCLUDED._source_ts`), đánh rơi sự kiện thứ 2 (`PENDING`) khi 2 sự kiện cùng `_source_ts` nằm ở 2 đợt flush/SQL khác nhau (`0 rows affected`). | Cho phép `_source_ts = EXCLUDED._source_ts AND EXCLUDED._source <> 'snapshot:v2'` (hoặc `<=` khi không có cột `_source`), kết hợp `_hash IS DISTINCT FROM EXCLUDED._hash` sẵn có. Biên lai: `EV-20261103T135657-b281` (`7/7 PASS`, `exit=0`), `EV-20261103T132536-a432` (`exit=0`). | **RESOLVED (`100%`)** |
| **`BUG-AUDIT-01`** | **LOW (Code Quality / DRY)** | `internal/handler/shadow/utils.go:72-74` | Khi gọi `/laya_router` Mode B (`06_laya_summary_dg3review_20261103T135231.json`), Laya trả `REJECT_REDUNDANT` vì `parseAnyInt64` tự viết lại vòng lặp unwrap map 1 phần tử vốn đã có sẵn trong `unwrapAvroUnion(raw)` tại `internal/handler/shadow/avro_helper.go:67-82`. | Khử bỏ vòng lặp thừa, gọi trực tiếp `raw = unwrapAvroUnion(raw)` tại `utils.go:73` (giảm từ `117` xuống `112` dòng). Sau khi sửa, `/laya_router` (`06_laya_summary_dg3_20261103T135840.json`) chuyển `A-FIX-1` sang `ACCEPT` ngay lập tức. Biên lai: `EV-20261103T135646-5a6e` (`exit=0`). | **RESOLVED (`100%`)** |
| **`BUG-AUDIT-02`** | **MEDIUM (Peer Consistency)** | `internal/sinkworker/utils.go:88-104` (`367` dòng)<br>`test/internal/sinkworker/sinkworker_test.go:135-171` (`267` dòng) | Truy vết đầu-cuối (`L029`) phát hiện binary phụ `cmd/sinkworker` cũng có hàm `extractSourceTsMs(envelope map[string]any) int64` tại `internal/sinkworker/utils.go:88-96` chỉ lấy `src["ts_ms"]` mà thiếu cộng `min(ord, 999)` khi `ts % 1000 == 0`. | Đồng bộ logic cộng `min(ord, 999)` khi `ts > 0 && ts%1000 == 0` vào `internal/sinkworker/utils.go:88-104` và bổ sung test tại `test/internal/sinkworker/sinkworker_test.go:146-160`. Biên lai: `EV-20261103T135653-7bc0` (`exit=0`). | **RESOLVED (`100%`)** |
| **`F2`** | **INFO (Out of Scope per User)** | `core-trans-proxy-service` (`cashin.publish.ts:34-44`) | `createTransHis` và `updateTransHis` gọi `handlePromise` không `await` làm log `updateTransHisStatus` in ra trước `createTransHis` `23 ms` dù trên MongoDB Oplog `insertOne` chạy trước `updateOne`. | Giữ nguyên 100% theo chỉ đạo của anh (*"ko đụng vào service khác"*). | **BY DESIGN** |

---

## 2. Kết Quả Kiểm Định 6 Bước Adversarial Audit (S5 — Sau Vòng Lặp Tự Chữa Lành)

### Mục 1: Phản biện từng dòng update (`git diff` Line-by-Line Critique), từng task (`08_tasks`) & Kiến trúc / Single-Responsibility (`L017`)
- **Kiểm tra tuân thủ Layer, Design Pattern & Chống phình to God-files (`wc -l` thực tế sau lần sửa cuối):**
  - `internal/handler/shadow/utils.go`: `HEAD: 104` dòng $\rightarrow$ `NOW: 112` dòng (`+8` dòng). File tiện ích nhỏ (`112` dòng), đặt đúng layer `internal/handler/shadow`, tái sử dụng `unwrapAvroUnion` từ `avro_helper.go:67`.
  - `internal/service/shadow/schema_adapter.go`: `HEAD: 1158` dòng (`BEFORE S4: 1137` dòng) $\rightarrow$ `NOW: 1136` dòng (**`-1` dòng so với trước S4, `-22` dòng so với `HEAD`**). Tuân thủ nghiêm ngặt bất biến `NOW <= HEAD` trên file lớn (`L017`), tuyệt đối không nhồi thêm hàm mới vào `schema_adapter.go`.
  - `internal/sinkworker/utils.go`: `HEAD: 359` dòng $\rightarrow$ `NOW: 367` dòng (`+8` dòng). Đặt đúng layer `internal/sinkworker`.
  - Các file test: `internal/handler/shadow/kafka_consumer_test.go` (`205` dòng), `internal/service/shadow/system_columns_upsert_test.go` (`139` dòng), `test/internal/sinkworker/sinkworker_test.go` (`267` dòng), `test/internal/service/schema_adapter_ordering_test.go` (`324` dòng).
  - **Tổng cộng (`wc -l` trên 7 file):** `2550` dòng.
- **Phản biện từng dòng `git diff` & Tương thích ngược (Backward Compatibility):**
  1. `internal/handler/shadow/utils.go:72-94` (`parseAnyInt64`): Gọi `raw = unwrapAvroUnion(raw)` rồi switch qua tất cả các kiểu số (`int64`, `int`, `int32`, `float64`, `json.Number`, `string`). Nếu không hợp lệ trả về `0` giống hệt hành vi cũ.
  2. `internal/handler/shadow/utils.go:96-112` & `internal/sinkworker/utils.go:88-104` (`extractSourceTsMs`):
     - Điều kiện `if tsMs > 0 && tsMs%1000 == 0` đảm bảo **chỉ** cộng `ord` (`1..999`) khi `ts_ms` là giây chẵn (đặc trưng của BSON Oplog `Timestamp(seconds, ord)` trên MongoDB).
     - Với mọi nguồn PostgreSQL/MySQL hoặc sự kiện có sẵn độ phân giải millisecond lẻ (`tsMs % 1000 != 0`), giá trị `tsMs` được giữ nguyên 100%, không gây bất kỳ thay đổi nào cho các connector khác.
     - Trần `if ord > 999 { ord = 999 }` đảm bảo `tsMs + ord` luôn nằm gọn trong phạm vi `[seconds*1000 + 1, seconds*1000 + 999]`, không bao giờ tràn sang giây kế tiếp `(seconds + 1) * 1000`.
     - **Kiểm tra tương tác với `SnapshotRunner` (`snapshot_runner_utils.go:60`):** `captureClusterTime` lấy `int64(ts.T) * 1000` (đuôi `.000`). Nhờ sự kiện Debezium live trong cùng giây `T` được cộng `ord >= 1`, bản ghi live luôn có `_source_ts = T*1000 + ord > T*1000` của `snapshot:v2`, bảo vệ tuyệt đối bản ghi live trước snapshot cùng giây!
     - **Kiểm tra tương tác với Recon & Master Transmute:** Trên bảng `trans_his`, đối soát Segment A/B dùng cột thời gian nghiệp vụ (`updatedAt` / `updated_at` tại `recon_tier_a.go:201-216`) và Master giữ nguyên `updatedAt` từ document gốc (`mongo_extjson_unwrapper.go:358-364`), nên việc cộng `ord` vào cột hệ thống `_source_ts` hoàn toàn tương thích ngược 100%.
  3. `internal/service/shadow/schema_adapter.go:773-781` (`buildOCCWhereClause`):
     - Đổi nhánh equal timestamp từ `AND %s."_source" = 'snapshot:v2' AND EXCLUDED."_source" <> 'snapshot:v2'` thành `AND EXCLUDED."_source" <> 'snapshot:v2'` (và `<=` khi bảng không có cột `_source`).
     - Vì `buildOCCWhereClause` luôn ghép `WHERE changeClause AND timeClause` với `changeClause = (%s."_hash" IS NULL OR %s."_hash" IS DISTINCT FROM EXCLUDED."_hash")` (`schema_adapter.go:755-758, 784-785`), mọi bản tin replay trùng nội dung (cùng `_hash`) vẫn bị chặn đứng (`0 rows affected`), trong khi bản tin cập nhật thực sự trong cùng giây (`_hash` khác) được phép cập nhật (`1 row affected`).

### Mục 2: Chống suy diễn Smoke Test / Mockup / Hallucination & Trace Runtime thật
- **Bóc tách minh bạch bản chất từng bài test (Zero Hallucination):**
  1. `TestExtractSourceTsMs_WithMongoOrdSubSecondTieBreaker` (`internal/handler/shadow/kafka_consumer_test.go:174-205`, biên lai `EV-20261103T135646-5a6e`) và `TestExtractSourceTsMs` (`test/internal/sinkworker/sinkworker_test.go:135-171`, biên lai `EV-20261103T135653-7bc0`): Unit test kiểm chứng hàm trích xuất `_source_ts` trong bộ nhớ.
  2. `TestSchemaAdapter_DedupSystemColumnsInBatchAndSingleUpsert` (`internal/service/shadow/system_columns_upsert_test.go:134-138`, biên lai `EV-20261103T132536-a432`): Unit test kiểm chứng chuỗi câu lệnh SQL PostgreSQL (`ON CONFLICT ... DO UPDATE WHERE ...`) do `SchemaAdapter` sinh ra.
  3. `TestEventOrdering_SameSecondPositiveTs_UpdatesAndGuardsSnapshot` (`test/internal/service/schema_adapter_ordering_test.go:258-322`, biên lai `EV-20261103T135657-b281`): Chạy câu lệnh SQL thực thi qua `setupTestDB(t)` (`schema_adapter_ordering_test.go:15-16`) sử dụng **SQLite in-memory (`sqlite.Open("file::memory:?cache=shared")`)**, KHÔNG PHẢI PostgreSQL thật.
- **Không gắn mock vào code production & Nêu rõ phối hợp Trace Runtime thật:**
  - Không có bất kỳ trường closure mock (`*Fn`), `sqlmock`, hay nhánh test giả nào được thêm vào code production (`utils.go`, `schema_adapter.go`).
  - Em tuyệt đối không suy diễn từ kết quả unit test / SQLite in-memory rằng đã trace xong trên PostgreSQL/Kafka runtime thật. Để kiểm chứng runtime value thật trên luồng Kafka Debezium $\rightarrow$ Shadow PostgreSQL $\rightarrow$ Master, em xin phép phối hợp cùng anh sau khi anh deploy/restart worker `centralized-data-service`.

### Mục 3: Kiểm tra Concurrency / Multi-Worker / Batch-Debouncer / Timeout-Heartbeat / Watermark Continuity
- **Concurrency & Multi-Worker Partitioning (`kafka_consumer.go` & `BatchBuffer`):** Debezium MongoDB định tuyến mọi sự kiện của cùng một document `_id` (`261029KHHDUH`) vào cùng một Kafka partition nên worker đọc tuần tự (`ord = k` trước, `ord = k + 1` sau).
- **Tương tác giữa In-Memory Batch (`dedupByConflictKey`) và Cross-Batch SQL (`buildOCCWhereClause`):**
  - Nếu 2 sự kiện cùng giây nằm trong **cùng 1 đợt flush của `BatchBuffer`**: `dedupByConflictKey` (`internal/handler/shadow/batch_buffer_utils.go:90-153`) dùng `incomingTs >= existingTs` nên giữ bản ghi thứ 2 (`PENDING`).
  - Nếu 2 sự kiện cùng giây **vắt qua ranh giới 2 đợt flush của `BatchBuffer`** (hoặc đi vào nhánh single-row fallback tại `batch_buffer.go:674-750`): PostgreSQL khóa row-level lock trên `_source_id`; nhờ cả `1A` (`ts1 < ts2`) lẫn `1B` (`_source_ts = EXCLUDED._source_ts AND EXCLUDED._source <> 'snapshot:v2'`), câu SQL thứ 2 cập nhật thành công `1 row affected` thay vì bị drop như trước.
- **Timeout/Heartbeat (`AckWait` vs `InProgress`) & Watermark Continuity (`TableDebouncer` / `Transmuter`):**
  - Thay đổi chỉ diễn ra trong phép tính số học `int64` (`utils.go`) và biểu thức vị từ SQL (`schema_adapter.go`), `0` I/O phụ trội, không giữ mutex qua mạng (`L033`), không ảnh hưởng tới chu kỳ `AckWait`/`InProgress` của NATS JetStream (`L050`) và đảm bảo `BatchBuffer` chỉ phát sự kiện sang `TableDebouncer` / commit Kafka offset (`WatermarkTracker` — `L051`) sau khi PostgreSQL ghi thành công.

### Mục 4: Kiểm tra suy diễn & báo cáo láo (Honesty & Anti-Speculation Verification)
- **Kiểm tra đối chiếu trung thực 100% mọi biên lai từ `evidence_run.py`:**
  - Biên lai PASS sau vòng Self-Healing:
    - `EV-20261103T135646-5a6e` (`exit=0`): `go test -v ./internal/handler/shadow -run TestExtractSourceTsMs_WithMongoOrdSubSecondTieBreaker` (`PASS 0.824s`).
    - `EV-20261103T135653-7bc0` (`exit=0`): `go test -v ./test/internal/sinkworker -run TestExtractSourceTsMs` (`PASS 0.600s`).
    - `EV-20261103T135657-b281` (`exit=0`): `go test -v ./test/internal/service/schema_adapter_ordering_test.go -run TestEventOrdering` (`7/7 PASS 0.524s`).
    - `EV-20261103T135700-e26a` (`exit=0`): `go build ./cmd/... ./internal/...` (`exit=0`).
  - Biên lai PASS ở vòng S4 đầu tiên:
    - `EV-20261103T132348-cf1d` (`exit=0`), `EV-20261103T132409-6241` (`exit=0`), `EV-20261103T132425-0123` (`exit=0`), `EV-20261103T132536-a432` (`exit=0`), `EV-20261103T132539-bcd2` (`exit=0`).
  - Khai báo trung thực toàn bộ các biên lai lỗi trong quá trình chạy lệnh (không giấu lỗi theo `L011`):
    - `EV-20261103T132356-e80e` (`exit=1`): Chạy cả thư mục `./test/internal/service` bị vướng file cũ không liên quan `recon_pipeline_span_test.go:12:24: undefined: servicerecon.PipelineFlowInfo`.
    - `EV-20261103T132455-4c2f` (`exit=1`): Assertion chuỗi SQL trong `system_columns_upsert_test.go` lần đầu chưa chuẩn hóa khoảng trắng multi-line; đã sửa tại dòng `135` và pass ở `EV-20261103T132536-a432`.
    - `EV-20261103T132458-5877` (`exit=1`): `go build ./...` vướng file backup cũ `docs/server_bk/worker_server_init.go`; đã chạy `go build ./cmd/... ./internal/...` pass ở `EV-20261103T132539-bcd2` & `EV-20261103T135700-e26a`.

### Mục 5: Vòng lặp Phản tỉnh & Khắc phục (Self-Improvement Loop — `S5 -> S3 -> S4 -> S5`)
- Trong lượt Audit này, khi chạy `/laya_router` Mode B (`06_laya_summary_dg3review_20261103T135231.json`) và truy vết Callchain (`L029`), em đã phát hiện 2 điểm cần hoàn thiện (`BUG-AUDIT-01`: vòng lặp unwrap Avro union dư thừa trong `parseAnyInt64` tại `internal/handler/shadow/utils.go:73-78`; `BUG-AUDIT-02`: hàm `extractSourceTsMs` bên `internal/sinkworker/utils.go:88-96` chưa cộng `ord`).
- Áp dụng đúng quy trình **Self-Healing Loop (`S5 -> S3 -> S4 -> S5`)** theo Rule #9 mục 8(d) và `L023`:
  1. Cập nhật `08_tasks_same_updatedat_cdc_unversioned_diff.md` (`TASK-BUG-01`) và `09_tasks_solution_same_updatedat_cdc_unversioned_diff.md` (Mục `4.3`).
  2. Thực thi sửa dứt điểm `internal/handler/shadow/utils.go`, `internal/sinkworker/utils.go`, `test/internal/sinkworker/sinkworker_test.go`.
  3. Chạy lại toàn bộ test & build qua `evidence_run.py` (`EV-20261103T135646-5a6e`, `EV-20261103T135653-7bc0`, `EV-20261103T135657-b281`, `EV-20261103T135700-e26a`) và chạy lại `/laya_router` (`DG3`) đạt `PASS` ngay trong cùng lượt mà không đẩy việc lại cho anh.

### Mục 6: Kết quả QC & Nhận xét từ `/laya_router` (`DG3`)
- **Mode B (`06_laya_summary_dg3review_20261103T135231.json`):**
  - `internal/service/shadow/schema_adapter.go`: `round_1: APPROVE` $\rightarrow$ `decision: ACCEPT`.
  - `internal/handler/shadow/utils.go`: `round_1: REJECT_REDUNDANT` (nhờ nhận xét này của Laya, em đã phát hiện và xóa vòng lặp unwrap Avro union bị lặp với `unwrapAvroUnion` trong `avro_helper.go:67`, giảm `utils.go` từ `117` xuống `112` dòng).
- **Mode A sau khi hoàn tất Self-Healing (`06_laya_summary_dg3_20261103T135840.json` — `gate: PASS`, `trusted_rounds: ["r1", "r2", "r3"]`):**
  - `A-FIX-1` (`internal/handler/shadow/utils.go:72-112`): **`ACCEPT`** (`r3: YES, confidence=0.5112`).
  - `A-FIX-2` (`internal/service/shadow/schema_adapter.go:771-782`): cả 3 vòng đều `CORRECT / CORRECT / YES`, đã resolve kèm bằng chứng `EV-20261103T135657-b281` vào `04_decisions_same_updatedat_cdc_unversioned_diff.md`.
  - `A-HONESTY-3` (`test/internal/service/schema_adapter_ordering_test.go:15-21` dùng SQLite in-memory): **`ACCEPT`** (`r2: CORRECT 0.5899`, `r3: YES 0.8642`).
  - `A-SINKWORKER-4` (`internal/sinkworker/utils.go:88-104`): **`ACCEPT`** (`r2: CORRECT 0.4639`, `r3: YES 0.8571`).
  - **Tổng kết cổng `/laya_router`:** `DG0` (`PASS`), `DG1` (`PASS`), `DG2` (`PASS`), `DG3` (`PASS`).
