# Kịch bản giải pháp: Thực hiện rẽ nhánh tạo Topic SFTP tại cdc-worker thay vì API

Tài liệu đặc tả chi tiết mã nguồn cần chỉnh sửa trong dự án `cdc-cms-service` (API) và `centralized-data-service` (Worker).

---

## I. Dự án `cdc-cms-service` (Revert các thay đổi cũ về API)

### 1. File `internal/api/source/source_object_actions_handler.go`
Revert hoàn toàn về trạng thái gốc ban đầu (xóa bỏ logic `isSFTP` check, helper `autoCreateKafkaTopic`, import `kafka-go`, `os` và tham số `db` trong struct/constructor):

```go
package source

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"cdc-cms-service/internal/api/system"
	"cdc-cms-service/internal/app/commands/recon"
	"cdc-cms-service/internal/app/commands/source"
	"cdc-cms-service/internal/app/ports"
	"cdc-cms-service/internal/app/queries/shadow"
	"cdc-cms-service/internal/infra/messaging"
	"cdc-cms-service/internal/infra/persistence"
	"cdc-cms-service/internal/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type SourceObjectActionsHandler struct {
	bridgeReader     shadow.BridgeStatusReader
	bus              ports.CommandBus
	publisher        ports.Publisher
	activityLogger   ports.ActivityLogger
	logger           *zap.Logger
	transformJobRepo *persistence.TransformJobRepo
}

func NewSourceObjectActionsHandler(
	bridgeReader shadow.BridgeStatusReader,
	bus ports.CommandBus,
	publisher ports.Publisher,
	activityLogger ports.ActivityLogger,
	logger *zap.Logger,
) *SourceObjectActionsHandler {
	return &SourceObjectActionsHandler{
		bridgeReader:   bridgeReader,
		bus:            bus,
		publisher:      publisher,
		activityLogger: activityLogger,
		logger:         logger,
	}
}
```

Và trả hàm `SnapshotV2` về nguyên bản gốc (luôn dispatch command sang NATS):

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

	bid := parseBindingIDQuery(c)
	scope, err := h.resolveDispatchScope(c, id)
	if err != nil {
		if ferr, ok := err.(*fiber.Error); ok {
			return c.Status(ferr.Code).JSON(fiber.Map{"error": ferr.Message})
		}
		if errors.Is(err, ports.ErrRecordNotFound) {
			return c.Status(404).JSON(fiber.Map{"error": "source_object_not_found"})
		}
		h.logger.Error("resolve source object dispatch scope failed", zap.Int64("source_object_id", id), zap.Int64("shadow_binding_id", bid), zap.Error(err))
		return c.Status(500).JSON(fiber.Map{"error": "resolve_source_object_scope_failed"})
	}
	if bid > 0 && scope.SourceObjectID != id {
		return c.Status(400).JSON(fiber.Map{"error": "binding_id_mismatch"})
	}

	user := middleware.GetUsername(c)
	ctx := messaging.WithMetadata(c.UserContext(), user, traceID, c.Get("Idempotency-Key"))
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

### 2. File `internal/server/server.go`
Revert về đăng ký gốc ban đầu:
```go
	h.Source.ObjectActions = apisource.NewSourceObjectActionsHandler(bridgeStatusReader, cmdBus, natsPublisher, activityLogger, logger)
```

### 3. File `test/internal/api/source_object_actions_handler_test.go`
Revert về mock constructor gốc ban đầu:
```go
	handler := apisource.NewSourceObjectActionsHandler(reader, nil, pub, activityLog, nil)
```

---

## II. Dự án `centralized-data-service` (Triển khai rẽ nhánh tại Worker)

### 1. File `internal/handler/orchestration/snapshot_runner_handler.go`

Thực hiện import thêm `os` và `"github.com/segmentio/kafka-go"` vào đầu file.

```diff
 import (
 	reposhadow "centralized-data-service/internal/repository/shadow"
 	reposource "centralized-data-service/internal/repository/source"
 	handlershadow "centralized-data-service/internal/handler/shadow"
 	"centralized-data-service/internal/model/shadow"
 	"centralized-data-service/internal/model/recon"
 	"context"
 	"encoding/json"
 	"errors"
 	"fmt"
+	"os"
 	"strconv"
 	"strings"
 	"sync/atomic"
 	"time"
 
+	"github.com/segmentio/kafka-go"
 	"github.com/jackc/pgx/v5"
```

Tại đầu hàm `runSnapshot`, thực hiện rẽ nhánh nếu engine là SFTP:

```go
func (r *SnapshotRunner) runSnapshot(ctx context.Context, p snapshotV2Payload, jobID string) (retErr error) {
	startedAt := time.Now()
	var (
		targetTable    string
		connectionCode string
		rowsTotal      int64
	)
	defer func() {
		status := "success"
		errMsg := ""
		if retErr != nil {
			status = "error"
			errMsg = retErr.Error()
		}
		r.writeActivity(p, jobID, targetTable, connectionCode, status, rowsTotal, startedAt, errMsg)
	}()

	// 1. Resolve source object + connection.
	so, err := r.soRepo.GetByID(ctx, p.SourceObjectID)
	if err != nil {
		return fmt.Errorf("source_object_registry lookup id=%d: %w", p.SourceObjectID, err)
	}
	if so == nil {
		return fmt.Errorf("source_object_id=%d not found", p.SourceObjectID)
	}

	engine := strings.ToLower(so.SourceEngineType)
	isSFTP := engine == "sftp" || engine == "file" || engine == "csv"

	if isSFTP {
		// A. Resolve connection code
		var connRow struct {
			ConnectionCode string `gorm:"column:connection_code"`
		}
		err = r.db.Raw("SELECT connection_code FROM cdc_system.connection_registry WHERE id = ?", so.SourceConnectionID).Scan(&connRow).Error
		if err != nil {
			return fmt.Errorf("failed to query connection registry: %w", err)
		}

		if connRow.ConnectionCode == "" {
			r.logger.Warn("sftp connection code not found for source object", zap.Int64("id", so.ID))
			return fmt.Errorf("connection code not found for source object")
		}
		connectionCode = connRow.ConnectionCode
		targetTable = so.ObjectCode

		// B. Resolve connector config from cdc_system.sources
		var rawConfig map[string]string
		var rawConfigSanitized string
		err = r.db.Raw("SELECT raw_config_sanitized::text FROM cdc_system.sources WHERE connector_name = ?", connRow.ConnectionCode).Scan(&rawConfigSanitized).Error
		if err != nil {
			return fmt.Errorf("failed to query sources configuration: %w", err)
		}

		if rawConfigSanitized != "" {
			_ = json.Unmarshal([]byte(rawConfigSanitized), &rawConfig)
		}

		if len(rawConfig) > 0 {
			topic := rawConfig["topic"]
			bootstrap := rawConfig["signal.kafka.bootstrap.servers"]
			if topic != "" {
				autoCreateKafkaTopic(ctx, bootstrap, topic, r.logger)
			}
		}

		// Cập nhật trạng thái snapshot_progress là done luôn
		var progressID int64
		if p.ShadowBindingID > 0 {
			// Query progress ID
			_ = r.db.Raw("SELECT id FROM cdc_system.snapshot_progress WHERE shadow_binding_id = ? AND status != 'completed' ORDER BY id DESC LIMIT 1", p.ShadowBindingID).Scan(&progressID)
		}
		if progressID > 0 {
			_ = r.db.Exec("UPDATE cdc_system.snapshot_progress SET status = 'completed', rows_processed = 0, completed_at = NOW() WHERE id = ?", progressID).Error
		}

		return nil
	}

	// ĐỐI VỚI DB CDC (Mongo/Postgres): Tiếp tục luồng xử lý cũ
	isMongo := strings.EqualFold(so.SourceEngineType, "mongodb")
	isPG := strings.EqualFold(so.SourceEngineType, "postgresql")
	if !isMongo && !isPG {
		return fmt.Errorf("snapshot.v2 currently supports engine=mongodb or postgresql (got %q)", so.SourceEngineType)
	}
```

Và định nghĩa helper `autoCreateKafkaTopic` ở cuối file `snapshot_runner_handler.go`:

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
