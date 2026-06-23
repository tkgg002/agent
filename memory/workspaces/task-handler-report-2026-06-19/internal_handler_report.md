# BÁO CÁO CHI TIẾT CẤU TRÚC VÀ CHỨC NĂNG CÁC HANDLER
**Thư mục:** `internal/handler`
**Dự án:** `centralized-data-service`

Báo cáo này cung cấp thông tin chi tiết về từng thư mục con, danh sách file, các struct, hàm/handler chính và mô tả chức năng chi tiết của từng thành phần trong lớp Handler.

---

## Sơ đồ cấu trúc thư mục `internal/handler`

```
internal/handler/
├── base/
│   ├── base_handler.go
│   └── provisioning_emit.go
├── shadow/
│   ├── batch_buffer.go
│   ├── batch_transform_handler.go
│   ├── consumer_pool.go
│   ├── dlq_circuit_breaker.go
│   ├── event_bridge.go
│   ├── event_handler.go
│   └── kafka_consumer.go
├── master/
│   ├── master_ddl_handler.go
│   ├── schema_ddl_handler.go
│   └── transmute_handler.go
├── orchestration/
│   ├── discover_handler.go
│   ├── dlq_handler.go
│   ├── dlq_state_machine.go
│   ├── mongo_discover_handler.go
│   ├── provisioning_emit.go
│   ├── provisioning_handler.go
│   ├── provisioning_step_handlers.go
│   ├── scan_handler.go
│   └── snapshot_runner_handler.go
├── recon/
│   ├── recon_handler.go
│   └── recon_heal_v4.go
└── source/
    ├── source_register.go
    └── sync_handler.go
```

---

## 1. Thư mục `base/` (Base Handler & Helpers)
Thư mục này chứa các logic dùng chung cho tất cả các handler trong hệ thống cdc, cung cấp các kết nối cơ sở dữ liệu, NATS connection và các phương thức tiện ích SQL.

### [base_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/base/base_handler.go)
*   **Struct chính:** `BaseHandler`
*   **Chức năng:** Struct cơ sở để các handler khác nhúng (struct embedding). Quản lý kết nối PostgreSQL (`DB`), NATS (`NatsConn`) và `Logger` (Zap).
*   **Các phương thức chính:**
    *   `TableExists(ctx, db, table)`: Kiểm tra xem bảng có tồn tại trong schema hiện tại không.
    *   `HasColumn(ctx, db, table, column)`: Kiểm tra sự tồn tại của một cột trong bảng.
    *   `PublishResult(...)` & `PublishResultWithSubject(...)`: Đóng gói kết quả xử lý (`CommandResult`) thành JSON và gửi ngược lại NATS.
    *   `ConnectGET(...)` / `ConnectPOST(...)` / `ConnectPUT(...)`: Các phương thức helper thực hiện HTTP Client request có timeout để tránh Goroutine leak.
    *   `ResolveTargetSchema(tableName)`: Phân tích và quyết định schema tương ứng (`cdc_dw` hoặc `cdc_shadow`).

### [provisioning_emit.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/base/provisioning_emit.go)
*   **Struct chính:** `StepResult`, `stepCompletedPayload`
*   **Chức năng:** Cung cấp phương thức phát sự kiện hoàn thành các bước thiết lập cấu hình lên NATS JetStream.
*   **Các phương thức chính:**
    *   `EmitStepCompleted(ctx, StepResult)`: Đóng gói kết quả của một bước provisioning (bao gồm `SourceID`, `Step`, `Error` được làm sạch qua `MaskingService`, `TraceID`, `SpanID`) và publish vào subject `cdc.evt.provisioning.step_completed` để điều phối máy trạng thái (state machine).

---

## 2. Thư mục `shadow/` (Shadow Data Ingestion)
Thư mục này chịu trách nhiệm trực tiếp thu nhận dữ liệu thô từ Kafka / NATS, thực hiện chuyển đổi sơ bộ và ghi dữ liệu dạng JSONB vào các bảng shadow.

### [batch_buffer.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/batch_buffer.go)
*   **Struct chính:** `BatchBuffer`, `bufferItem`
*   **Chức năng:** Buffer gom cụm ghi loạt (batching) các thao tác ghi dữ liệu xuống Postgres. Giảm tải tần suất ghi và tránh lock bảng khi có lưu lượng CDC lớn.
*   **Các phương thức chính:**
    *   `Add(db, query, values)`: Đưa câu lệnh SQL kèm tham số vào hàng đợi.
    *   `Flush()`: Duyệt qua hàng đợi và thực thi gộp toàn bộ câu lệnh trong một transaction duy nhất. Có cơ chế tự động hạ cấp ghi đơn lẻ (fallback to single rows) nếu ghi gộp bị lỗi để cô lập dòng lỗi.
    *   `StartPeriodicFlush(interval)`: Khởi chạy worker chạy ngầm tự động kích hoạt `Flush` theo chu kỳ thời gian.

### [batch_transform_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/batch_transform_handler.go)
*   **Struct chính:** `BatchTransformHandler`
*   **Chức năng:** Handler xử lý lệnh đồng bộ cấu trúc cột thô sang cột chuẩn hóa ở tầng shadow (`cdc.cmd.batch-transform`).
*   **Các phương thức chính:**
    *   `HandleTransform(msg)`: Chuyển đổi hàng loạt các bản ghi có cột `_raw_data` chưa được bóc tách sang dạng cột rời dựa trên các quy tắc mapping hiện có. Tiến hành cập nhật theo từng chunk (mặc định 1000 dòng).

### [consumer_pool.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/consumer_pool.go)
*   **Struct chính:** `ConsumerPool`
*   **Chức năng:** Quản lý vòng đời và số lượng các worker kéo (pull subscriptions) tin nhắn từ các NATS JetStream stream/consumer.
*   **Các phương thức chính:**
    *   `Start(ctx)`: Khởi chạy các goroutine subscriber song song để xử lý tin nhắn CDC.
    *   `Stop()`: Dừng an toàn toàn bộ subscriber, chờ xử lý nốt tin nhắn đang dở dang (graceful shutdown).

### [dlq_circuit_breaker.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/dlq_circuit_breaker.go)
*   **Struct chính:** `DLQCircuitBreaker`
*   **Chức năng:** Cắt mạch tự động (circuit breaker) để tạm dừng pipeline CDC nếu phát hiện số lượng lỗi ghi/đồng bộ đẩy vào DLQ vượt quá ngưỡng an toàn được định cấu hình. Ngăn chặn việc làm nghẽn DLQ hoặc tràn đĩa khi có lỗi hệ thống hoặc schema drift diện rộng.

### [event_bridge.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/event_bridge.go)
*   **Struct chính:** `EventBridge`
*   **Chức năng:** Cầu nối trung chuyển dữ liệu từ hệ thống triggers PostgreSQL cũ sang hệ thống CDC hiện đại. 
*   **Các phương thức chính:**
    *   `HandleTriggerEvent(msg)`: Lắng nghe các trigger events cũ từ NATS, định dạng lại thành cấu trúc CDC tiêu chuẩn và gửi tiếp vào pipeline xử lý sự kiện mới.

### [event_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/event_handler.go)
*   **Struct chính:** `EventHandler`
*   **Chức năng:** Điều phối xử lý sự kiện CDC thô sau khi được parse. Phục vụ định tuyến ghi dữ liệu vào bảng shadow thích hợp.
*   **Các phương thức chính:**
    *   `HandleRaw(ctx, subject, data)`: Nhận event payload từ Kafka consumer hoặc snapshot runner. Thực hiện bóc tách metadata, kiểm tra schema, tạo câu lệnh SQL INSERT/UPDATE tương ứng và đưa vào `BatchBuffer`.
    *   `FlushBatchBuffer()`: Ép ghi toàn bộ dữ liệu đang chờ trong buffer xuống database.

### [kafka_consumer.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/kafka_consumer.go)
*   **Struct chính:** `KafkaConsumer`, `avroMessageDecoder`
*   **Chức năng:** Consumer chính kết nối tới Apache Kafka. Giải mã các bản ghi CDC được mã hóa bằng định dạng Avro (qua Schema Registry) hoặc định dạng JSON thô (từ Debezium).
*   **Các phương thức chính:**
    *   `Start(ctx)`: Đăng ký nhóm tiêu thụ (consumer group) và lắng nghe các topic CDC được cấu hình.
    *   `processMessage(msg)`: Đọc bytes dữ liệu, phát hiện định dạng (Avro magic byte vs JSON), giải mã và đẩy kết quả sang `EventHandler`.

---

## 3. Thư mục `master/` (Master Table Operations)
Thư mục này quản lý cấu trúc bảng dữ liệu master (tầng kho dữ liệu chuẩn hóa cuối cùng), cập nhật cấu trúc cột và kích hoạt chuyển đổi dữ liệu từ shadow sang master.

### [master_ddl_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/master_ddl_handler.go)
*   **Struct chính:** `MasterDDLHandler`
*   **Chức năng:** Xử lý các lệnh DDL liên quan trực tiếp tới tầng master bảng đích.
*   **Các phương thức chính:**
    *   `HandleMasterCreate(msg)`: Lắng nghe `cdc.cmd.master-create` / `cdc.cmd.master.bind`. Thực hiện tạo bảng master (nếu chưa có) và thiết lập binding liên kết.
    *   `HandleMasterAlterColumn(msg)`: Lắng nghe `cdc.cmd.master-alter-column`. Thực hiện sửa đổi định dạng/kiểu dữ liệu cột trong bảng master.
    *   `HandleMasterSwap(msg)`: Lắng nghe `cdc.cmd.master-swap`. Thực hiện swap (tráo đổi) bảng master nguyên tử (atomic swap) sử dụng transaction để đưa bảng cấu trúc mới thay thế bảng cũ không gây gián đoạn truy vấn.

### [schema_ddl_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/schema_ddl_handler.go)
*   **Struct chính:** `SchemaDDLHandler`
*   **Chức năng:** Quản lý cấu trúc DDL tầng shadow và các cột mặc định.
*   **Các phương thức chính:**
    *   `HandleStandardize(msg)`: Lắng nghe `cdc.cmd.standardize`. Chuẩn hóa cấu trúc bảng shadow.
    *   `HandleCreateDefaultColumns(msg)`: Lắng nghe `cdc.cmd.create-default-columns`. Tự động thêm các cột metadata mặc định (`_gpay_id`, `_raw_data`, `_synced_at`, `_source_ts`, `_hash_value`, `_deleted_at`) vào bảng shadow. Tự động khắc phục kiểu dữ liệu bị lệch (type drift) bằng cách chạy `ALTER TABLE`.
    *   `HandleDropGINIndex(msg)`: Lắng nghe `cdc.cmd.drop-gin-index`. Xóa chỉ mục GIN trên cột `_raw_data` sau khi hoàn tất các tác vụ đồng bộ để tối ưu tốc độ ghi.

### [transmute_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/transmute_handler.go)
*   **Struct chính:** `TransmuteHandler`
*   **Chức năng:** Điều phối tiến trình dịch chuyển, làm sạch và ghi dữ liệu từ bảng shadow sang bảng master (Transmutation).
*   **Các phương thức chính:**
    *   `HandleTransmute(msg)`: Lắng nghe `cdc.cmd.transmute`. Nhận yêu cầu và kích hoạt module `TransmuterModule` để xử lý dữ liệu shadow mới.
    *   `HandleTransmuteShadow(msg)`: Lắng nghe `cdc.cmd.transmute-shadow`. Chạy tiến trình transmutation định hướng cho từng binding cụ thể từ tầng shadow.

---

## 4. Thư mục `orchestration/` (Pipeline Orchestration & Auto-Provisioning)
Thư mục này quản lý toàn bộ vòng đời thiết lập tự động (Auto-Provisioning) cho một nguồn dữ liệu mới, điều khiển máy trạng thái (state machine) và xử lý lỗi / DLQ.

### [discover_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/orchestration/discover_handler.go)
*   **Struct chính:** `DiscoverHandler`
*   **Chức năng:** Xử lý tự động phát hiện schema từ dữ liệu nguồn.
*   **Các phương thức chính:**
    *   `HandleDiscover(msg)`: Lắng nghe `cdc.cmd.discover`. Kích hoạt bộ quét trường (`FieldScanner`) để lấy mẫu dữ liệu và phân tích kiểu trường.
    *   `ScanFieldsDebezium(...)`: Quét mẫu dữ liệu CDC Debezium hiện có trong shadow table để suy luận ra các quy tắc ánh xạ mapping tự động.
    *   `scanFieldsMongoSource(...)`: Quét phân tích trực tiếp từ MongoDB.

### [dlq_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/orchestration/dlq_handler.go)
*   **Struct chính:** `DLQHandler`
*   **Chức năng:** Tiếp nhận các tin nhắn lỗi xử lý từ shadow/master, lưu trữ phục vụ việc retry sau này.
*   **Các phương thức chính:**
    *   `HandleDLQ(msg)`: Lắng nghe `cdc.dlq`. Thực hiện che giấu dữ liệu nhạy cảm (data masking) qua `MaskingService`, sau đó ghi bản ghi lỗi vào bảng `failed_sync_logs` kèm thông tin lỗi và metadata retry.

### [dlq_state_machine.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/orchestration/dlq_state_machine.go)
*   **Struct chính:** `DLQStateMachine`
*   **Chức năng:** Worker chạy ngầm định kỳ quét bảng `failed_sync_logs` để tự động replay (thử lại) các tin nhắn bị lỗi.
*   **Các phương thức chính:**
    *   `Start(ctx)`: Khởi chạy chu kỳ quét.
    *   `pollAndReplay()`: Truy vấn các dòng tin nhắn lỗi có trạng thái `pending` hoặc đến hạn retry (sử dụng exponential backoff: `1m`, `5m`, `30m`, `2h`, `6h` qua hàm `nextReplayDelay`), sau đó publish lại lên NATS để xử lý lại.

### [mongo_discover_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/orchestration/mongo_discover_handler.go)
*   **Struct chính:** `MongoDiscoverHandler`
*   **Chức năng:** Trả lời các truy vấn khám phá nhanh tài nguyên của MongoDB.
*   **Các phương thức chính:**
    *   `HandleListDatabases(msg)`: Truy vấn danh sách các database khả dụng trên Mongo.
    *   `HandleListCollections(msg)`: Truy vấn danh sách các collection thuộc database cụ thể.

### [provisioning_emit.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/orchestration/provisioning_emit.go)
*   **Hàm chính:** `emitStepCompleted(...)`
*   **Chức năng:** Hàm helper độc lập trong package để gửi trạng thái hoàn thành các bước của orchestration lên NATS topic `cdc.evt.provisioning.step_completed`.

### [provisioning_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/orchestration/provisioning_handler.go)
*   **Struct chính:** `ProvisioningHandler`
*   **Chức năng:** NATS subscriber điều hướng sự kiện hoàn thành bước trung gian vào Core Orchestrator.
*   **Các phương thức chính:**
    *   `HandleStepCompleted(msg)`: Lắng nghe `cdc.evt.provisioning.step_completed`, chuyển đổi payload và gọi `ProvisioningOrchestrator.HandleStepCompleted` để dịch chuyển trạng thái máy. Xử lý an toàn các xung đột đồng thời (concurrency conflict).

### [provisioning_step_handlers.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/orchestration/provisioning_step_handlers.go)
*   **Struct chính:** `ProvisioningStepHandler`, `shadowBindRequest`, `scheduleEnableRequest`
*   **Chức năng:** Worker thực thi logic cụ thể của các bước provisioning (shadow_bind và schedule_enable).
*   **Các phương thức chính:**
    *   `HandleShadowBind(msg)`: Lắng nghe `cdc.cmd.shadow.bind`. Suy luận cấu trúc cột nguồn (Postgres/MySQL/MongoDB), tự động tạo bảng shadow có sẵn các cột nghiệp vụ bằng `SchemaAdapter` và lưu thông tin đăng ký vào bảng `cdc_system.shadow_binding`. Với MongoDB, có cơ chế kiểm tra trước (preflight check) xem collection có rỗng không để tránh tạo schema ảo.
    *   `HandleScheduleEnable(msg)`: Lắng nghe `cdc.cmd.schedule.enable`. Kích hoạt schedule dịch chuyển dữ liệu trong bảng `cdc_system.transmute_schedule` sang hoạt động (`is_enabled=true`).
    *   Các hàm nội bộ trợ giúp suy luận kiểu cột: `inferSourceColumns`, `inferPGCols`, `inferMySQLCols`, `inferMongoCols`.

### [scan_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/orchestration/scan_handler.go)
*   **Struct chính:** `ScanHandler`
*   **Chức năng:** Xử lý quét dữ liệu lịch sử thô để trích xuất cấu trúc và thực hiện bulk updates dữ liệu cũ.
*   **Các phương thức chính:**
    *   `HandleBackfill(msg)`: Lắng nghe `cdc.cmd.backfill`. Quét dữ liệu cũ có `_raw_data` chưa được bóc tách và chạy truy vấn `UPDATE` hàng loạt dùng lệnh `CAST` SQL của Postgres để ghi dữ liệu vào các cột thực tế.
    *   `HandleScanRawData(msg)`: Lắng nghe `cdc.cmd.scan-raw-data`. Lấy mẫu các bản ghi trong trường `_raw_data` của shadow table để tìm kiếm các key mới xuất hiện chưa được cấu hình cột.
    *   `HandleScanArrayFields(msg)`: Lắng nghe `cdc.cmd.scan-array-fields`. Xác định các trường có kiểu dữ liệu là mảng (JSON array) trong `_raw_data`.
    *   `HandlePeriodicScan(msg)`: Kích hoạt quét tuần kỳ toàn bộ các bảng trong hệ thống để tìm kiếm schema drift.

### [snapshot_runner_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/orchestration/snapshot_runner_handler.go)
*   **Struct chính:** `SnapshotRunner`, `snapshotV2Payload`, `progressClaim`
*   **Chức năng:** Thực thi nạp dữ liệu ban đầu dạng snapshot chỉ đọc (Path B) cho MongoDB. Tránh ghi tài liệu watermark/signal vào database nguồn để bảo vệ quyền hạn chỉ đọc (read-only).
*   **Các phương thức chính:**
    *   `Handle(msg)`: Tiếp nhận lệnh `cdc.cmd.snapshot.v2`, phân tích payload và tách tiến trình chạy snapshot sang một goroutine riêng biệt.
    *   `runSnapshot(...)`: Tiến trình nạp snapshot chính. Thực hiện kết nối Mongo phụ (SecondaryPreferred), lấy clusterTime, quét tuần tự các bản ghi theo thứ tự `_id` từ checkpoint cũ (`last_seen_id`), sinh CDC envelope giả lập Debezium và đẩy qua `EventHandler` để chạy qua pipeline chuẩn.
    *   Worker có tích hợp các cơ chế bảo vệ cao cấp: cắt mạch (circuit breaker) dừng snapshot nếu tỷ lệ lỗi hoặc số lỗi liên tiếp vượt ngưỡng; điều tiết tốc độ đọc bằng `SnapshotMaxRPS`; lưu vết tiến trình vào bảng `snapshot_progress`.

---

## 5. Thư mục `recon/` (Data Reconciliation & Self-Healing)
Thư mục này chịu trách nhiệm đối soát dữ liệu chéo giữa nguồn và đích, phát hiện lệch dữ liệu (mismatch/orphan) và tự động kích hoạt sửa lỗi (self-healing).

### [recon_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_handler.go)
*   **Struct chính:** `ReconHandler`
*   **Chức năng:** Điểm nhận lệnh đối soát, phát hiện sai khác và điều phối sửa lỗi.
*   **Các phương thức chính:**
    *   `HandleReconCheck(msg)`: Lắng nghe `cdc.cmd.recon-check`. Thực hiện so khớp số lượng bản ghi (Tier 1/2/3 cho Segment A, hoặc Segment B shadow <-> master). Hỗ trợ lệnh `prune` dọn dẹp các dòng shadow mồ côi (không còn ở source).
    *   `HandleReconHeal(msg)`: Lắng nghe `cdc.cmd.recon-heal`. Điều phối sửa lỗi dữ liệu bị lệch của Segment A hoặc B.
    *   `HandleRetryFailed(msg)`: Lắng nghe `cdc.cmd.retry-failed`. Thử lại thủ công một bản ghi lỗi trong bảng `failed_sync_logs`.
    *   `HandleDebeziumSignal(msg)`: Lắng nghe `cdc.cmd.debezium-signal` / `cdc.cmd.debezium-snapshot`. Gửi tín hiệu chụp lại snapshot tăng dần tới Kafka Connect qua `DebeziumSignalClient`, sau đó kiểm tra sức khỏe của connector để báo cáo lỗi thực tế.
    *   `HandleBackfillSourceTs(msg)`: Lắng nghe `cdc.cmd.recon-backfill-source-ts`. Chạy backfill đồng bộ thời gian nguồn sử dụng `BackfillSourceTsService`.
    *   `HandleDetectTimestampField(msg)`: Lắng nghe `cdc.cmd.detect-timestamp-field`. Tự động phát hiện trường lưu thời gian cập nhật của bản ghi Mongo để tối ưu hóa đối soát.

### [recon_heal_v4.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_heal_v4.go)
*   **Hàm/Phương thức chính:**
    *   `healSegmentA(...)`: Tự động sửa lỗi Segment A (source <-> shadow) bằng cách lấy danh sách các ID bị thiếu từ báo cáo đối soát, chia nhỏ thành các chunk và gửi tín hiệu yêu cầu Debezium re-emit sự kiện qua NATS `cdc.cmd.debezium-signal`. Tuyệt đối không update trực tiếp vào db nguồn.
    *   `healSegmentB(...)`: Tự động sửa lỗi Segment B (shadow <-> master) bằng cách map các ID bị thiếu sang `_source_id` của shadow, sau đó kích hoạt dịch chuyển lại bằng cách publish lên NATS `cdc.cmd.transmute`.
    *   `healThresholdBlocked(...)`: Bộ lọc an toàn (safety gate) để ngăn ngừa "bão" tự heal làm nghẽn hệ thống. Chặn tự động sửa lỗi nếu tổng số dòng lệch vượt quá 5000 dòng hoặc tỷ lệ lệch vượt quá 5%. Khi bị chặn, worker sẽ ghi nhận cảnh báo lên bảng `cdc_alerts`.

---

## 6. Thư mục `source/` (Source Connectors & Sync Status)
Thư mục này quản lý kết nối và cấu hình đồng bộ hóa với hệ thống Kafka Connect / Debezium.

### [source_register.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/source/source_register.go)
*   **Struct chính:** `RegisterHandler`, `RegisterSourceRequest`, `RegisterSourceResponse`
*   **Chức năng:** HTTP Endpoint phục vụ việc đăng ký nguồn dữ liệu mới (`POST /v2/sources/register`).
*   **Các bước xử lý chính:**
    1.  `step1InsertRegistry`: Ghi nhận thông tin kết nối và bảng vào registry trong một transaction an toàn.
    2.  `extendDebeziumInclude`: Gọi API REST tới Debezium để cập nhật danh sách các bảng/database cần theo dõi (include list) tương ứng với engine.
    3.  `preemptSchemaRegistry`: Cấu hình độ tương thích tương thích của Schema Registry sang `compatibility = NONE` đối với subject của topic tương ứng.
    4.  Gửi tin nhắn NATS lên `cdc.cmd.kafka.refresh-topics` để cập nhật cấu hình router.
    5.  Thiết lập trạng thái hoạt động `provisioning_state = active`.

### [sync_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/source/sync_handler.go)
*   **Struct chính:** `SyncHandler`
*   **Chức năng:** Lắng nghe các lệnh điều phối trạng thái đồng bộ hóa của Debezium.
*   **Các phương thức chính:**
    *   `HandleSyncRegister(msg)`: Lắng nghe `cdc.cmd.sync-register`. Thực hiện ping kiểm tra kết nối tới dịch vụ Kafka Connect.
    *   `HandleSyncState(msg)`: Lắng nghe `cdc.cmd.sync-state`. Nhận hành động `activate` hoặc `deactivate` để gửi yêu cầu `resume` hoặc `pause` tới connector cụ thể của Debezium.
    *   `HandleRestartDebezium(msg)`: Lắng nghe `cdc.cmd.restart-debezium`. Khởi động lại connector Debezium bị lỗi và các tasks đi kèm.
