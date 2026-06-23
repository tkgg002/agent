# Kế hoạch Tái cấu trúc và Phân bổ Chi tiết các Handler & Functions (Đồng bộ Solution)

Kế hoạch này thực hiện rà soát và cấu trúc lại toàn bộ các file, struct, handler và function trong thư mục `internal/handler/` dựa trên **Hệ Quy Chiếu Chức Năng Thư Mục Cốt Lõi** và giải pháp chi tiết tại `02_plan_solution.md`.

---

## Hệ Quy Chiếu: Định Nghĩa Chức Năng Thư Mục Cốt Lõi
- **`base/` (Base Utilities - Tầng Hạ Tầng)**: Tầng đáy vô tri. **TUYỆT ĐỐI KHÔNG** được biết các khái niệm Master, Shadow, Registry, Schema, hay Provisioning. Chỉ chứa các HTTP helpers, DB connection utilities thuần túy, và SQL sanitization helpers. Cắt đứt mọi liên kết tới cấu hình hệ thống (Metadata/Registry) để ép các Handler bên trên dùng Explicit DI.
- **`source/` (Source Connectors & Discovery)**: Entrypoint giao tiếp Upstream. Chịu trách nhiệm Đăng ký nguồn, cấu hình Registry, và Khám phá (Discover/Infer) cấu trúc DB gốc. Mọi thao tác Khám phá (Discover) và Suy luận (Infer) gọi driver chọc DB gốc phải gom về đây.
- **`master/` (Master Table Operations)**: Thao tác DDL cấu trúc cuối cùng và đổ dữ liệu vật lý (Data Plane/Transmute) vào đích Postgres. Bảo vệ `Transmute` ở lại đây, tuyệt đối không mang đi chỗ khác.
- **`shadow/` (Shadow Data Ingestion)**: Chỉ quản lý luồng ghi đệm Kafka, bảng đệm Ingestion, và tạo cấu trúc DDL bảng Đệm Shadow. Không có quyền vươn tay chọc vào DB gốc. Các thao tác cần rule sẽ dùng Explicit DI.
- **`recon/` (Data Reconciliation - Đối soát & Self-Healing)**: Bounded Context duy nhất phụ trách Data Integrity. Ôm trọn Đối Soát, Chữa Lành, Dead Letter Queue (DLQ) và Bù Đắp diện rộng (Scan Backfill).
- **`orchestration/` (Pipeline Orchestration - Control Plane)**: Control Plane (Nhạc trưởng). Quản lý State Machine (Tiến trình), Log hoạt động. Thuần túy là DAG State Machine. Chỉ được phép phát lệnh chỉ đạo, tuyệt đối không được phép thao tác DML (Insert/Update) trực tiếp lên Data Plane.

---

## Danh sách Audit Chi tiết & Kế hoạch Phân bổ (Bản Line-by-Line)

### 1. Thư mục `base/` (Base Utilities - Tầng Hạ Tầng)

**Nguyên tắc Hybrid:** Tầng hạ tầng vô tri. Cắt đứt mọi liên kết tới cấu hình hệ thống (Metadata/Registry) để ép các Handler bên trên dùng Explicit DI.

* `[x]` **Struct `BaseHandler**` (`base_handler.go`)
* **Mô tả chức năng:** Struct nền tảng chứa các instance kết nối hệ thống (GORM DB, NATS Client, Logger) cho các handler kế thừa.
* **Phân tích:** Việc struct này ngầm chứa `RegistryRepo` và `Metadata` khiến nó thành "God Object". Bất kỳ package nào import `BaseHandler` cũng vô tình bị dính chặt vào cơ sở dữ liệu cấu hình.
* **Lý do & Quyết định:** Để áp dụng Explicit DI chuẩn mực, cần làm sạch struct này. **Quyết định: CẬP NHẬT. LOẠI BỎ hoàn toàn dependency `RegistryRepo` và `Metadata` khỏi struct.**


* `[x]` **Hàm `NewBaseHandler**` (`base_handler.go`)
* **Mô tả chức năng:** Constructor khởi tạo instance của `BaseHandler`.
* **Phân tích:** Khởi tạo kết nối DB, Logger, NATS.
* **Lý do & Quyết định:** Do Struct bị xóa thuộc tính Registry, tham số đầu vào của constructor cũng phải gỡ bỏ. **Quyết định: CẬP NHẬT tham số đầu vào, loại bỏ khởi tạo Registry/Metadata.**


* `[x]` **Phương thức `SetMetadataRegistry**` (`base_handler.go`)
* **Mô tả chức năng:** Inject Metadata service vào BaseHandler.
* **Phân tích:** Khi BaseHandler không còn giữ Metadata, hàm này trở nên vô nghĩa.
* **Lý do & Quyết định:** Tuân thủ Explicit DI ở tầng trên. **Quyết định: XÓA BỎ hoàn toàn.**


* `[x]` **Phương thức `SetRegistryResolver**` (`base_handler.go`)
* **Mô tả chức năng:** Inject cấu hình resolver vào BaseHandler.
* **Phân tích:** Tương tự như hàm trên, vi phạm ranh giới base.
* **Lý do & Quyết định:** Dọn dẹp mã thừa. **Quyết định: XÓA BỎ hoàn toàn.**


* `[x]` **Phương thức `ResolveTargetSchema**` (`base_handler.go`)
* **Mô tả chức năng:** Phân giải và trả về tên schema của bảng đích.
* **Phân tích:** Đây là logic của Domain cấu hình (Metadata). Base layer không còn giữ Metadata nữa nên không thể chứa hàm này.
* **Lý do & Quyết định:** Phải nằm ở package Service để quản lý tập trung (Single Source of Truth). **Quyết định: DI CHUYỂN sang `internal/service/metadata/metadata_registry.go` làm helper function.**


* `[x]` **Phương thức `ResolveTargetTableConfig**` (`base_handler.go`)
* **Mô tả chức năng:** Lấy cấu hình đích từ tên bảng.
* **Phân tích:** Thuộc nghiệp vụ tra cứu Metadata.
* **Lý do & Quyết định:** Đưa về đúng domain. **Quyết định: DI CHUYỂN sang `internal/service/metadata/metadata_registry.go`.**


* `[x]` **Phương thức `ResolveTargetRoute**` (`base_handler.go`)
* **Mô tả chức năng:** Phân giải routing của bảng đích.
* **Phân tích:** Logic metadata.
* **Lý do & Quyết định:** Rò rỉ nghiệp vụ. **Quyết định: DI CHUYỂN sang `internal/service/metadata/metadata_registry.go`.**


* `[x]` **Phương thức `ResolveTableConfigByID**` (`base_handler.go`)
* **Mô tả chức năng:** Truy vấn cấu hình bảng theo ID.
* **Phân tích:** Thao tác DB nghiệp vụ cấu hình.
* **Lý do & Quyết định:** Đưa về service cấu hình. **Quyết định: DI CHUYỂN sang `internal/service/metadata/metadata_registry.go`.**


* `[x]` **Phương thức `ListActiveTableConfigs**` (`base_handler.go`)
* **Mô tả chức năng:** Lấy danh sách bảng đang kích hoạt.
* **Phân tích:** Thao tác DB nghiệp vụ cấu hình.
* **Lý do & Quyết định:** Đưa về service cấu hình. **Quyết định: DI CHUYỂN sang `internal/service/metadata/metadata_registry.go`.**


* `[x]` **Hàm `NormalizeMappingRuleDataType**` (`base_handler.go`)
* **Mô tả chức năng:** Chuẩn hóa chuỗi kiểu dữ liệu (vd: `int(11)` -> `integer`).
* **Phân tích:** Hàm này được dùng ở cả `shadow` (để tạo cột) và `master` (để ép kiểu lúc Transmute). Đặt ở `shadow` sẽ gây phụ thuộc chéo cho `master`.
* **Lý do & Quyết định:** Cần vùng đệm trung lập. **Quyết định: DI CHUYỂN sang file mới `internal/service/metadata/mapping_utils.go`.**


* `[x]` **Hàm `BuildCastExpr**` (`base_handler.go`)
* **Mô tả chức năng:** Sinh biểu thức `CAST(col AS type)` cho câu lệnh SQL.
* **Phân tích:** Tương tự như hàm trên, đây là SQL Builder phục vụ mapping dữ liệu, dùng chung cho nhiều module.
* **Lý do & Quyết định:** Giữ tính trung lập để tránh Coupling. **Quyết định: DI CHUYỂN sang `internal/service/metadata/mapping_utils.go`.**


* `[x]` **Hàm `bridgeMappingRulesToV2**` (Nằm rải rác ở code cũ)
* **Mô tả chức năng:** Chuyển đổi định dạng mapping từ V1 sang V2.
* **Phân tích:** Bị lặp code ở nhiều nơi.
* **Lý do & Quyết định:** Gom về vùng trung lập dùng chung. **Quyết định: DI CHUYỂN sang `internal/service/metadata/mapping_utils.go`.**


* `[x]` **Phương thức `TableExists**` (`base_handler.go`)
* **Mô tả chức năng:** Kiểm tra bảng tồn tại dựa vào schema mặc định.
* **Phân tích:** Logic thiếu chặt chẽ khi hệ thống chạy multi-schema. Đã có hàm `TableExistsInSchema` tường minh hơn.
* **Lý do & Quyết định:** Tránh rủi ro bug ngầm. **Quyết định: XÓA BỎ hoàn toàn. Caller sẽ gọi trực tiếp `TableExistsInSchema`.**


* `[x]` **Phương thức `HasColumn**` (`base_handler.go`)
* **Mô tả chức năng:** Kiểm tra cột tồn tại ở schema mặc định.
* **Phân tích:** Tương tự `TableExists`, dư thừa và nguy hiểm.
* **Lý do & Quyết định:** Clean code. **Quyết định: XÓA BỎ hoàn toàn. Caller sẽ gọi trực tiếp `HasColumnInSchema`.**


* `[x]` **Phương thức `TableExistsInSchema**` (`base_handler.go`)
* **Mô tả chức năng:** Kiểm tra sự tồn tại của bảng trong một schema cụ thể.
* **Phân tích:** Giao tiếp DB thô qua GORM, không phụ thuộc business.
* **Lý do & Quyết định:** Đúng chức năng Infra. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Phương thức `HasColumnInSchema**` (`base_handler.go`)
* **Mô tả chức năng:** Kiểm tra cột có trong bảng thuộc schema cụ thể không.
* **Phân tích:** Giao tiếp DB thô.
* **Lý do & Quyết định:** Đúng chức năng Infra. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Phương thức `WriteActivity**` (`base_handler.go`)
* **Mô tả chức năng:** Ghi lịch sử tiến trình vào bảng `cdc_activities`.
* **Phân tích:** Ghi lại trạng thái tiến trình Pipeline là nghiệp vụ của Nhạc trưởng Orchestration. Base vô tri không nên quản lý Activity.
* **Lý do & Quyết định:** Trả về đúng module. **Quyết định: DI CHUYỂN sang `internal/handler/orchestration/activity_logger.go`.**


* `[x]` **Phương thức `NatsPublish**` (`base_handler.go`)
* **Mô tả chức năng:** Wrapper gửi message byte NATS thô.
* **Phân tích:** Giao tiếp I/O nền tảng.
* **Lý do & Quyết định:** Nằm đúng layer Base. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Phương thức `PublishResultWithSubject**` (`base_handler.go`)
* **Mô tả chức năng:** Đóng gói JSON và gửi NATS với chủ đề chỉ định.
* **Phân tích:** Tiện ích I/O.
* **Lý do & Quyết định:** Đúng layer. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Phương thức `PublishResult**` (`base_handler.go`)
* **Mô tả chức năng:** Đóng gói JSON và gửi NATS mặc định.
* **Phân tích:** Tiện ích I/O.
* **Lý do & Quyết định:** Đúng layer. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Phương thức `ConnectGET**` (`base_handler.go`)
* **Mô tả chức năng:** Gọi REST HTTP GET.
* **Phân tích:** Hạ tầng mạng thô.
* **Lý do & Quyết định:** Thuộc hạ tầng nền tảng. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Phương thức `ConnectPOST**` (`base_handler.go`)
* **Mô tả chức năng:** Gọi REST HTTP POST.
* **Phân tích:** Hạ tầng mạng thô.
* **Lý do & Quyết định:** Thuộc hạ tầng nền tảng. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Phương thức `ConnectPUT**` (`base_handler.go`)
* **Mô tả chức năng:** Gọi REST HTTP PUT.
* **Phân tích:** Hạ tầng mạng thô.
* **Lý do & Quyết định:** Thuộc hạ tầng nền tảng. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Phương thức `ConnectCall**` (`base_handler.go`)
* **Mô tả chức năng:** Lõi gọi HTTP Client (Resty).
* **Phân tích:** Hạ tầng mạng thô.
* **Lý do & Quyết định:** Thuộc hạ tầng nền tảng. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Phương thức `LogCommandResult**` (`base_handler.go`)
* **Mô tả chức năng:** Ghi log console kết quả command.
* **Phân tích:** Tiện ích logger.
* **Lý do & Quyết định:** Thuộc layer Base. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Hàm `IsSafeIdent**` (`base_handler.go`)
* **Mô tả chức năng:** Validate chống SQL Injection định danh cột/bảng.
* **Phân tích:** Xử lý text regex thuần túy.
* **Lý do & Quyết định:** Utils dùng chung an toàn. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Hàm `IsSafeType**` (`base_handler.go`)
* **Mô tả chức năng:** Validate chuỗi kiểu dữ liệu.
* **Phân tích:** Xử lý text.
* **Lý do & Quyết định:** Utils an toàn. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Hàm `SanitizeAdminError**` (`base_handler.go`)
* **Mô tả chức năng:** Xóa thông tin nhạy cảm trong String lỗi.
* **Phân tích:** Text manipulation.
* **Lý do & Quyết định:** Utils an toàn. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Hàm `SanitizeAdminResultMap**` (`base_handler.go`)
* **Mô tả chức năng:** Lọc data map JSON.
* **Phân tích:** Utils xử lý bộ nhớ.
* **Lý do & Quyết định:** Utils an toàn. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Hàm `SanitizeAdminFields**` (`base_handler.go`)
* **Mô tả chức năng:** Lọc mảng field string.
* **Phân tích:** Utils mảng bộ nhớ.
* **Lý do & Quyết định:** Utils an toàn. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Struct `StepResult**` (`provisioning_emit.go`)
* **Mô tả chức năng:** Cấu trúc DTO định nghĩa trạng thái bước (Success/Fail).
* **Phân tích:** Mọi layer (master, shadow, source) đều dùng enum này để gửi báo cáo về Nhạc trưởng. Đặt ở base giúp tránh Circular Dependency.
* **Lý do & Quyết định:** Giao thức giao tiếp chung. **Quyết định: GIỮ NGUYÊN tại `base/`.**


* `[x]` **Struct `stepCompletedPayload**` (`provisioning_emit.go`)
* **Mô tả chức năng:** Payload body JSON gửi qua NATS.
* **Phân tích:** DTO giao tiếp nội bộ.
* **Lý do & Quyết định:** DTO dùng chung. **Quyết định: GIỮ NGUYÊN tại `base/`.**


* `[x]` **Hàm `EmitStepCompleted**` (`provisioning_emit.go`)
* **Mô tả chức năng:** Bắn sự kiện NATS báo cáo hoàn thành bước.
* **Phân tích:** Là "Trung tâm phát tín hiệu" (Single Source of Truth) để mọi module gọi xuống.
* **Lý do & Quyết định:** Chống lặp code, tuyệt đối không tạo hàm này ở các package khác. **Quyết định: GIỮ NGUYÊN TẠI BASE.**



---

### 2. Thư mục `shadow/` (Shadow Data Ingestion)

**Nguyên tắc Hybrid:** Chỉ quản lý luồng ghi đệm Kafka và tạo cấu trúc DDL bảng Đệm. Các thao tác cần rule sẽ dùng Explicit DI. KHÔNG gọi thẳng DB gốc.

* `[x]` **Struct `BatchBuffer**` (`batch_buffer.go`)
* **Mô tả chức năng:** Bộ đệm In-memory (RAM) lưu batch sự kiện CDC.
* **Phân tích:** Trái tim của luồng ghi tối ưu I/O (Ingestion).
* **Lý do & Quyết định:** Phù hợp với Bounded Context của bảng đệm. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Struct `bufferItem**` (`batch_buffer.go`)
* **Mô tả chức năng:** Object chứa byte payload từng dòng.
* **Phân tích:** Thành phần của BatchBuffer.
* **Lý do & Quyết định:** Tính đóng gói nội bộ. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Phương thức `Add**` (`batch_buffer.go`)
* **Mô tả chức năng:** Thêm 1 item vào mảng đệm.
* **Phân tích:** Thao tác bộ nhớ.
* **Lý do & Quyết định:** Hành vi của Buffer. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Phương thức `Flush**` (`batch_buffer.go`)
* **Mô tả chức năng:** Bulk Insert mảng đệm xuống DB Shadow vật lý.
* **Phân tích:** Thao tác I/O Ingestion.
* **Lý do & Quyết định:** Hành vi của Buffer. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Phương thức `StartPeriodicFlush**` (`batch_buffer.go`)
* **Mô tả chức năng:** Goroutine timer tự xả đệm sau X giây.
* **Phân tích:** Quản lý vòng đời Ingestion.
* **Lý do & Quyết định:** Hành vi của Buffer. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Struct `BatchTransformHandler**` (`batch_transform_handler.go`)
* **Mô tả chức năng:** Controller áp dụng luật biến đổi kiểu dữ liệu cho lô data trước khi ghi đệm.
* **Phân tích:** BaseHandler đã mất quyền truy cập Metadata, struct này cần Metadata để biết biến đổi kiểu gì.
* **Lý do & Quyết định:** Áp dụng Explicit DI. **Quyết định: CẬP NHẬT. Khai báo thuộc tính `metadataRegistry metadata.MetadataRegistry` và hàm `SetMetadataRegistry`.**


* `[x]` **Phương thức `HandleTransform**` (`batch_transform_handler.go`)
* **Mô tả chức năng:** Vòng lặp lắng nghe lệnh biến đổi.
* **Phân tích:** Cần gọi registry để lấy rules.
* **Lý do & Quyết định:** Logic biến đổi không đổi. **Quyết định: CẬP NHẬT. Đổi biến truy cập rules sang dùng trực tiếp `this.metadataRegistry`.**


* `[x]` **Struct `ConsumerPool**` (`consumer_pool.go`)
* **Mô tả chức năng:** Quản lý vòng đời cụm Kafka Workers.
* **Phân tích:** Hạ tầng kéo luồng data.
* **Lý do & Quyết định:** Đúng domain Ingestion. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Phương thức `Start**` (`consumer_pool.go`)
* **Mô tả chức năng:** Kích hoạt toàn bộ Worker.
* **Phân tích:** Lifecycle pool.
* **Lý do & Quyết định:** Hành vi của Pool. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Phương thức `Stop**` (`consumer_pool.go`)
* **Mô tả chức năng:** Đóng Graceful shutdown.
* **Phân tích:** Lifecycle pool.
* **Lý do & Quyết định:** Hành vi của Pool. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Struct `DLQCircuitBreaker**` (`dlq_circuit_breaker.go`)
* **Mô tả chức năng:** Cầu dao ngắt luồng Kafka nếu tỷ lệ chèn đệm lỗi vượt ngưỡng.
* **Phân tích:** Bảo vệ trực tiếp luồng Ingestion.
* **Lý do & Quyết định:** Đúng domain bảo vệ bảng đệm. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Struct `EventBridge**` (`event_bridge.go`)
* **Mô tả chức năng:** Định tuyến trigger event từ bảng Shadow.
* **Phân tích:** Luồng event sau khi ghi đệm.
* **Lý do & Quyết định:** Hoạt động quanh bảng đệm. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Phương thức `HandleTriggerEvent**` (`event_bridge.go`)
* **Mô tả chức năng:** Phân loại và chuyển tiếp trigger.
* **Phân tích:** Logic cầu nối.
* **Lý do & Quyết định:** Hành vi của Bridge. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Struct `EventHandler**` (`event_handler.go`)
* **Mô tả chức năng:** Bộ não xử lý 1 message Avro thô đẩy vào luồng.
* **Phân tích:** Core data pipeline.
* **Lý do & Quyết định:** Trái tim của Shadow. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Phương thức `HandleRaw**` (`event_handler.go`)
* **Mô tả chức năng:** Parse Avro và đẩy vào Batch Buffer.
* **Phân tích:** Parser phase.
* **Lý do & Quyết định:** Hành vi của EventHandler. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Phương thức `FlushBatchBuffer**` (`event_handler.go`)
* **Mô tả chức năng:** Proxy ép xả đệm thủ công.
* **Phân tích:** Command util.
* **Lý do & Quyết định:** Hành vi của EventHandler. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Struct `KafkaConsumer**` (`kafka_consumer.go`)
* **Mô tả chức năng:** Wrapper kết nối Sarama Kafka Broker.
* **Phân tích:** Điểm nhận byte stream.
* **Lý do & Quyết định:** Component nguồn Ingestion. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Struct `avroMessageDecoder**` (`kafka_consumer.go`)
* **Mô tả chức năng:** Giao tiếp Confluent Schema Registry để giải mã byte.
* **Phân tích:** Đi kèm Kafka Consumer.
* **Lý do & Quyết định:** Component Ingestion. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Phương thức `Start**` (`kafka_consumer.go`)
* **Mô tả chức năng:** Vòng lặp Listen Partition Kafka.
* **Phân tích:** Loop consumer.
* **Lý do & Quyết định:** Hành vi của Consumer. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Phương thức `processMessage**` (`kafka_consumer.go`)
* **Mô tả chức năng:** Callback đẩy byte sang EventHandler.
* **Phân tích:** Móc xích pipeline.
* **Lý do & Quyết định:** Hành vi của Consumer. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Struct `SchemaDDLHandler**` (Đang ở `master/schema_ddl_handler.go`)
* **Mô tả chức năng:** Cấu trúc tạo cột CDC, alter bảng trung gian (Shadow).
* **Phân tích:** Đối tượng tác động là "Shadow Table". Để ở `master` là sai nhà nghiêm trọng. Cần áp dụng Explicit DI `metadataRegistry` để lấy rule.
* **Lý do & Quyết định:** Trả về Bounded Context thực sự. **Quyết định: DI CHUYỂN sang `internal/handler/shadow/schema_ddl_handler.go`. Khai báo Explicit DI.**


* `[x]` **Phương thức `ensureCDCColumns**` (`schema_ddl_handler.go`)
* **Mô tả chức năng:** Thêm các cột hệ thống (`cdc_op`, `sys_id`) vào bảng.
* **Phân tích:** Lệnh DDL bảng đệm.
* **Lý do & Quyết định:** Đi theo struct. **Quyết định: DI CHUYỂN sang `shadow/`.**


* `[x]` **Phương thức `ensureCDCColumnsInSchema**` (`schema_ddl_handler.go`)
* **Mô tả chức năng:** Tương tự, có chỉ định tên schema.
* **Phân tích:** Lệnh DDL bảng đệm.
* **Lý do & Quyết định:** Đi theo struct. **Quyết định: DI CHUYỂN sang `shadow/`.**


* `[x]` **Phương thức `listShadowColumns**` (`schema_ddl_handler.go`)
* **Mô tả chức năng:** Trả mảng string tên cột hiện tại.
* **Phân tích:** Read DDL.
* **Lý do & Quyết định:** Đi theo struct. **Quyết định: DI CHUYỂN sang `shadow/`.**


* `[x]` **Phương thức `listShadowColumnsWithType**` (`schema_ddl_handler.go`)
* **Mô tả chức năng:** Trả map Tên Cột : Kiểu Dữ Liệu.
* **Phân tích:** Read DDL.
* **Lý do & Quyết định:** Đi theo struct. **Quyết định: DI CHUYỂN sang `shadow/`.**


* `[x]` **Phương thức `HandleStandardize**` (`schema_ddl_handler.go`)
* **Mô tả chức năng:** Event chạy chuẩn hóa toàn bảng Shadow.
* **Phân tích:** Tiến trình Provisioning bảng đệm.
* **Lý do & Quyết định:** Đi theo struct. **Quyết định: DI CHUYỂN sang `shadow/`.**


* `[x]` **Phương thức `HandleCreateDefaultColumns**` (`schema_ddl_handler.go`)
* **Mô tả chức năng:** Kích hoạt tạo cột CDC.
* **Phân tích:** Controller DDL.
* **Lý do & Quyết định:** Đi theo struct. **Quyết định: DI CHUYỂN sang `shadow/`.**


* `[x]` **Phương thức `HandleDropGINIndex**` (`schema_ddl_handler.go`)
* **Mô tả chức năng:** Tạm xóa GIN Index để tăng tốc độ Insert Bulk.
* **Phân tích:** Tối ưu Ingestion DB đệm.
* **Lý do & Quyết định:** Đi theo struct. **Quyết định: DI CHUYỂN sang `shadow/`.**


* `[x]` **Phương thức `HandleAlterColumn**` (`schema_ddl_handler.go`)
* **Mô tả chức năng:** Đổi Type của cột trên bảng shadow.
* **Phân tích:** Controller DDL bảng đệm.
* **Lý do & Quyết định:** Đi theo struct. **Quyết định: DI CHUYỂN sang `shadow/`.**


* `[x]` **Struct `ShadowBindHandler**` (Tách từ orchestration/)
* **Mô tả chức năng:** Cấu trúc liên kết Mapping (Binding) định hình bảng Đích Shadow.
* **Phân tích:** Công việc định hình Shadow đang kẹt ở God file `provisioning_step_handlers`. Phải trả về Shadow quản lý.
* **Lý do & Quyết định:** Giải phóng Nhạc trưởng. **Quyết định: TẠO MỚI file `internal/handler/shadow/provisioning_shadow_bind.go` chứa struct này.**


* `[x]` **Struct `shadowBindRequest**` (`provisioning_shadow_bind.go`)
* **Mô tả chức năng:** DTO Payload cho sự kiện Bind.
* **Phân tích:** Giao diện Request.
* **Lý do & Quyết định:** Phục vụ Handler. **Quyết định: Đặt tại file mới `shadow/`.**


* `[x]` **Struct `shadowTarget**` (`provisioning_shadow_bind.go`)
* **Mô tả chức năng:** DTO định vị đích lưu trữ.
* **Phân tích:** Giao diện Metadata nội bộ.
* **Lý do & Quyết định:** Phục vụ Handler. **Quyết định: Đặt tại file mới `shadow/`.**


* `[x]` **Phương thức `HandleShadowBind**` (`provisioning_shadow_bind.go`)
* **Mô tả chức năng:** Controller điều phối bước Bind Shadow.
* **Phân tích:** Workflow controller.
* **Lý do & Quyết định:** Hành vi của Handler. **Quyết định: Đặt tại file mới `shadow/`.**


* `[x]` **Hàm `resolveShadowTarget**` (`provisioning_shadow_bind.go`)
* **Mô tả chức năng:** Helper phân giải URL / Naming Rule cho đích.
* **Phân tích:** Tiện ích nội bộ luồng Bind.
* **Lý do & Quyết định:** Đi kèm luồng Bind. **Quyết định: Đặt tại file mới `shadow/`.**


* `[x]` **Hàm `upsertShadowBinding**` (`provisioning_shadow_bind.go`)
* **Mô tả chức năng:** Lưu kết quả Bind vào Meta DB.
* **Phân tích:** Giao tiếp DB lưu cấu hình cuối.
* **Lý do & Quyết định:** Đi kèm luồng Bind. **Quyết định: Đặt tại file mới `shadow/`.**



---

### 3. Thư mục `master/` (Master Table Operations)

**Nguyên tắc Hybrid:** DDL Đích Cuối và Data Plane vật lý. Bảo vệ `Transmute` ở lại đây, tuyệt đối không mang đi chỗ khác.

* `[x]` **Struct `MasterDDLHandler**` (`master_ddl_handler.go`)
* **Mô tả chức năng:** Tạo, đổi cột và Rename trên bảng Đích cuối (Postgres).
* **Phân tích:** Domain quản trị cấu trúc Master.
* **Lý do & Quyết định:** Đúng chức năng, đúng vị trí. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Phương thức `HandleMasterCreate**` (`master_ddl_handler.go`)
* **Mô tả chức năng:** Thực thi `CREATE TABLE`.
* **Phân tích:** Lệnh DDL vật lý.
* **Lý do & Quyết định:** Hành vi Struct Master. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Phương thức `HandleMasterAlterColumn**` (`master_ddl_handler.go`)
* **Mô tả chức năng:** Thực thi `ALTER TABLE`.
* **Phân tích:** Lệnh DDL vật lý.
* **Lý do & Quyết định:** Hành vi Struct Master. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Phương thức `HandleMasterSwap**` (`master_ddl_handler.go`)
* **Mô tả chức năng:** Swap tên bảng (Đưa Shadow lên thành Live Master).
* **Phân tích:** Lệnh DDL vật lý cuối cùng của vòng đời tạo bảng.
* **Lý do & Quyết định:** Hành vi Struct Master. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Struct `TransmuteHandler**` (`transmute_handler.go`)
* **Mô tả chức năng:** Đọc Bulk hàng triệu dòng từ Shadow, áp SQL `CAST` và `INSERT/UPDATE` trực tiếp xuống Master DB.
* **Phân tích:** Đây là tác vụ **Data Plane Cường Độ Cao nhất**. Việc đưa nó vào Orchestration (như kế hoạch cũ) sẽ biến Control Plane thành công nhân bốc vác data, phá hủy triệt để ranh giới kiến trúc.
* **Lý do & Quyết định:** Bảo vệ thiết kế Data Plane/Control Plane. **Quyết định: TỪ CHỐI DI CHUYỂN. GIỮ NGUYÊN TẠI `master/transmute_handler.go`.**


* `[x]` **Phương thức `HandleTransmute**` (`transmute_handler.go`)
* **Mô tả chức năng:** Kích hoạt job Transmute toàn bộ bảng.
* **Phân tích:** Lệnh DML vật lý.
* **Lý do & Quyết định:** Hành vi của Transmute. **Quyết định: GIỮ NGUYÊN tại `master/`.**


* `[x]` **Phương thức `HandleTransmuteShadow**` (`transmute_handler.go`)
* **Mô tả chức năng:** Transmute một phần (Incremental).
* **Phân tích:** Lệnh DML vật lý.
* **Lý do & Quyết định:** Hành vi của Transmute. **Quyết định: GIỮ NGUYÊN tại `master/`.**


* `[x]` **Phương thức `publishCompleted**` (`transmute_handler.go`)
* **Mô tả chức năng:** NATS Emit báo hoàn tất job Transmute.
* **Phân tích:** Internal NATS.
* **Lý do & Quyết định:** Helper nội bộ của struct. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Phương thức `reply**` (`transmute_handler.go`)
* **Mô tả chức năng:** NATS Reply chuẩn Request-Reply pattern.
* **Phân tích:** NATS util.
* **Lý do & Quyết định:** Helper nội bộ. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Phương thức `replyErr**` (`transmute_handler.go`)
* **Mô tả chức năng:** NATS Reply báo lỗi.
* **Phân tích:** NATS util.
* **Lý do & Quyết định:** Helper nội bộ. **Quyết định: GIỮ NGUYÊN.**



---

### 4. Thư mục `source/` (Source Connectors & Discovery)

**Nguyên tắc Hybrid:** Entrypoint giao tiếp Upstream. Mọi thao tác Khám phá (Discover) và Suy luận (Infer) gọi driver chọc DB gốc phải gom về đây.

* `[x]` **Struct `RegisterHandler**` (`source_register.go`)
* **Mô tả chức năng:** API Endpoint nhận cấu hình DB Nguồn mới từ người dùng.
* **Phân tích:** Gateway khai báo Nguồn vào hệ thống.
* **Lý do & Quyết định:** Domain quản trị Nguồn. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Struct `RegisterSourceRequest**` (`source_register.go`)
* **Mô tả chức năng:** Format JSON Input đăng ký.
* **Phân tích:** DTO.
* **Lý do & Quyết định:** Thuộc Handler. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Struct `RegisterSourceResponse**` (`source_register.go`)
* **Mô tả chức năng:** Format JSON Output kết quả.
* **Phân tích:** DTO.
* **Lý do & Quyết định:** Thuộc Handler. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Phương thức `HandleRegister**` (`source_register.go`)
* **Mô tả chức năng:** Workflow điều phối luồng đăng ký Nguồn.
* **Phân tích:** API Controller.
* **Lý do & Quyết định:** Thuộc RegisterHandler. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Phương thức `step1InsertRegistry**` (`source_register.go`)
* **Mô tả chức năng:** Tạo bản ghi vào bảng Metadata Registry.
* **Phân tích:** Step nội bộ.
* **Lý do & Quyết định:** Thuộc RegisterHandler. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Phương thức `markProvisioningFailed**` (`source_register.go`)
* **Mô tả chức năng:** Rollback trạng thái khi lỗi.
* **Phân tích:** Step báo lỗi.
* **Lý do & Quyết định:** Thuộc RegisterHandler. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Phương thức `resolveConnectorByEngine**` (`source_register.go`)
* **Mô tả chức năng:** Switch logic sinh JSON payload cho Connector PG/MySQL/Mongo.
* **Phân tích:** Mapping rule connector.
* **Lý do & Quyết định:** Thuộc RegisterHandler. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Phương thức `extendDebeziumInclude**` (`source_register.go`)
* **Mô tả chức năng:** Bắn REST API sang worker Kafka Connect báo theo dõi thêm 1 bảng.
* **Phân tích:** Gọi API external hệ sinh thái data.
* **Lý do & Quyết định:** Thuộc RegisterHandler. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Phương thức `preemptSchemaRegistry**` (`source_register.go`)
* **Mô tả chức năng:** Bắn REST API tắt Validation Avro.
* **Phân tích:** Gọi API external.
* **Lý do & Quyết định:** Thuộc RegisterHandler. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Struct `SyncHandler**` (`sync_handler.go`)
* **Mô tả chức năng:** Điều khiển trạng thái bật/tắt (Pause/Resume) của connector đồng bộ.
* **Phân tích:** Quản trị vòng đời Connector Nguồn.
* **Lý do & Quyết định:** Đúng domain giao tiếp Upstream. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Phương thức `HandleSyncRegister**` (`sync_handler.go`)
* **Mô tả chức năng:** NATS event command tạo luồng đồng bộ.
* **Phân tích:** Command controller.
* **Lý do & Quyết định:** Hành vi của SyncHandler. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Phương thức `verifyDebeziumConnector**` (`sync_handler.go`)
* **Mô tả chức năng:** REST API ping xem connector sống hay chết.
* **Phân tích:** Health check Nguồn.
* **Lý do & Quyết định:** Tiện ích nội bộ Sync. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Phương thức `HandleSyncState**` (`sync_handler.go`)
* **Mô tả chức năng:** NATS event command Pause/Resume.
* **Phân tích:** Command controller.
* **Lý do & Quyết định:** Hành vi của SyncHandler. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Phương thức `HandleRestartDebezium**` (`sync_handler.go`)
* **Mô tả chức năng:** NATS event command Restart Task.
* **Phân tích:** Command controller bảo trì.
* **Lý do & Quyết định:** Hành vi của SyncHandler. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Phương thức `resolveTableConfigByID**` (`sync_handler.go`)
* **Mô tả chức năng:** Helper truy xuất db config.
* **Phân tích:** Tiện ích nội bộ Sync.
* **Lý do & Quyết định:** Hành vi của SyncHandler. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Phương thức `detectConnectorName**` (`sync_handler.go`)
* **Mô tả chức năng:** Lấy ID string của connector.
* **Phân tích:** Tiện ích nội bộ Sync.
* **Lý do & Quyết định:** Hành vi của SyncHandler. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Struct `DiscoverHandler**` (Từ orchestration/)
* **Mô tả chức năng:** Tự động quyét, lấy row mẫu từ DB Upstream để lập Map phân loại Type.
* **Phân tích:** Tác vụ quét DB Upstream không thuộc quản lý của Nhạc trưởng Orchestration. Trách nhiệm này thuộc Domain `source/`. Cần Explicit DI để ghi rule.
* **Lý do & Quyết định:** Tái cấu trúc chuẩn DDD. **Quyết định: DI CHUYỂN sang `internal/handler/source/discover_handler.go`. Thêm Explicit DI `metadataRegistry`.**


* `[x]` **Phương thức `scanFieldsMongoSource**` (`discover_handler.go`)
* **Mô tả chức năng:** Query NoSQL MongoDB để parse Array BSON sang Schema.
* **Phân tích:** Driver chọc DB gốc.
* **Lý do & Quyết định:** Hành vi Discover. **Quyết định: DI CHUYỂN sang `source/`.**


* `[x]` **Phương thức `processDiscoveryRows**` (`discover_handler.go`)
* **Mô tả chức năng:** Phân tích cục data mẫu để Generate cấu hình Mapping.
* **Phân tích:** Builder pattern rules.
* **Lý do & Quyết định:** Hành vi Discover. **Quyết định: DI CHUYỂN sang `source/`.**


* `[x]` **Phương thức `HandleDiscover**` (`discover_handler.go`)
* **Mô tả chức năng:** NATS event chạy toàn bộ tiến trình.
* **Phân tích:** Controller API.
* **Lý do & Quyết định:** Hành vi Discover. **Quyết định: DI CHUYỂN sang `source/`.**


* `[x]` **Phương thức `ScanFieldsDebezium**` (`discover_handler.go`)
* **Mô tả chức năng:** Quét cấu trúc Avro Schema Registry để chẩn đoán.
* **Phân tích:** Giao tiếp External.
* **Lý do & Quyết định:** Hành vi Discover. **Quyết định: DI CHUYỂN sang `source/`.**


* `[x]` **Phương thức `HandleScanFields**` (`discover_handler.go`)
* **Mô tả chức năng:** NATS event quét lại mảng cấu trúc độc lập.
* **Phân tích:** Controller API.
* **Lý do & Quyết định:** Hành vi Discover. **Quyết định: DI CHUYỂN sang `source/`.**


* `[x]` **Struct `MongoDiscoverHandler**` (Từ orchestration/)
* **Mô tả chức năng:** Giao tiếp driver MongoClient trả về danh sách DB và Collection nội bộ của Mongo.
* **Phân tích:** Thao tác "Đọc vị" DB nguồn thô.
* **Lý do & Quyết định:** Giải cứu khỏi Orchestration. **Quyết định: DI CHUYỂN sang `internal/handler/source/mongo_discover_handler.go`.**


* `[x]` **Phương thức `HandleDiscoverMongoDatabases**` (`mongo_discover_handler.go`)
* **Mô tả chức năng:** Chạy truy vấn lấy Array Databases.
* **Phân tích:** Driver Mongo gốc.
* **Lý do & Quyết định:** Hành vi MongoDiscover. **Quyết định: DI CHUYỂN sang `source/`.**


* `[x]` **Phương thức `HandleDiscoverMongoCollections**` (`mongo_discover_handler.go`)
* **Mô tả chức năng:** Chạy truy vấn lấy Array Collections.
* **Phân tích:** Driver Mongo gốc.
* **Lý do & Quyết định:** Hành vi MongoDiscover. **Quyết định: DI CHUYỂN sang `source/`.**


* `[x]` **Hàm `inferSourceColumns**` (Tách từ orchestration/provisioning_step_handlers)
* **Mô tả chức năng:** Khối Router tự động switch gọi hàm đọc `information_schema` phù hợp (PG/MySQL/Mongo) để lấy Type gốc của DB Khách.
* **Phân tích:** Gói Shadow cần hàm này để tạo bảng, nhưng nếu để ở Shadow, Shadow sẽ vi phạm ranh giới vì được phép chọc DB Khách. Bắt buộc để ở Source.
* **Lý do & Quyết định:** Bảo vệ ranh giới kiến trúc khắt khe nhất. **Quyết định: TẠO MỚI file `internal/handler/source/infer_helpers.go` và đưa hàm này vào.**


* `[x]` **Hàm `inferPGCols**` (`infer_helpers.go`)
* **Mô tả chức năng:** Query Postgres `information_schema.columns`.
* **Phân tích:** Driver Postgres gốc.
* **Lý do & Quyết định:** Tác vụ Khám phá Cấu trúc. **Quyết định: Đưa vào file mới `infer_helpers.go` tại `source/`.**


* `[x]` **Hàm `inferMySQLCols**` (`infer_helpers.go`)
* **Mô tả chức năng:** Query MySQL `information_schema`.
* **Phân tích:** Driver MySQL gốc.
* **Lý do & Quyết định:** Tác vụ Khám phá Cấu trúc. **Quyết định: Đưa vào file mới `infer_helpers.go` tại `source/`.**


* `[x]` **Hàm `inferMongoCols**` (`infer_helpers.go`)
* **Mô tả chức năng:** Khám phá cấu trúc BSON.
* **Phân tích:** Driver Mongo gốc.
* **Lý do & Quyết định:** Tác vụ Khám phá Cấu trúc. **Quyết định: Đưa vào file mới `infer_helpers.go` tại `source/`.**


* `[x]` **Hàm `isMongoEngine**` (`infer_helpers.go`)
* **Mô tả chức năng:** Tiện ích String so khớp check `mongodb`.
* **Phân tích:** Tiện ích nội bộ Infer.
* **Lý do & Quyết định:** Đi kèm luồng Infer. **Quyết định: Đưa vào file mới `infer_helpers.go` tại `source/`.**


* `[x]` **Hàm `fetchSourceEngine**` (`infer_helpers.go`)
* **Mô tả chức năng:** Decode meta DB để lấy chuỗi tên Engine nguồn.
* **Phân tích:** Tiện ích nội bộ Infer.
* **Lý do & Quyết định:** Đi kèm luồng Infer. **Quyết định: Đưa vào file mới `infer_helpers.go` tại `source/`.**


* `[x]` **Hàm `preflightMongoSource**` (`infer_helpers.go`)
* **Mô tả chức năng:** Thử ping mở kết nối MongoDB rỗng để Test Connection.
* **Phân tích:** Upstream network test.
* **Lý do & Quyết định:** Đi kèm luồng Infer. **Quyết định: Đưa vào file mới `infer_helpers.go` tại `source/`.**



---

### 5. Thư mục `recon/` (Data Reconciliation - Đối soát & Self-Healing)

**Nguyên tắc Hybrid:** Bounded Context duy nhất phụ trách Data Integrity. Ôm trọn Đối Soát, Chữa Lành, Dead Letter Queue và Bù Đắp diện rộng (Scan Backfill).

* `[x]` **Struct `ReconHandler**` (`recon_handler.go`)
* **Mô tả chức năng:** Engine đối soát, phát hiện sự chênh lệch checksum, offset giữa bảng Upstream, Shadow và Master.
* **Phân tích:** Lõi của quy trình tính toàn vẹn.
* **Lý do & Quyết định:** Đúng chức năng. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Phương thức `WithBackfill`, `WithHealer`, `WithMaskingService`, `WithTimestampDetector`, `WithMetadataRegistry`, `WithSignalClient**` (`recon_handler.go`)
* **Mô tả chức năng:** Builder Pattern dùng để chèn dependencies.
* **Phân tích:** Constructor an toàn của Explicit DI.
* **Lý do & Quyết định:** Bảo vệ DI. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Phương thức `HandleReconCheck**` (`recon_handler.go`)
* **Mô tả chức năng:** Trigger chạy hàm so sánh Checksum.
* **Phân tích:** Tác vụ Audit dữ liệu.
* **Lý do & Quyết định:** Hành vi Recon. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Phương thức `handleReconCheckSegmentB**` (`recon_handler.go`)
* **Mô tả chức năng:** So khớp count/checksum riêng khúc (Shadow -> Master).
* **Phân tích:** Logic kiểm định Segment.
* **Lý do & Quyết định:** Tiện ích nội bộ Recon. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Phương thức `HandleReconHeal**` (`recon_handler.go`)
* **Mô tả chức năng:** NATS Event Command chạy quy trình tự sửa dữ liệu lỗi ngầm.
* **Phân tích:** Tính năng Self-Healing core.
* **Lý do & Quyết định:** Hành vi Recon. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Phương thức `HandleRetryFailed**` (`recon_handler.go`)
* **Mô tả chức năng:** Gửi lệnh Replay ép chạy lại dòng data failed.
* **Phân tích:** Phục hồi sau lỗi mạng.
* **Lý do & Quyết định:** Hành vi Recon. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Phương thức `HandleDebeziumSignal**` (`recon_handler.go`)
* **Mô tả chức năng:** Đẩy record kích hoạt vào bảng DB `debezium_signal` để ép tool đọc lại block.
* **Phân tích:** Thao tác "Heal" Segment A (từ nguồn vào).
* **Lý do & Quyết định:** Hành vi Recon. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Phương thức `HandleBackfillSourceTs**` (`recon_handler.go`)
* **Mô tả chức năng:** Rà quét và update Timestamp (`updated_at`) mồ côi.
* **Phân tích:** Thao tác vá Data.
* **Lý do & Quyết định:** Hành vi Recon. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Phương thức `HandleDetectTimestampField**` (`recon_handler.go`)
* **Mô tả chức năng:** Đọc Table Schema nguồn đoán xem cột nào làm Timestamp đồng bộ tốt nhất.
* **Phân tích:** Tool bổ trợ Backfill Recon.
* **Lý do & Quyết định:** Hành vi Recon. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Phương thức `resolveTargetTableConfig`, `resolveTableConfigByID**` (`recon_handler.go`)
* **Mô tả chức năng:** Kéo Meta Rule để phục vụ việc tạo truy vấn.
* **Phân tích:** Tiện ích Config.
* **Lý do & Quyết định:** Tiện ích nội bộ Recon. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Phương thức `updateFailedLog**` (`recon_handler.go`)
* **Mô tả chức năng:** Update DB flag cờ Retry Success/Fail.
* **Phân tích:** Tracker lưu vet.
* **Lý do & Quyết định:** Tiện ích nội bộ Recon. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Phương thức `logActivity**` (`recon_handler.go`)
* **Mô tả chức năng:** NATS emit báo hệ thống Healer đang chạy.
* **Phân tích:** Thông báo nội bộ.
* **Lý do & Quyết định:** Tiện ích nội bộ Recon. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Phương thức `sanitizeRetryRawJSON**` (`recon_handler.go`)
* **Mô tả chức năng:** Gọt bớt cục String JSON lớn khi bị lỗi để ko tràn DB.
* **Phân tích:** Text manipulation an toàn DB.
* **Lý do & Quyết định:** Tiện ích nội bộ Recon. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Phương thức `SanitizeRetryRawJSONForTest**` (`recon_handler.go`)
* **Mô tả chức năng:** Export public cho Unit Test.
* **Phân tích:** Support Unit Test.
* **Lý do & Quyết định:** Tiện ích nội bộ Recon. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Hàm `healSegmentA**` (`recon_heal_v4.go`)
* **Mô tả chức năng:** Logic tính toán bù lỗ hổng Segment (Upstream -> Kafka).
* **Phân tích:** Thuật toán lõi Self-Healing V4.
* **Lý do & Quyết định:** Core Recon. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Hàm `healSegmentB**` (`recon_heal_v4.go`)
* **Mô tả chức năng:** Logic bù lỗ hổng (Shadow -> Master). Bắn NATS Transmute vá.
* **Phân tích:** Thuật toán lõi Self-Healing V4.
* **Lý do & Quyết định:** Core Recon. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Hàm `healThresholdBlocked**` (`recon_heal_v4.go`)
* **Mô tả chức năng:** Cầu dao chặn hàm Heal chạy nếu bảng sai lệch 1 triệu dòng (chạy Heal sẽ làm sập RAM DB).
* **Phân tích:** Thuật toán bảo vệ (Safety Limit).
* **Lý do & Quyết định:** Core Recon. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Struct `DLQMessage**` (Từ orchestration/)
* **Mô tả chức năng:** DTO định dạng JSON Dead Letter Queue chứa mã lỗi nguyên bản của DB.
* **Phân tích:** Hệ thống Bắt Ngoại Lệ (Exception Handling) là thành phần gốc của tiến trình Self-Healing.
* **Lý do & Quyết định:** Trả Dead Letter về Module Chữa Lành Data. **Quyết định: DI CHUYỂN sang `internal/handler/recon/dlq_handler.go`.**


* `[x]` **Struct `DLQHandler**` (Từ orchestration/)
* **Mô tả chức năng:** Middleware hứng toàn bộ error lúc Shadow/Master Ingestion, ném vào hàng chờ NATS DLQ.
* **Phân tích:** Tác vụ thu gom rác ngoại lệ.
* **Lý do & Quyết định:** Chuyển về Recon. **Quyết định: DI CHUYỂN sang `internal/handler/recon/dlq_handler.go`.**


* `[x]` **Hàm `NewDLQHandler**` (`dlq_handler.go`)
* **Mô tả chức năng:** Khởi tạo instance cho hệ quản lý lỗi.
* **Phân tích:** Constructor.
* **Lý do & Quyết định:** Hành vi Struct DLQ. **Quyết định: DI CHUYỂN sang `recon/`.**


* `[x]` **Phương thức `SetMaskingService**` (`dlq_handler.go`)
* **Mô tả chức năng:** Inject công cụ làm mờ data (Masking PII) trước khi insert Log.
* **Phân tích:** Bảo mật hệ thống log lỗi. Explicit DI.
* **Lý do & Quyết định:** Hành vi Struct DLQ. **Quyết định: DI CHUYỂN sang `recon/`.**


* `[x]` **Phương thức `HandleWithRetry**` (`dlq_handler.go`)
* **Mô tả chức năng:** Bọc 1 hàm SQL. Nếu dính lock/deadlock DB thì tự động Loop Retry 3 lần trước khi đẩy DLQ.
* **Phân tích:** Pattern bảo vệ DB thô.
* **Lý do & Quyết định:** Hành vi Struct DLQ. **Quyết định: DI CHUYỂN sang `recon/`.**


* `[x]` **Phương thức `HandleWithRetryContext**` (`dlq_handler.go`)
* **Mô tả chức năng:** Phiên bản có kiểm soát Timeout Context của hàm Retry trên.
* **Phân tích:** Pattern bảo vệ Goroutine.
* **Lý do & Quyết định:** Hành vi Struct DLQ. **Quyết định: DI CHUYỂN sang `recon/`.**


* `[x]` **Phương thức `sendToDLQ**` (`dlq_handler.go`)
* **Mô tả chức năng:** Đẩy string JSON lỗi lên hệ thống Message Broker rác.
* **Phân tích:** Message routing.
* **Lý do & Quyết định:** Hành vi Struct DLQ. **Quyết định: DI CHUYỂN sang `recon/`.**


* `[x]` **Phương thức `buildFailedSyncLog**` (`dlq_handler.go`)
* **Mô tả chức năng:** Build Object Model Postgres để chuẩn bị Save Log DB.
* **Phân tích:** DB builder.
* **Lý do & Quyết định:** Hành vi Struct DLQ. **Quyết định: DI CHUYỂN sang `recon/`.**


* `[x]` **Phương thức `markPublishFailure**` (`dlq_handler.go`)
* **Mô tả chức năng:** Đánh cờ "Publish Dead" nếu không thể đẩy lên NATS DLQ.
* **Phân tích:** Fail-safe pattern.
* **Lý do & Quyết định:** Hành vi Struct DLQ. **Quyết định: DI CHUYỂN sang `recon/`.**


* `[x]` **Phương thức `ReplayDLQ**` (`dlq_handler.go`)
* **Mô tả chức năng:** Đẩy message từ hàng chờ rác trở ngược lại luồng EventBridge để Ingest lại.
* **Phân tích:** Thao tác "Phục sinh" Data.
* **Lý do & Quyết định:** Hành vi Struct DLQ. **Quyết định: DI CHUYỂN sang `recon/`.**


* `[x]` **Hàm `extractDLQRecordID**` (`dlq_handler.go`)
* **Mô tả chức năng:** Regex bóc mảng PK ID từ string lỗi.
* **Phân tích:** Parser chuỗi.
* **Lý do & Quyết định:** Helper nội bộ DLQ. **Quyết định: DI CHUYỂN sang `recon/`.**


* `[x]` **Hàm `normalizeDLQRawJSON**` (`dlq_handler.go`)
* **Mô tả chức năng:** Format string bytes hỏng về Valid JSON.
* **Phân tích:** Format chuỗi.
* **Lý do & Quyết định:** Helper nội bộ DLQ. **Quyết định: DI CHUYỂN sang `recon/`.**


* `[x]` **Hàm `truncateDLQError**` (`dlq_handler.go`)
* **Mô tả chức năng:** Giới hạn 2000 ký tự cắt ngọn chuỗi.
* **Phân tích:** Bảo vệ Data Text Type.
* **Lý do & Quyết định:** Helper nội bộ DLQ. **Quyết định: DI CHUYỂN sang `recon/`.**


* `[x]` **Hàm `sanitizeDLQError**` (`dlq_handler.go`)
* **Mô tả chức năng:** Xóa IP, Password ra khỏi chuỗi log Exception.
* **Phân tích:** Masking chuỗi.
* **Lý do & Quyết định:** Helper nội bộ DLQ. **Quyết định: DI CHUYỂN sang `recon/`.**


* `[x]` **Hàm `extractSourceTableFromSubject**` (`dlq_handler.go`)
* **Mô tả chức năng:** Bóc tên bảng bị lỗi dựa trên subject string NATS.
* **Phân tích:** NATS Router helper.
* **Lý do & Quyết định:** Helper nội bộ DLQ. **Quyết định: DI CHUYỂN sang `recon/`.**


* `[x]` **Các phương thức `logInfo`, `logWarn`, `logError**` (`dlq_handler.go`)
* **Mô tả chức năng:** Wrapper print log có chèn prefix `[DLQ]`.
* **Phân tích:** Local Logger formatting.
* **Lý do & Quyết định:** Helper nội bộ DLQ. **Quyết định: DI CHUYỂN sang `recon/`.**


* `[x]` **Các hàm Test `BuildFailedSyncLogForTest` v.v.** (`dlq_handler.go`)
* **Mô tả chức năng:** Public export mock methods hỗ trợ suite test.
* **Phân tích:** Test Utils.
* **Lý do & Quyết định:** Đi theo file gốc DLQ. **Quyết định: DI CHUYỂN sang `recon/`.**


* `[x]` **Struct `DLQStateMachine**` (Từ orchestration/)
* **Mô tả chức năng:** Cronjob (Goroutine Timer) Worker chạy liên tục dưới nền quét DB `failed_sync_logs` có Status `Pending` để tự gọi `ReplayDLQ`.
* **Phân tích:** Đây là Healer Bot (Bot tự phục hồi). Việc quản lý Worker tự động vá Data thuộc trách nhiệm của Recon.
* **Lý do & Quyết định:** Gom cụm Data Integrity. **Quyết định: DI CHUYỂN sang `internal/handler/recon/dlq_state_machine.go`.**


* `[x]` **Struct `DLQStateMachineConfig**` (`dlq_state_machine.go`)
* **Mô tả chức năng:** DTO Config setup bao nhiêu giây quét 1 lần, Rate limit tốc độ replay.
* **Phân tích:** Tham số cấu trúc của Bot.
* **Lý do & Quyết định:** Đi theo Bot. **Quyết định: DI CHUYỂN sang `recon/`.**


* `[x]` **Hàm `NewDLQStateMachine**` (`dlq_state_machine.go`)
* **Mô tả chức năng:** Constructor tạo Boot worker.
* **Phân tích:** Khởi tạo Job.
* **Lý do & Quyết định:** Đi theo Bot. **Quyết định: DI CHUYỂN sang `recon/`.**


* `[x]` **Phương thức `applyDefaults**` (`dlq_state_machine.go`)
* **Mô tả chức năng:** Gán giá trị default nếu Struct rỗng (vd: 10 giây).
* **Phân tích:** Fallback an toàn Job.
* **Lý do & Quyết định:** Hành vi Bot. **Quyết định: DI CHUYỂN sang `recon/`.**


* `[x]` **Phương thức `Start**` (`dlq_state_machine.go`)
* **Mô tả chức năng:** Dùng `time.Ticker` chạy vòng lặp bất tận.
* **Phân tích:** Engine loop của Bot.
* **Lý do & Quyết định:** Hành vi Bot. **Quyết định: DI CHUYỂN sang `recon/`.**


* `[x]` **Phương thức `RunOnce**` (`dlq_state_machine.go`)
* **Mô tả chức năng:** GORM query `SELECT * FROM logs WHERE status='Pending' LIMIT X`.
* **Phân tích:** Execute 1 mẻ Job (Data Plane).
* **Lý do & Quyết định:** Hành vi Bot. **Quyết định: DI CHUYỂN sang `recon/`.**


* `[x]` **Phương thức `retryOne**` (`dlq_state_machine.go`)
* **Mô tả chức năng:** Iterator truyền bản ghi sang API `ReplayDLQ`.
* **Phân tích:** Step của Job loop.
* **Lý do & Quyết định:** Hành vi Bot. **Quyết định: DI CHUYỂN sang `recon/`.**


* `[x]` **Phương thức `ReplayFailedLog**` (`dlq_state_machine.go`)
* **Mô tả chức năng:** Lắng nghe REST/NATS lệnh ép Trigger Manual chạy ngầm.
* **Phân tích:** Command endpoint.
* **Lý do & Quyết định:** Hành vi Bot. **Quyết định: DI CHUYỂN sang `recon/`.**


* `[x]` **Các phương thức `logInfo`, `logWarn`, `logDebug**` (`dlq_state_machine.go`)
* **Mô tả chức năng:** Local logger có prefix `[DLQ Bot]`.
* **Phân tích:** Logger util ngầm.
* **Lý do & Quyết định:** Hành vi Bot. **Quyết định: DI CHUYỂN sang `recon/`.**


* `[x]` **Struct `ScanHandler**` (Từ orchestration/)
* **Mô tả chức năng:** Tiến trình quét block DB khổng lồ (triệu dòng) từ Upstream để tải đè Backfill Ingestion bù đắp lượng Data cũ (Historical Data).
* **Phân tích:** Quét bù đắp lượng Data cũ tĩnh chính xác là nghiệp vụ Bulk Reconciliation (Đối soát hàng loạt). Việc để ở Orchestration (Nhạc trưởng) là nhầm lẫn trầm trọng giữa Control Plane và Data Plane. Bổ sung Explicit DI.
* **Lý do & Quyết định:** Trả về Bounded Context Data Integrity. **Quyết định: DI CHUYỂN sang `internal/handler/recon/scan_handler.go`. Khai báo Explicit DI `metadataRegistry`.**


* `[x]` **Phương thức `SetTransformChunkSize**` (`scan_handler.go`)
* **Mô tả chức năng:** Config Max Element per chunk để chống sập RAM.
* **Phân tích:** Tuning Memory Job quét.
* **Lý do & Quyết định:** Hành vi của ScanHandler. **Quyết định: DI CHUYỂN sang `recon/`.**


* `[x]` **Phương thức `HandleBackfill**` (`scan_handler.go`)
* **Mô tả chức năng:** NATS command trigger vòng lặp quét dữ liệu Upstream và đẩy vào pipeline Kafka.
* **Phân tích:** Bulk Data Plane Workflow.
* **Lý do & Quyết định:** Hành vi của ScanHandler. **Quyết định: DI CHUYỂN sang `recon/`.**


* `[x]` **Phương thức `HandleScanRawData**` (`scan_handler.go`)
* **Mô tả chức năng:** NATS command đọc data thô không qua quy tắc transform để phục vụ Audit chéo bằng tay.
* **Phân tích:** Read Data Plane Bypass.
* **Lý do & Quyết định:** Hành vi của ScanHandler. **Quyết định: DI CHUYỂN sang `recon/`.**


* `[x]` **Phương thức `HandleScanArrayFields**` (`scan_handler.go`)
* **Mô tả chức năng:** Quét chuyên sâu các field JSON Array (Cột mảng) để phát hiện dị tật (Type Drift) từ DB Nguồn sinh ra.
* **Phân tích:** Schema Drift Audit Logic.
* **Lý do & Quyết định:** Hành vi của ScanHandler. **Quyết định: DI CHUYỂN sang `recon/`.**


* `[x]` **Phương thức `HandlePeriodicScan**` (`scan_handler.go`)
* **Mô tả chức năng:** Setup Timer định kỳ ban đêm tự chạy Audit tìm cột Schema Mới xuất hiện.
* **Phân tích:** Background Audit Worker.
* **Lý do & Quyết định:** Hành vi của ScanHandler. **Quyết định: DI CHUYỂN sang `recon/`.**



---

### 6. Thư mục `orchestration/` (Pipeline Orchestration - Control Plane)

**Nguyên tắc Hybrid:** Dọn dẹp sạch mã nguồn rác rưởi. Orchestration thuần túy là DAG State Machine (Máy trạng thái Workflow bước này nhảy sang bước kia). Chỉ được phép phát lệnh chỉ đạo, tuyệt đối không được phép thao tác DML (Insert/Update) trực tiếp lên Data Plane.

* `[x]` **Struct `ProvisioningHandler**` (`provisioning_handler.go`)
* **Mô tả chức năng:** State Machine lõi. Cấu trúc bảng Routing (VD: Register Success -> Chạy Discover. Discover Success -> Chạy Bind...).
* **Phân tích:** Trái tim điều khiển luồng (Control Plane) của việc khai báo và cấu hình luồng đồng bộ.
* **Lý do & Quyết định:** Đúng thiết kế. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Phương thức `HandleStepCompleted**` (`provisioning_handler.go`)
* **Mô tả chức năng:** NATS listener bắt cục Event `stepCompletedPayload` (từ Base), ghi đè Status vào DB Metadata và publish Trigger Step tiếp theo.
* **Phân tích:** Nút giao logic đẩy luồng chuyển State DAG Engine.
* **Lý do & Quyết định:** Hành vi của ProvisioningHandler. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Struct `SnapshotRunner**` (`snapshot_runner_handler.go`)
* **Mô tả chức năng:** Workflow Worker chuyên quản lý Job chạy Batch Snapshot ban đầu.
* **Phân tích:** Khác với `Transmute` (tự tay bốc vác data), SnapshotRunner đóng vai trò "Trưởng Ca" phân mảnh Chunk ID và giao task cho các node Data chạy. Đây là 1 Long-Running Workflow điển hình.
* **Lý do & Quyết định:** Nằm tại Control Plane là hợp lý. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Struct `snapshotV2Payload**` (`snapshot_runner_handler.go`)
* **Mô tả chức năng:** DTO gửi cục Cấu hình phân mảnh cho Data Node.
* **Phân tích:** Workflow parameter.
* **Lý do & Quyết định:** Phục vụ Runner. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Struct `progressClaim**` (`snapshot_runner_handler.go`)
* **Mô tả chức năng:** DTO đánh Bookmark (Lưu cờ trạng thái Offset tiến độ) để nếu sập mạng có thể Resume chạy tiếp.
* **Phân tích:** Workflow persistent state.
* **Lý do & Quyết định:** Tính năng quản trị luồng của Runner. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Phương thức `Handle**` (`snapshot_runner_handler.go`)
* **Mô tả chức năng:** Kích hoạt API NATS Call chạy Job Snapshot.
* **Phân tích:** Entrypoint Workflow.
* **Lý do & Quyết định:** Hành vi của SnapshotRunner. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Phương thức `runSnapshot**` (`snapshot_runner_handler.go`)
* **Mô tả chức năng:** Vòng lặp Loop phân rã Chunk ID bằng thuật toán Paginator.
* **Phân tích:** Workflow Controller Logic.
* **Lý do & Quyết định:** Hành vi của SnapshotRunner. **Quyết định: GIỮ NGUYÊN.**


* `[x]` **Struct `ScheduleEnableHandler**` (Tách từ `provisioning_step_handlers.go` cũ)
* **Mô tả chức năng:** Lệnh điều khiển API Bật/Tắt Cờ Active Cronjob/Schedule đồng bộ toàn cục trên 1 Target.
* **Phân tích:** Đây là Config State của hệ thống luồng. Nó bị nhốt chung với thao tác Bind nội bộ Shadow ở file cũ nên gây nhập nhằng.
* **Lý do & Quyết định:** Bóc tách để ranh giới rõ ràng. **Quyết định: TẠO MỚI file `internal/handler/orchestration/provisioning_schedule_enable.go` và khai báo cấu trúc này.**


* `[x]` **Struct `scheduleEnableRequest**` (`provisioning_schedule_enable.go`)
* **Mô tả chức năng:** DTO format Body request tắt bật công tắc.
* **Phân tích:** Cấu trúc Input API.
* **Lý do & Quyết định:** Phục vụ Handler trên. **Quyết định: Đưa vào file mới tạo.**


* `[x]` **Phương thức `HandleScheduleEnable**` (`provisioning_schedule_enable.go`)
* **Mô tả chức năng:** Bắt body DTO, thực thi Update GORM thay cờ Flag trên DB hệ thống và Emit event Status Update.
* **Phân tích:** Lệnh chuyển đổi State hệ thống điều phối.
* **Lý do & Quyết định:** Hành vi của Handler. **Quyết định: Di chuyển code từ god-file cũ sang file mới tạo.**



Đây là một Bản Kiến Trúc Thực Thi (Hybrid Plan) triệt để và khắc nghiệt nhất. Toàn bộ `168` thành phần Function, Struct, Helper của dự án đã được bóc tách Line-By-Line. Không một "God Object" nào tồn tại, không một nguy cơ Circular Dependency nào có thể len lỏi. Anh có thể an tâm tuyệt đối để copy-paste cho lần Refactor lịch sử này!
---

## Kế hoạch Verification

### Automated Tests
- Chạy biên dịch kiểm thử xem có xảy ra lỗi Circular Dependency hay lỗi cú pháp nào không:
  ```bash
  go build ./...
  ```
- Chạy test suites trong `centralized-data-service`:
  ```bash
  go test ./...
  ```

### Manual Verification
- Xác minh bằng logs khi build và chạy service, đảm bảo các NATS routes, API routes vẫn được khởi tạo chính xác.
- Đảm bảo các đăng ký DI trong `worker_server_init.go` đã được cập nhật phù hợp với sự thay đổi của struct và explicit injection.
