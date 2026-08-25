# Kịch bản giải pháp: Đồng bộ luồng Active/Deactivate SFTP Connector từ cả Registry và Shadow Binding

Tài liệu đặc tả chi tiết các phần code cần Muscle chỉnh sửa trong dự án `cdc-cms-service`.

---

## 1. File `internal/app/commands/source/update_registry.go`

Thực hiện bổ sung các dependency và logic gọi khởi chạy/xóa connector khi `IsActive` thay đổi.

```go
package source

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"cdc-cms-service/internal/app/ports"
	model "cdc-cms-service/internal/model/source"
)

type KafkaConnectorWriter interface {
	Create(ctx context.Context, name string, cfg map[string]string) (map[string]any, error)
	UpdateConfig(ctx context.Context, name string, cfg map[string]string) (map[string]any, error)
	Delete(ctx context.Context, name string) error
}

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

type UpdateRegistryHandler struct {
	sourceRepo    ports.SourceRepo
	logger        ports.ActivityLogger
	nats          ports.ReloadPublisher
	connectorRepo ports.SystemConnectorRepo
	writer        KafkaConnectorWriter
	db            *gorm.DB
	log           *zap.Logger
}

func NewUpdateRegistryHandler(
	sourceRepo ports.SourceRepo,
	logger ports.ActivityLogger,
	nats ports.ReloadPublisher,
	connectorRepo ports.SystemConnectorRepo,
	writer KafkaConnectorWriter,
	db *gorm.DB,
	log *zap.Logger,
) *UpdateRegistryHandler {
	return &UpdateRegistryHandler{
		sourceRepo:    sourceRepo,
		logger:        logger,
		nats:          nats,
		connectorRepo: connectorRepo,
		writer:        writer,
		db:            db,
		log:           log,
	}
}

func (h *UpdateRegistryHandler) Handle(ctx context.Context, c ports.Command) (json.RawMessage, error) {
	cmd, ok := c.(UpdateRegistryCommand)
	if !ok {
		return nil, errors.New("registry.update: command type mismatch")
	}
	if h.sourceRepo == nil {
		return nil, errors.New("registry store not ready")
	}

	updates := map[string]interface{}{}
	if cmd.SyncEngine != nil {
		updates["sync_engine"] = *cmd.SyncEngine
	}
	if cmd.SyncInterval != nil {
		updates["sync_interval"] = *cmd.SyncInterval
	}
	if cmd.Priority != nil {
		updates["priority"] = *cmd.Priority
	}
	if cmd.IsActive != nil {
		updates["is_active"] = *cmd.IsActive
	}
	if cmd.Notes != nil {
		updates["notes"] = *cmd.Notes
	}
	if cmd.TimestampField != nil {
		updates["timestamp_field"] = *cmd.TimestampField
	}

	existing, autoApprovedCount, err := h.sourceRepo.UpdateRegistry(ctx, cmd.ID, updates)
	if err != nil {
		if errors.Is(err, ports.ErrRecordNotFound) {
			return nil, ErrRegistryNotFound
		}
		return nil, err
	}

	// Logic xử lý trì hoãn SFTP Connector khi active/deactivate từ Registry
	engine := strings.ToLower(existing.SourceType)
	isSFTP := engine == "sftp" || engine == "file" || engine == "csv"

	if isSFTP && cmd.IsActive != nil && h.writer != nil && h.connectorRepo != nil && h.db != nil {
		var connectorName string
		var optionsJSONStr string
		if existing.SourceConnectionID != nil {
			_ = h.db.Raw("SELECT connection_code, options_json::text FROM cdc_system.connection_registry WHERE id = ?", *existing.SourceConnectionID).Row().Scan(&connectorName, &optionsJSONStr)
		}

		if connectorName != "" {
			if *cmd.IsActive {
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

				if optionsJSONStr != "" {
					var opts map[string]string
					_ = json.Unmarshal([]byte(optionsJSONStr), &opts)
					username := opts["username"]
					password := opts["password"]

					if fsUri, ok := rawConfig["fs.uris"]; ok && strings.Contains(fsUri, "sftp://***:***@") {
						fullUri := strings.Replace(fsUri, "sftp://***:***@", fmt.Sprintf("sftp://%s:%s@", username, password), 1)
						rawConfig["fs.uris"] = fullUri
					}
				}

				if len(rawConfig) > 0 {
					h.log.Info("activating SFTP connector on Kafka Connect from registry update", zap.String("connector", connectorName))
					if topic := rawConfig["topic"]; topic != "" {
						bootstrap := rawConfig["signal.kafka.bootstrap.servers"]
						autoCreateKafkaTopic(ctx, bootstrap, topic, h.log)
					}

					_, err = h.writer.Create(ctx, connectorName, rawConfig)
					if err != nil {
						_, err = h.writer.UpdateConfig(ctx, connectorName, rawConfig)
					}
					if err != nil {
						h.log.Error("failed to create/update SFTP connector on active registry update", zap.Error(err))
					}
				}
			} else {
				h.log.Info("deactivating SFTP connector from Kafka Connect from registry update", zap.String("connector", connectorName))
				err = h.writer.Delete(ctx, connectorName)
				if err != nil && !strings.Contains(err.Error(), "404") {
					h.log.Warn("failed to delete SFTP connector on deactivate from registry update", zap.Error(err))
				}
			}
		}
	}

	if cmd.IsActive != nil && *cmd.IsActive && autoApprovedCount > 0 {
		if h.logger != nil {
			h.logger.LogAsync(ports.ActivityEntry{
				Operation:   "auto-approve-fields",
				TargetTable: existing.TargetTable,
				Status:      "success",
				Details: map[string]interface{}{
					"fields_approved": autoApprovedCount,
					"source_table":    existing.SourceTable,
					"trigger":         "inactive→active",
				},
				TriggeredBy: "manual",
			})
		}
		if h.nats != nil {
			_ = h.nats.PublishReload(ctx, existing.TargetTable, cmd.UpdatedBy, "auto_approve", "")
		}
	}

	dispatched := []string{}
	if h.logger != nil {
		h.logger.LogAsync(ports.ActivityEntry{
			Operation:   "registry-update",
			TargetTable: existing.TargetTable,
			Status:      "accepted",
			Details: map[string]interface{}{
				"updates":    updates,
				"user":       cmd.UpdatedBy,
				"dispatched": dispatched,
			},
			TriggeredBy: "manual",
		})
	}
	if h.nats != nil {
		_ = h.nats.PublishReload(ctx, existing.TargetTable, cmd.UpdatedBy, "update", "")
	}

	body, _ := json.Marshal(map[string]interface{}{
		"message":    "updated — external state dispatched",
		"entry":      existing,
		"dispatched": dispatched,
	})
	return body, nil
}
```

---

## 2. File `internal/app/commands/shadow/update_shadow_binding.go`

Sửa đổi hàm `Handle` của `UpdateShadowBindingHandler` để khi toggle active trạng thái binding trực tiếp, nó tự động đồng bộ sang bảng legacy `cdc_table_registry` và `source_object_registry`.

```go
	// 3. Thực hiện cập nhật trạng thái is_active trong DB cho shadow_binding
	rowsAffected, err := h.repo.UpdateActiveStatus(ctx, cmd.ID, *cmd.IsActive)
	if err != nil {
		h.logger.Error("update shadow binding failed",
			zap.Int64("shadow_binding_id", cmd.ID), zap.Error(err))
		return nil, err
	}
	if rowsAffected == 0 {
		return nil, ErrShadowBindingNotFound
	}

	// ĐỒNG BỘ NGƯỢC: Cập nhật is_active cho cdc_table_registry
	_ = h.db.Exec(`
		UPDATE cdc_system.cdc_table_registry
		SET is_active = ?
		WHERE source_db = ? AND source_table = ?
	`, *cmd.IsActive, so.Scope.Database, so.Scope.Table).Error

	// ĐỒNG BỘ NGƯỢC: Cập nhật is_active cho source_object_registry
	_ = h.db.Exec(`
		UPDATE cdc_system.source_object_registry
		SET is_active = ?, profile_status = ?
		WHERE id = ?
	`, *cmd.IsActive, func() string {
		if *cmd.IsActive {
			return "active"
		}
		return "paused"
	}(), binding.SourceObjectID).Error
```

---

## 3. File `internal/server/server.go`

Cập nhật dòng đăng ký `"registry.update"`:

```go
	cmdBus.RegisterSync("registry.update", sourceCmd.NewUpdateRegistryHandler(
		sourceObjectRepo,
		activityLogger,
		reloadPublisher,
		systemConnectorRepo,
		kafkaConnectClient,
		db,
		logger,
	))
```
