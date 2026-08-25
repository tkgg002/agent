# Giải pháp Kỹ thuật: Trì hoãn khởi tạo SFTP Connector theo sự kiện Active Binding

Tài liệu này đặc tả chi tiết các phần mã nguồn cần thay đổi, comment lại hoặc bổ sung mới để triển khai giải pháp trì hoãn tạo Kafka Connector cho nguồn SFTP cho tới khi người dùng bấm kích hoạt liên kết (`Active Binding`).

---

## 1. Các file cần thay đổi (Files to Modify)

1. `cdc-cms-service/internal/app/commands/source/debezium_connector.go`
   - **Mục tiêu:** 
     - Bỏ qua bước gọi HTTP sang Kafka Connect để tạo connector khi cấu hình SFTP Source.
     - Xóa bỏ hoàn toàn hàm `autoCreateKafkaTopic` và logic tự tạo topic tại bước đăng ký connection này (dọn dẹp các thư viện `os` và `kafka-go` không còn sử dụng).
2. `cdc-cms-service/internal/app/commands/shadow/update_shadow_binding.go`
   - **Mục tiêu:** 
     - Bổ sung hàm `autoCreateKafkaTopic` để tự động tạo topic Kafka cho nguồn SFTP.
     - Triển khai logic tạo topic và tạo connector thực tế trên Kafka Connect chỉ khi `IsActive` được bật lên `true`.
     - Xóa connector khỏi Kafka Connect khi `IsActive` tắt về `false`.
3. `cdc-cms-service/internal/server/server.go`
   - **Mục tiêu:** Bổ sung các dependency (`sourceObjectRepo`, `systemConnectorRepo`, `kafkaConnectClient`, `db`) vào hàm khởi tạo `NewUpdateShadowBindingHandler`.

---

## 2. Chi tiết thay đổi và Mã nguồn Demo

### 2.1 File `debezium_connector.go`

#### Code cũ cần comment lại / chỉnh sửa:
Tại hàm `Handle` của `CreateSystemConnectorHandler`, phần logic gửi yêu cầu tạo connector lên Kafka Connect:

```go
// Line 158-180:
				if isSFTP {
					if topic := cmd.Config["topic"]; topic != "" {
						bootstrap := cmd.Config["signal.kafka.bootstrap.servers"]
						if bootstrap == "" {
							bootstrap = h.signalKafkaBootstrap
						}
						if bootstrap == "" {
							bootstrap = os.Getenv("KAFKA_BROKERS")
						}
						if bootstrap == "" {
							bootstrap = os.Getenv("CMS_SYSTEM_SIGNAL_KAFKA_BOOTSTRAP")
						}
						autoCreateKafkaTopic(ctx, bootstrap, topic, h.logger)
					}
					// ĐỐI VỚI SFTP: Bỏ qua không gọi h.writer.Create.
					// Trả về response giả lập để Saga hoàn thành việc ghi DB.
					resp = map[string]any{
						"name":   cmd.Name,
						"config": cmd.Config,
						"status": "deferred_provisioning",
					}
					return nil
				}
```

#### Code mới thay thế (Xóa bỏ hoàn toàn `autoCreateKafkaTopic`):
```go
				if isSFTP {
					// ĐỐI VỚI SFTP: Bỏ qua hoàn toàn việc tạo topic và tạo connector trên Kafka Connect.
					// Trả về response giả lập để Saga hoàn thành việc ghi DB.
					resp = map[string]any{
						"name":   cmd.Name,
						"config": cmd.Config,
						"status": "deferred_provisioning",
					}
					return nil
				}
```

*Lưu ý:* Xóa hàm `autoCreateKafkaTopic` ở đầu file và các import `"github.com/segmentio/kafka-go"` và `"os"` không còn sử dụng.

---

### 2.2 File `update_shadow_binding.go`

#### Bổ sung hàm `autoCreateKafkaTopic` vào đầu file:
```go
func autoCreateKafkaTopic(ctx context.Context, bootstrapServers, topic string, logger *zap.Logger) {
	if topic == "" {
		return
	}
	if bootstrapServers == "" {
		bootstrapServers = os.Getenv("KAFKA_BROKERS")
		if bootstrapServers == "" {
			bootstrapServers = os.Getenv("CMS_SYSTEM_SIGNAL_KAFKA_BOOTSTRAP")
		}
	}
	if strings.TrimSpace(bootstrapServers) == "" {
		if logger != nil {
			logger.Warn("auto-create kafka topic skipped: no kafka brokers configured in environment")
		}
		return
	}

	brokers := strings.Split(bootstrapServers, ",")
	for _, broker := range brokers {
		broker = strings.TrimSpace(broker)
		if broker == "" {
			continue
		}
		conn, err := kafka.DialContext(ctx, "tcp", broker)
		if err != nil {
			continue
		}
		topicConfig := kafka.TopicConfig{
			Topic:             topic,
			NumPartitions:     1,
			ReplicationFactor: 1,
		}
		err = conn.CreateTopics(topicConfig)
		conn.Close()
		if err == nil {
			if logger != nil {
				logger.Info("auto-created kafka topic for connector",
					zap.String("topic", topic), zap.String("broker", broker))
			}
			return
		}
	}
}
```

*Lưu ý:* Bổ sung các gói import: `"github.com/segmentio/kafka-go"`, `"os"`, `"context"`.

#### Khai báo Struct mới (Bổ sung Dependencies):
```go
type UpdateShadowBindingHandler struct {
	repo          ports.ShadowBindingRepo
	sourceRepo    ports.SourceRepo
	connectorRepo ports.SystemConnectorRepo
	writer        source.KafkaConnectorWriter // hoặc interface narrow tương tự
	db            *gorm.DB
	logger        *zap.Logger
}
```

#### Logic điều khiển tại hàm `Handle`:

```go
	if isSFTP && h.writer != nil && h.connectorRepo != nil && h.db != nil {
		connectorName := so.ConnectionCode // connection_code chính là tên connector

		if *cmd.IsActive {
			// A. KÍCH HOẠT: Tái dựng cấu hình & Tạo/Chạy Connector trên Kafka Connect
			
			// Lấy cấu hình thô đã che (sanitized) từ cdc_sources
			var rawConfig map[string]string
			sources, err := h.connectorRepo.List(ctx)
			if err == nil {
				for i := range sources {
					if sources[i].ConnectorName == connectorName {
						_ = json.Unmarshal(sources[i].RawConfigSanitized, &rawConfig)
						break
					}
				}
			}

			// Lấy thông tin mật khẩu giải mã từ connection_registry
			var optionsJSONStr string
			err = h.db.Raw("SELECT options_json::text FROM cdc_system.connection_registry WHERE connection_code = ?", connectorName).Scan(&optionsJSONStr).Error
			if err == nil && optionsJSONStr != "" {
				var opts map[string]string
				_ = json.Unmarshal([]byte(optionsJSONStr), &opts)
				
				username := opts["username"]
				password := opts["password"]
				
				// Reconstruct fs.uris (Chèn lại mật khẩu thực tế vào vị trí bị ẩn)
				if fsUri, ok := rawConfig["fs.uris"]; ok && strings.Contains(fsUri, "sftp://***:***@") {
					fullUri := strings.Replace(fsUri, "sftp://***:***@", fmt.Sprintf("sftp://%s:%s@", username, password), 1)
					rawConfig["fs.uris"] = fullUri
				}
			}

			if len(rawConfig) > 0 {
				// TẠO TOPIC KAFKA TRƯỚC KHI CHẠY CONNECTOR
				if topic := rawConfig["topic"]; topic != "" {
					bootstrap := rawConfig["signal.kafka.bootstrap.servers"]
					if bootstrap == "" {
						bootstrap = os.Getenv("KAFKA_BROKERS")
					}
					if bootstrap == "" {
						bootstrap = os.Getenv("CMS_SYSTEM_SIGNAL_KAFKA_BOOTSTRAP")
					}
					autoCreateKafkaTopic(ctx, bootstrap, topic, h.logger)
				}

				h.logger.Info("activating SFTP connector on Kafka Connect", zap.String("connector", connectorName))
				// Gọi Kafka Connect để tạo connector thực tế
				_, err = h.writer.Create(ctx, connectorName, rawConfig)
				if err != nil {
					// Nếu đã tồn tại, update cấu hình
					_, err = h.writer.UpdateConfig(ctx, connectorName, rawConfig)
				}
				if err != nil {
					h.logger.Error("failed to create/update SFTP connector on active binding", zap.Error(err))
					// Rollback trạng thái is_active về false nếu tạo connector thất bại
					_, _ = h.repo.UpdateActiveStatus(ctx, cmd.ID, false)
					return nil, fmt.Errorf("active_binding failed: cannot provision kafka connector: %w", err)
				}
			}

		} else {
			// B. HỦY KÍCH HOẠT: Xóa Connector khỏi Kafka Connect để dừng hoàn toàn việc đọc file
			h.logger.Info("deactivating SFTP connector from Kafka Connect", zap.String("connector", connectorName))
			err = h.writer.Delete(ctx, connectorName)
			if err != nil && !strings.Contains(err.Error(), "404") {
				h.logger.Warn("failed to delete SFTP connector on deactivate", zap.Error(err))
			}
		}
	}
```

---

### 2.3 File `server.go`

Giữ nguyên đăng ký dependency cho `shadow-binding.update` như phiên trước:
```go
cmdBus.RegisterSync("shadow-binding.update", shadowCmd.NewUpdateShadowBindingHandler(
    shadowBindingRepo, 
    sourceObjectRepo, 
    systemConnectorRepo, 
    kafkaConnectClient, 
    db, 
    logger,
))
```
