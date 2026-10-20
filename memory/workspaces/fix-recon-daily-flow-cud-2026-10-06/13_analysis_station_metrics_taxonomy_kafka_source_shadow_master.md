# Đặc Tả Thực Thể & Bộ Metrics Vận Hành Của 4 Trạm: Kafka, Bảng Source, Bảng Shadow, Bảng Master

- **Workspace:** `fix-recon-daily-flow-cud-2026-10-06`
- **Ngày cập nhật:** `2026-10-16`
- **Nguyên tắc kiến trúc cốt lõi (Bắt buộc):**
  1. **CẤM TUYỆT ĐỐI các tác vụ đếm tổng toàn bảng / toàn hệ thống (`COUNT(*)` toàn bảng như `source_total`, `source_active`, `full_source_count`, `window_source_count`, `shadow_total`, `master_total`):** Hệ thống CDC vận hành trên các bảng dữ liệu lớn không bao giờ quét tổng toàn bảng để làm metric. Mọi thông số đếm tổng toàn bảng đều vô giá trị về mặt giám sát thời gian thực và gây quá tải DB.
  2. **Bản chất của Prometheus Metrics (`:8082/metrics`):** Chỉ xuất các chỉ số **Time-Series phục vụ giám sát vận hành và cảnh báo theo thời gian**, gồm 2 nhóm duy nhất:
     - **Nhóm A — Tín hiệu Luồng & Sức khỏe Thời gian thực (Stream Throughput, Lag, Latency, Errors):** Tốc độ sự kiện chảy qua, độ trễ xử lý, số bản ghi lỗi/rơi rụng.
     - **Nhóm B — Số đếm Bản ghi theo Lát cắt Ngày (`Daily Sliced Count` theo `report_date = YYYY-MM-DD`):** Số lượng bản ghi phát sinh trong đúng ngày `YYYY-MM-DD` tại từng trạm để đối chiếu theo ngày, không quét toàn bộ lịch sử.
  3. **Không đoán mò tên cột:** Phải tách bạch rõ trong mã nguồn hiện tại mỗi thực thể (`Kafka`, `Bảng Source`, `Bảng Shadow`, `Bảng Master`) **thực sự đang lưu những thông số cấu hình/schema nào**, trạm nào đã có cột ngày tạo, trạm nào **chưa hề có chỗ cấu hình ngày tạo**.

---

## 1. TRẠM KAFKA (`Source Connector` $\rightarrow$ `Kafka Topic` $\rightarrow$ `KafkaConsumer`)

### 1.1. Một luồng Kafka trong hệ thống có những thông số định danh/cấu hình nào?
Theo [`SourceConnectors.tsx:85-131`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/SourceConnectors.tsx#L85-L131) và [`kafka_consumer.go:294-418`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/kafka_consumer.go#L294-L418):
- **`connector_name` (`source_connection_code`):** Mã định danh kết nối nguồn (ví dụ: `traitestpbs`, `traitestces`).
- **`source_type` (`dbKind`):** Loại DB nguồn (`mongodb`, `postgresql`, `mysql`, `sftp`).
- **`topic_prefix` & `topic`:** Tên Kafka topic chứa sự kiện CDC của bảng nguồn (ví dụ: `cdc.goopay.payment-bill-service.payment-bills`).
- **`partition` & `group_id`:** Phân vùng Kafka (`0, 1, ...`) và Consumer Group của Worker (`cdc-worker-group`).
- **Payload sự kiện Debezium trên Kafka:** Chứa `op` (`"c"` = tạo mới, `"u"` = cập nhật, `"d"` = xóa, `"r"` = đọc snapshot), `ts_ms` (thời điểm sự kiện), và `after` / `before` (dữ liệu BSON/JSON gốc).

### 1.2. Bộ Metrics của Kafka trên `:8082/metrics`
| Tên Metric trên `:8082/metrics` | Kiểu | Labels | Ý nghĩa vận hành thực tế | Mã nguồn |
| :--- | :--- | :--- | :--- | :--- |
| **`cdc_kafka_consumer_lag`** | Gauge | `topic`, `partition` | Số lượng message còn tồn đọng trong Kafka partition chưa được Worker đọc (chỉ báo nghẽn cổ chai). | [`kafka_consumer.go:294-296`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/kafka_consumer.go#L294-L296) |
| **`cdc_kafka_consumer_offset`** | Gauge | `topic`, `partition` | Vị trí offset hiện tại Worker vừa đọc tới trên partition. | [`kafka_consumer.go:297-299`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/kafka_consumer.go#L297-L299) |
| **`cdc_e2e_latency_seconds`** | Histogram | *(không)* | Độ trễ từ lúc sự kiện ghi vào Kafka (`msg.Time`) tới lúc Worker tiêu thụ. | [`kafka_consumer.go:415-418`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/kafka_consumer.go#L415-L418) |
| **`cdc_sink_events_total`** | Counter | `table`, `op` (`c`,`u`,`d`,`r`) | Lưu lượng sự kiện Debezium đọc từ Kafka phân loại theo thao tác Insert (`c`), Update (`u`), Delete (`d`), Snapshot (`r`). | [`event_handler.go:500-504`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/event_handler.go#L500-L504) |
| **`cdc_sink_events_dropped_total`** | Counter | `reason`, `topic` | Số sự kiện bị loại bỏ ngay ở cửa ngõ Kafka (do `source_not_registered` hoặc `missing_pk`). | [`event_handler.go:296,357`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/event_handler.go#L296) |
| **`cdc_events_processed_total`** | Counter | `operation`, `status` | Số lượng message Kafka xử lý thành công hoặc lỗi (`status="error"`) tại consumer. | [`kafka_consumer.go:471-473`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/kafka_consumer.go#L471-L473) |

---

## 2. TRẠM BẢNG SOURCE (1 Bảng / Collection Nguồn — Upstream DB)

### 2.1. Một Bảng Source trong hệ thống thực sự có những thông số nào?
Theo [`source_object_registry.go:8-49`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/model/source/source_object_registry.go#L8-L49), [`table_registry.go:9-74`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/model/source/table_registry.go#L9-L74) và Form đăng ký/chỉnh sửa tại [`TableRegistry.tsx:1647-1801`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/TableRegistry.tsx#L1647-L1801), một Bảng Source **CHỈ CÓ** các thông số cấu hình sau:
1. **Định danh nguồn:** `source_connection_id` (`source_connection_code`), `source_engine_type` (`mongodb` / `postgresql`), `source_database` / `source_schema`, `source_object_name` (`source_table`).
2. **Khóa chính (Primary Key):** `primary_key_field` (mặc định `_id` với MongoDB, `id` với PostgreSQL) và `primary_key_type` (`VARCHAR(24)`, `BIGINT`).
3. **Trường thời gian cập nhật (`timestamp_field`):** Mặc định là `lastUpdatedAt` ([`TableRegistry.tsx:1648,1734-1739`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/TableRegistry.tsx#L1734-L1739)). Đây là trường thời gian **cập nhật (`updatedAt` / `lastUpdatedAt`)** dùng cho reconciliation window, **KHÔNG PHẢI cột ngày tạo bản ghi (`created_at`)**.
4. **Thông số điều khiển đồng bộ:** `is_active` (bật/tắt Debezium sync), `priority` (`critical`/`high`/`normal`/`low`), `snapshot_batch_size`, `snapshot_max_rps`.

> [!WARNING]
> **Sự thật trong Code — Lỗ hổng Thiết kế tại Bảng Source:**
> Bảng Source (`source_object_registry` và `cdc_table_registry`) **HOÀN TOÀN CHƯA CÓ thông số cấu hình cột ngày tạo (`created_at_field` / `creation_date_field`)!**
> Chính vì Bảng Source không có nơi khai báo cột ngày tạo trong Registry, nên khi lập lịch đếm theo ngày ở [`ActivityManager.tsx:221-234`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/ActivityManager.tsx#L221-L234), hệ thống đang phải **đoán mò** bằng Regex (`created_at`, `createdAt`, `create_time`...) trên danh sách field đã scan hoặc bắt người dùng tự gõ tay `date_field`. Nếu bảng Source chưa khai báo rõ cột ngày tạo (và kiểu dữ liệu của cột đó trên Source: `ISODate`, `Epoch ms`, hay `TIMESTAMPTZ`), việc đếm Bảng Source theo ngày là đếm mù!

### 2.2. Bộ Metrics của Bảng Source trên `:8082/metrics` (Không quét tổng toàn bảng)
| Tên Metric | Kiểu | Labels | Ý nghĩa vận hành thực tế | Hiện trạng trên `:8082/metrics` |
| :--- | :--- | :--- | :--- | :--- |
| **`cdc_source_query_duration_seconds`** | Histogram | `role="source"` | Độ trễ khi thực thi truy vấn vào DB nguồn. | 🟢 Đã có ([`prometheus.go:210-217`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/pkgs/metrics/prometheus.go#L210-L217)) |
| **`cdc_per_source_pool_in_use`** | Gauge | `source_code` | Số kết nối đang sử dụng đồng thời trên từng DB nguồn (bảo vệ không làm sập nguồn). | 🟢 Đã có ([`prometheus.go:198-204`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/pkgs/metrics/prometheus.go#L198-L204)) |
| **Số bản ghi phát sinh trong ngày (`Daily Source Count`)** | Gauge | `report_date`, `connection_code`, `source_db`, `source_table`, `date_field` | Số bản ghi được tạo mới trên Bảng Source trong đúng ngày `report_date` (`00:00` $\rightarrow$ `24:00` GMT+7) theo cột ngày tạo đã khai báo. | 🔴 **Chưa đẩy ra `:8082/metrics`** & **Bảng Source chưa có trường khai báo cột ngày tạo chính thức trong Registry** |

---

## 3. TRẠM BẢNG SHADOW (1 Bảng Shadow — PostgreSQL Staging Layer)

### 3.1. Một Bảng Shadow trong hệ thống có những thông số nào?
Theo [`shadow_binding.go:8-48`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/model/source/shadow_binding.go#L8-L48), [`TableRegistry.tsx:1401-1479`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/TableRegistry.tsx#L1401-L1479) và [`MappingFieldsPage.tsx:29-92`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/MappingFieldsPage.tsx#L29-L92), một Bảng Shadow gồm 3 lớp thông số:
1. **Thông số Định danh & Routing (`shadow_binding`):**
   - `binding_code`, `source_object_id` (nối về Bảng Source), `shadow_connection_id` (`default_shadow`), `shadow_schema` (quy ước `shadow_<source_db>`), `shadow_table` (`target_table`), `write_mode` (`upsert` hoặc `explode` mảng con qua `explode_path`), `ddl_status` (`pending`/`created`/`drifted`/`failed`), `is_active`.
2. **11 Cột Hệ Thống Mặc Định (`SYSTEM_DEFAULT_FIELDS`) luôn gắn trên mọi bảng vật lý Shadow** ([`MappingFieldsPage.tsx:29-41`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/MappingFieldsPage.tsx#L29-L41)):
   - Định danh & Dữ liệu gốc: `_gpay_id` (`BIGINT` Sonyflake PK), `_source_id` (`TEXT` UNIQUE khóa chính từ nguồn), `_raw_data` (`JSONB` toàn bộ payload gốc từ Source), `_source` (`VARCHAR`).
   - Chống xung đột & Xóa mềm: `_source_ts` (`BIGINT` timestamp sự kiện nguồn dùng cho OCC guard), `_version` (`BIGINT`), `_hash` (`VARCHAR` SHA256 dedup), `_deleted` (`BOOLEAN` cờ xóa mềm).
   - Thời gian hệ thống DW: `_synced_at` (`TIMESTAMPTZ`), **`_created_at` (`TIMESTAMPTZ` — thời điểm bản ghi được insert vào DW Shadow)**, **`_updated_at` (`TIMESTAMPTZ` — thời điểm bản ghi được cập nhật tại DW Shadow)**.
3. **Các Cột Nghiệp Vụ Đã Duyệt (`mapping_rule_v2`):**
   - Các cột được bóc từ `_raw_data` (`source_field` $\rightarrow$ `target_column`, `data_type`, `source_format`, `status = 'approved'`).
   - **Điểm cực kỳ quan trọng về Ngày Tạo trên Bảng Shadow:** Cột hệ thống `_created_at` của Shadow là **thời điểm Worker ghi dòng đó vào Shadow** (nếu chạy Snapshot nạp dữ liệu cũ của tháng trước thì `_created_at` vẫn mang ngày hôm nay!). Do đó, muốn đếm số bản ghi theo **ngày tạo nghiệp vụ gốc** để khớp với Source và Master, Bảng Shadow phải đếm trên **cột ngày tạo nghiệp vụ đã được bóc ra từ `_raw_data`** (ví dụ cột `created_at` được map từ `after.createdAt`), chứ không phải đếm mù trên `_created_at` hệ thống khi có chạy Snapshot!

### 3.2. Bộ Metrics của Bảng Shadow trên `:8082/metrics`
| Tên Metric | Kiểu | Labels | Ý nghĩa vận hành thực tế | Hiện trạng trên `:8082/metrics` |
| :--- | :--- | :--- | :--- | :--- |
| **`cdc_sync_success_total`** | Counter | `table`, `operation`, `source` | Số dòng thực sự được `BatchBuffer` ghi (upsert/delete) thành công xuống bảng vật lý Shadow. | 🟢 Đã có ([`batch_buffer.go:845`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/batch_buffer.go#L845)) |
| **`cdc_sync_failed_total`** | Counter | `table`, `operation`, `source` | Số dòng ghi vào bảng Shadow bị lỗi. | 🟢 Đã có ([`batch_buffer.go:836`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/batch_buffer.go#L836)) |
| **`cdc_dlq_depth`** | Gauge | *(không)* | Số lượng bản ghi lỗi đang nằm chờ trong bảng Dead Letter Queue (`cdc_failed_sync_logs`). | 🟢 Đã có ([`prometheus.go:79-83`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/pkgs/metrics/prometheus.go#L79-L83)) |
| **`cdc_batch_buffer_size`** | Gauge | `table` | Số lượng record đang chờ flush trong bộ nhớ đệm của từng bảng Shadow. | 🟢 Đã có ([`prometheus.go:53-57`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/pkgs/metrics/prometheus.go#L53-L57)) |
| **`cdc_db_write_duration_seconds`** | Histogram | `table` | Thời gian thực thi câu lệnh SQL bulk write xuống DB Shadow. | 🟢 Đã có ([`prometheus.go:44-50`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/pkgs/metrics/prometheus.go#L44-L50)) |
| **`cdc_recon_ingest_lag_ms`** | Gauge | `table` | Độ trễ đồng bộ (ms) giữa bản ghi mới nhất ở Source và bản ghi mới nhất ở Shadow. | 🟢 Đã có ([`recon_smoke.go:418`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_smoke.go#L418)) |
| **Số bản ghi phát sinh trong ngày (`Daily Shadow Count`)** | Gauge | `report_date`, `connection_code`, `shadow_schema`, `shadow_table`, `date_field` | Số bản ghi (`_deleted = false`) thuộc ngày `report_date` (`00:00` $\rightarrow$ `24:00` GMT+7) trên Bảng Shadow. | 🔴 **Chưa đẩy ra `:8082/metrics`** |

---

## 4. TRẠM BẢNG MASTER (1 Bảng Master — PostgreSQL Business Target Layer)

### 4.1. Một Bảng Master trong hệ thống có những thông số nào?
Theo [`master_binding.go:8-48`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/model/source/master_binding.go#L8-L48), [`MasterRegistry.tsx:288-296,853-910`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/MasterRegistry.tsx#L288-L296) và [`MasterMappingFieldsPage.tsx:13-52`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/MasterMappingFieldsPage.tsx#L13-L52), một Bảng Master gồm 4 lớp thông số:
1. **Thông số Định danh & Nguồn (`master_binding`):**
   - `binding_code`, `master_connection_code` (kết nối PostgreSQL đích, vd `default` hoặc `master_2`), `master_schema` (quy ước `master_<source_db>`), `master_name` (`master_table`), liên kết từ `shadow_binding_id` (`shadow_schema`.`shadow_table`).
2. **Thông số Biến đổi & Lịch đồng bộ (Transmute Config):**
   - `transform_type` (`copy_1_to_1` giữ nguyên 1-1 từ Shadow hoặc `flatten` bóc mảng JSON thành nhiều dòng qua `explode_path`), `spec` (JSON chứa `{"pk":"_id"}` hoặc `{"pk":"_source_id","explode_path":"items[*]"}`), `schema_status` (`pending_review` / `approved` / `rejected`), `is_active`.
   - Chế độ kích hoạt (`transmute_schedules`): **Realtime (`post_ingest`)** tự bắn sang Master ngay khi Shadow nhận dữ liệu mới, hoặc **Hẹn giờ (`cron`)**.
3. **Các Cột Hệ Thống trên Bảng Vật Lý Master:**
   - `_gpay_id` (`BIGINT` PK), `_source_ts` (`BIGINT` mốc thời gian nguồn để kiểm tra OCC `EXCLUDED._source_ts >= master._source_ts`), `_deleted` (`BOOLEAN` cờ xóa mềm), `_updated_at` (`TIMESTAMPTZ` thời điểm ghi vào Master).
   - **Lưu ý:** Bảng Master **KHÔNG CÓ** cột hệ thống `_created_at` và **KHÔNG CÓ** cột `_raw_data`.
4. **Các Cột Nghiệp Vụ Đã Duyệt (`master_mapping_rules`):**
   - Chỉ những cột được chọn và `approved` (`in_master = true`) từ Shadow mới được tạo cột vật lý trên bảng Master (`target_column`, `data_type`, `transform_fn`).
   - Do đó, muốn đếm số bản ghi theo ngày trên Bảng Master, bắt buộc phải đếm trên **cột ngày tạo nghiệp vụ đã được ánh xạ sang Master** (ví dụ `created_at` được nối từ `mapping_v2_id` của Shadow).

### 4.2. Bộ Metrics của Bảng Master trên `:8082/metrics`
| Tên Metric | Kiểu | Labels | Ý nghĩa vận hành thực tế | Hiện trạng trên `:8082/metrics` |
| :--- | :--- | :--- | :--- | :--- |
| **`cdc_transmute_ops_total`** | Counter | `master_table`, `op`, `trigger_type` | Lưu lượng dòng xử lý từ Shadow sang Master theo từng loại kết quả: `op="inserted"` (dòng thêm mới), `op="updated"` (dòng cập nhật), `op="skipped"` (bỏ qua do OCC `_source_ts` cũ hơn), `op="rule_miss"`, `op="type_error"`. | 🟢 Đã có ([`transmute_handler.go:568-572`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/transmute_handler.go#L568-L572)) |
| **`cdc_transmute_error_total`** | Counter | `master_table`, `stage` | Số lần job Transmute cho bảng Master gặp lỗi thực thi. | 🟢 Đã có ([`transmute_handler.go:519`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/transmute_handler.go#L519)) |
| **`cdc_transmute_duration_ms`** | Histogram | `master_table` | Thời gian thực thi (ms) của mỗi lần chạy Transmute từ Shadow sang Master. | 🟢 Đã có ([`transmute_handler.go:573`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/transmute_handler.go#L573)) |
| **Số bản ghi phát sinh trong ngày (`Daily Master Count`)** | Gauge | `report_date`, `connection_code`, `master_schema`, `master_table`, `date_field` | Số bản ghi (`_deleted = false`) thuộc ngày `report_date` (`00:00` $\rightarrow$ `24:00` GMT+7) trên Bảng Master theo cột ngày tạo nghiệp vụ. | 🔴 **Chưa đẩy ra `:8082/metrics`** |

---

## 5. KẾT LUẬN CỐT LÕI: TẠI SAO ĐẾM THEO NGÀY ĐANG SAI TỪ GỐC?

Khi đặt 4 trạm (`Kafka`, `Bảng Source`, `Bảng Shadow`, `Bảng Master`) cạnh nhau, nguyên nhân gốc rễ khiến việc đếm bảng theo ngày và đẩy ra `:8082/metrics` chưa chính xác nằm ở **3 điểm đứt gãy cấu trúc**:

1. **Bảng Source chưa hề có cấu hình Cột Ngày Tạo (`Creation Date Field`):**
   - Trong `source_object_registry` ([`source_object_registry.go:8-49`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/model/source/source_object_registry.go#L8-L49)) chỉ có `primary_key_field` (`_id`) và `timestamp_field` (`lastUpdatedAt` dùng cho cửa sổ cập nhật). Hoàn toàn không có chỗ nào xác định cột ngày tạo của Bảng Source, dẫn tới phải đoán mò khi đếm.
2. **Sự lệch pha giữa Cột Ngày Tạo Nghiệp Vụ và Cột `_created_at` Hệ Thống của Shadow:**
   - Bảng Source đếm theo trường ngày tạo gốc (vd `createdAt`), Bảng Master đếm theo cột nghiệp vụ đã map (vd `created_at`), nhưng nếu Bảng Shadow lại mặc định đếm theo `_created_at` (thời điểm Worker ghi vào DB Shadow) thì mỗi khi chạy Snapshot/Backfill dữ liệu cũ, số đếm theo ngày của Shadow sẽ vọt lên sai lệch hoàn toàn so với Source và Master.
   - Đúng bản chất: Cột ngày tạo phải được định nghĩa từ **Bảng Source** $\rightarrow$ ánh xạ qua `mapping_rule_v2` sang **cột tương ứng trên Bảng Shadow** $\rightarrow$ ánh xạ qua `master_mapping_rules` sang **cột tương ứng trên Bảng Master**. Khi đó cả 3 bảng mới đếm trên **cùng 1 trường thời gian tạo bản ghi**.
3. **Trên `:8082/metrics` đang hoàn toàn vắng bóng Metric Đếm Theo Ngày (`Daily Sliced Count`), trong khi lại tồn tại các Metric đếm tổng toàn bảng (`cdc_source_table_row_count`, `cdc_shadow_table_row_count`, `cdc_master_table_row_count`):**
   - Các metric đếm tổng toàn bảng không phục vụ đối soát theo ngày và vi phạm nguyên tắc không làm tác vụ tổng toàn hệ thống.
   - Kết quả đếm theo ngày trong `cdc_system.daily_source_counts` lại chưa được экспоrt ra `:8082/metrics`.

