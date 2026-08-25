# Kịch bản giải pháp: Truyền danh sách Kafka Brokers cấu hình từ Config vào SnapshotRunner của Worker

Tài liệu đặc tả chi tiết mã nguồn cần chỉnh sửa trong dự án `centralized-data-service` để giải quyết việc cdc-worker bỏ qua tạo topic SFTP do thiếu cấu hình địa chỉ Kafka Brokers.

---

## 1. Nguyên nhân lỗi (Root Cause)
- **Thiếu biến cấu hình tại Runtime:** SFTP là connector loại File Stream (không dùng Debezium signals), cấu hình của nó không chứa trường `"signal.kafka.bootstrap.servers"`.
- **Môi trường cục bộ không có Env:** Khi chạy local/staging bằng tệp cấu hình YAML, shell environment hoàn toàn không có biến `KAFKA_BROKERS` hay `CMS_SYSTEM_SIGNAL_KAFKA_BOOTSTRAP`.
- **Hệ quả:** Logic cũ rơi vào case fallback env bị rỗng, dẫn đến ghi nhận cảnh báo `auto-create kafka topic skipped: no kafka brokers configured in environment` và không tạo topic.

---

## 2. Các file cần sửa đổi trong `centralized-data-service`

### 1. File `internal/handler/orchestration/snapshot_runner_handler.go`

Cập nhật struct `SnapshotRunner` và constructor `NewSnapshotRunner` để nhận thêm tham số `kafkaBrokers []string`:

```go
type SnapshotRunner struct {
	db           *gorm.DB
	eventHandler snapshotEventHandler
	registrySvc  metadata.MetadataRegistry
	connRepo     *reposource.ConnectionRegistryRepo
	soRepo       *reposource.SourceObjectRegistryRepo
	shadowRepo   *reposhadow.ShadowBindingRepo
	natsConn     *nats.Conn
	logger       *zap.Logger
	kafkaBrokers []string
}

func NewSnapshotRunner(
	db *gorm.DB,
	eventHandler snapshotEventHandler,
	registrySvc metadata.MetadataRegistry,
	connRepo *reposource.ConnectionRegistryRepo,
	soRepo *reposource.SourceObjectRegistryRepo,
	shadowRepo *reposhadow.ShadowBindingRepo,
	natsConn *nats.Conn,
	logger *zap.Logger,
	kafkaBrokers []string,
) *SnapshotRunner {
	return &SnapshotRunner{
		db:           db,
		eventHandler: eventHandler,
		registrySvc:  registrySvc,
		connRepo:     connRepo,
		soRepo:       soRepo,
		shadowRepo:   shadowRepo,
		natsConn:     natsConn,
		logger:       logger,
		kafkaBrokers: kafkaBrokers,
	}
}
```

Tại hàm `runSnapshot`, nếu `bootstrap` từ raw config bị rỗng, fallback sang `r.kafkaBrokers`:

```go
		if len(rawConfig) > 0 {
			topic := rawConfig["topic"]
			bootstrap := rawConfig["signal.kafka.bootstrap.servers"]
			if bootstrap == "" && len(r.kafkaBrokers) > 0 {
				bootstrap = strings.Join(r.kafkaBrokers, ",")
			}
			if topic != "" {
				autoCreateKafkaTopic(ctx, bootstrap, topic, r.logger)
			}
		}
```

### 2. File `internal/server/server_setup.go`

Truyền thêm tham số `cfg.Kafka.Brokers` vào cuộc gọi khởi tạo `NewSnapshotRunner`:

```go
	snapshotRunner := handlerorchestration.NewSnapshotRunner(
		db,
		eventHandler,
		registrySvc,
		connectionRepo,
		sourceObjectRepo,
		shadowBindingRepo,
		natsClient.Conn,
		logger,
		cfg.Kafka.Brokers,
	)
```
