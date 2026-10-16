# 12 Kế Hoạch Triển Khai Chi Tiết (Implementation Plan - Finalized)

## Mục Tiêu
Khắc phục triệt để lỗi drop dữ liệu CDC do ngộ nhận tiền tố môi trường `goopaylocal`, ngăn chặn việc Kafka commit offset khi chưa ghi dữ liệu, bảo toàn dữ liệu bằng Dead Letter Queue (DLQ), khôi phục 212 bản ghi bị mất trong `shadow_traitestctphs.trans_his` và thiết lập cơ chế kiểm định trên DB thật.

## Các Bước Triển Khai Kỹ Thuật Thực Tế

### Bước 1: Nạp Động Generic Topic Prefix từ Cấu hình Hệ thống (Config-Driven)
- **Tệp tin:** `internal/service/source/metadata_registry_utils.go`
- **Nội dung:** Bổ sung `RegisterGenericPrefixes(prefixes []string)` để đăng ký động các tiền tố từ `cfg.Kafka.TopicPrefix` (`cdc.gpaylocal`, `cdc.goopaylocal`, `cdc.mariadblocal`, `cdc.sftplocal`). Cập nhật `IsGenericConnectionCode` tra cứu an toàn qua `sync.RWMutex`. Tuyệt đối không hardcode `"local"`, `"dev"`, `"staging"`, `"prod"`.

### Bước 2: Khởi động Đăng ký Tiền Tố tại Server Setup
- **Tệp tin:** `internal/server/server_setup.go`
- **Nội dung:** Gọi `servicesource.RegisterGenericPrefixes(cfg.Kafka.TopicPrefix)` ngay tại dòng 97 trước khi khởi tạo pipeline CDC.

### Bước 3: Chuẩn Hóa Cắt Tiền Tố Động & Reset Generic Connection tại Kafka Consumer
- **Tệp tin:** `internal/handler/shadow/kafka_consumer.go`
- **Nội dung:** Duyệt qua mảng `kc.config.TopicPrefix` để cắt sạch tiền tố môi trường động. Nếu connection code vẫn là generic engine/platform (`IsGenericConnectionCode == true`), tự động reset `sourceConnCode = ""` để không gây over-filtering.

### Bước 4: Triệt Tiêu Nuốt Lỗi (Anti-Silent Drop) tại Event Handler
- **Tệp tin:** `internal/handler/shadow/event_handler.go`
- **Nội dung:** Khi không tìm thấy route trong cache (`len(routes) == 0`), thay vì `return 0, nil` nuốt lỗi, đổi thành `return 0, fmt.Errorf("source route not found in registry cache...")`. Bắt buộc `KafkaConsumer` nhận `procErr != nil` và đẩy vào DLQ (`failed_sync_logs`), không cho phép commit offset mù quáng khi `rows == 0`.

### Bước 5: Kiểm Thử Đơn Vị & Tích Hợp Live DB Không Mockup (DoD G3/G6)
- **Unit Test:** `internal/service/source/metadata_registry_service_test.go` (`TestRegisterGenericPrefixes`, `TestIsGenericConnectionCode`, `TestResolveSourceRoutes_GenericConnectionFallback`).
- **Live DB Integration Test:** `test/internal/service/routing_live_test.go` kết nối PostgreSQL `cdc_dw` cổng 5433 thật, kiểm chứng phân giải 3 routes thật của `trans-his` và kiểm chứng `EventHandler` bắt buộc trả về explicit error.
- **Biên dịch thật:** `go build -o bin/worker ./cmd/worker/` và `go build -o bin/admin-api ./cmd/admin-api/`.

### Bước 6: Bù Dữ Liệu Bị Thiếu (Recon Heal)
- Restart worker nhị phân mới và kích hoạt `POST http://localhost:8082/api/reconciliation/heal/shadow_traitestctphs.trans_his` để bù 212 bản ghi.
