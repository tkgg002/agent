# Kế hoạch triển khai chi tiết của AI: Di chuyển tạo Topic Kafka SFTP sang nút Snapshot

Tài liệu thiết kế chi tiết mã nguồn cho giải pháp rẽ nhánh Snapshot của SFTP.

---

## 1. Thiết kế mã nguồn chi tiết

### A. Revert thay đổi trong `internal/app/commands/shadow/update_shadow_binding.go`
Quay về cấu trúc nguyên bản:
```go
package shadow

import (
	"context"
	"encoding/json"
	"errors"

	"go.uber.org/zap"

	"cdc-cms-service/internal/app/ports"
)

var ErrShadowBindingNotFound = errors.New("shadow binding not found")

type UpdateShadowBindingCommand struct {
	ports.SyncCommandMixin
	ID       int64 `json:"id"`
	IsActive *bool `json:"is_active"`
}

func (UpdateShadowBindingCommand) Type() string { return "shadow-binding.update" }

func (c UpdateShadowBindingCommand) Validate() error {
	if c.ID <= 0 {
		return errors.New("invalid_shadow_binding_id")
	}
	if c.IsActive == nil {
		return errors.New("is_active_required")
	}
	return nil
}

type UpdateShadowBindingHandler struct {
	repo   ports.ShadowBindingRepo
	logger *zap.Logger
}

func NewUpdateShadowBindingHandler(repo ports.ShadowBindingRepo, logger *zap.Logger) *UpdateShadowBindingHandler {
	return &UpdateShadowBindingHandler{repo: repo, logger: logger}
}

func (h *UpdateShadowBindingHandler) Handle(ctx context.Context, c ports.Command) (json.RawMessage, error) {
	cmd, ok := c.(UpdateShadowBindingCommand)
	if !ok {
		return nil, errors.New("shadow-binding.update: command type mismatch")
	}
	if h.repo == nil {
		return nil, errors.New("shadow binding store not ready")
	}

	rowsAffected, err := h.repo.UpdateActiveStatus(ctx, cmd.ID, *cmd.IsActive)
	if err != nil {
		h.logger.Error("update shadow binding failed",
			zap.Int64("shadow_binding_id", cmd.ID), zap.Error(err))
		return nil, err
	}
	if rowsAffected == 0 {
		return nil, ErrShadowBindingNotFound
	}

	body, _ := json.Marshal(map[string]interface{}{
		"message":           "shadow binding updated",
		"shadow_binding_id": cmd.ID,
		"is_active":         *cmd.IsActive,
	})
	return body, nil
}
```

---

### B. Sửa đổi trong `internal/app/commands/source/debezium_connector.go`
Quay về luồng tạo connector SFTP lập tức, loại bỏ rẽ nhánh bỏ qua:
```go
			Execute: func(ctx context.Context) error {
				// Tạo trực tiếp connector cho tất cả các loại (bao gồm cả SFTP/fssource)
				r, err := h.writer.Create(ctx, cmd.Name, cmd.Config)
				if err != nil {
					return err
				}
				resp = r
				return nil
			},
			Compensate: func(ctx context.Context) error {
				return h.writer.Delete(ctx, cmd.Name)
			},
```

---

### C. Thiết kế rẽ nhánh trong `internal/api/source/source_object_actions_handler.go`
Bổ sung `db *gorm.DB` vào handler struct, sau đó tại hàm `SnapshotV2` rẽ nhánh SFTP:
```go
func (h *SourceObjectActionsHandler) SnapshotV2(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || id <= 0 {
		return c.Status(400).JSON(fiber.Map{"error": "invalid_source_object_id"})
	}

	var body struct {
		TraceID   string `json:"trace_id"`
		Action    string `json:"action"`
		Origin    string `json:"origin"`
		BatchSize int    `json:"batch_size"`
		Overwrite bool   `json:"overwrite"`
	}
	_ = c.BodyParser(&body)

	traceID := strings.TrimSpace(body.TraceID)
	if sc := trace.SpanFromContext(c.UserContext()).SpanContext(); sc.IsValid() {
		traceID = sc.TraceID().String()
	} else if traceID == "" {
		traceID = strings.TrimSpace(c.Get("X-Correlation-Id"))
	}
	if traceID == "" {
		traceID = strings.ReplaceAll(uuid.NewString(), "-", "")
	}

	// 1. Phân tích loại nguồn (engine_type) từ database
	var soRow struct {
		SourceEngineType   string `gorm:"column:source_engine_type"`
		SourceConnectionID int64  `gorm:"column:source_connection_id"`
	}
	err = h.db.Raw("SELECT source_engine_type, source_connection_id FROM cdc_system.source_object_registry WHERE id = ?", id).Scan(&soRow).Error
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to query source object registry"})
	}

	engine := strings.ToLower(soRow.SourceEngineType)
	isSFTP := engine == "sftp" || engine == "file" || engine == "csv"

	bid := parseBindingIDQuery(c)
	scope, err := h.resolveDispatchScope(c, id)
	if err != nil {
		if ferr, ok := err.(*fiber.Error); ok {
			return c.Status(ferr.Code).JSON(fiber.Map{"error": ferr.Message})
		}
		if errors.Is(err, ports.ErrRecordNotFound) {
			return c.Status(404).JSON(fiber.Map{"error": "source_object_not_found"})
		}
		return c.Status(500).JSON(fiber.Map{"error": "resolve_source_object_scope_failed"})
	}

	user := middleware.GetUsername(c)
	ctx := messaging.WithMetadata(c.UserContext(), user, traceID, c.Get("Idempotency-Key"))

	if isSFTP {
		// ĐỐI VỚI NGUỒN SFTP: Tự động tạo topic Kafka sạch để kích hoạt connector đã chạy
		var connRow struct {
			ConnectionCode string `gorm:"column:connection_code"`
		}
		_ = h.db.Raw("SELECT connection_code FROM cdc_system.connection_registry WHERE id = ?", soRow.SourceConnectionID).Scan(&connRow)

		var rawConfig map[string]string
		if connRow.ConnectionCode != "" {
			var rawConfigSanitized string
			_ = h.db.Raw("SELECT raw_config_sanitized::text FROM cdc_system.sources WHERE connector_name = ?", connRow.ConnectionCode).Scan(&rawConfigSanitized)
			if rawConfigSanitized != "" {
				_ = json.Unmarshal([]byte(rawConfigSanitized), &rawConfig)
			}
		}

		if len(rawConfig) > 0 {
			topic := rawConfig["topic"]
			bootstrap := rawConfig["signal.kafka.bootstrap.servers"]
			if topic != "" {
				autoCreateKafkaTopic(ctx, bootstrap, topic, h.logger)
			}
		}

		h.activityLogger.LogAsync(ports.ActivityEntry{
			Operation:   "snapshot.v2",
			TargetTable: strconv.FormatInt(id, 10),
			Status:      "success",
			Details: map[string]any{
				"user":              user,
				"source_object_id":  id,
				"shadow_binding_id": bid,
				"target_table":      scope.TargetTable,
				"shadow_schema":     scope.ShadowSchema,
				"trace_id":          traceID,
				"path":              "sftp_auto_create_topic_bypass_nats",
			},
		})

		return c.Status(202).JSON(fiber.Map{
			"message":           "sftp topic created, sync started",
			"source_object_id":  id,
			"shadow_binding_id": bid,
			"trace_id":          traceID,
			"server_time":       time.Now().Format(time.RFC3339),
		})
	}

	// ĐỐI VỚI DB CDC (Mongo/Postgres): Giữ nguyên luồng dispatch NATS cũ
	cmd := recon.SnapshotV2Command{
		SourceObjectID:  id,
		ShadowBindingID: bid,
		TraceID:         traceID,
		Action:          body.Action,
		Origin:          body.Origin,
		BatchSize:       body.BatchSize,
		Overwrite:       body.Overwrite,
	}

	res, derr := h.bus.Dispatch(ctx, cmd)
	if derr != nil {
		h.activityLogger.LogAsync(ports.ActivityEntry{
			Operation:   "snapshot.v2",
			TargetTable: strconv.FormatInt(id, 10),
			Status:      "error",
			ErrorMsg:    derr.Error(),
		})
		return c.Status(500).JSON(fiber.Map{"error": "dispatch failed: " + derr.Error()})
	}

	h.activityLogger.LogAsync(ports.ActivityEntry{
		Operation:   "snapshot.v2",
		TargetTable: strconv.FormatInt(id, 10),
		Status:      "accepted",
		Details: map[string]any{
			"user":              user,
			"source_object_id":  id,
			"shadow_binding_id": bid,
			"target_table":      scope.TargetTable,
			"shadow_schema":     scope.ShadowSchema,
			"trace_id":          traceID,
			"job_id":            res.JobID,
			"path":              "custom_runner_bypass_debezium_signal",
		},
	})

	return c.Status(202).JSON(fiber.Map{
		"message":           "snapshot.v2 dispatched",
		"source_object_id":  id,
		"shadow_binding_id": bid,
		"trace_id":          traceID,
		"job_id":            res.JobID,
		"server_time":       time.Now().Format(time.RFC3339),
	})
}
```
