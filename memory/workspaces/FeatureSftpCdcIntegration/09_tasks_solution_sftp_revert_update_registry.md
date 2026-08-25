# Kịch bản giải pháp: Revert hoàn toàn thay đổi trong update_registry.go

Tài liệu đặc tả chi tiết phần code cần Muscle revert để trả `update_registry.go` và đăng ký trong `server.go` về nguyên trạng ban đầu.

---

## 1. File `internal/app/commands/source/update_registry.go`

Khôi phục nguyên bản (loại bỏ hoàn toàn logic liên quan đến Kafka Connect / SFTP / Topic / GORM):

```go
package source

import (
	"context"
	"encoding/json"
	"errors"

	"go.uber.org/zap"

	"cdc-cms-service/internal/app/ports"
	model "cdc-cms-service/internal/model/source"
)

// UpdateRegistryCommand patches allow-listed fields on a TableRegistry
// row. Sync — single primary update plus optional cascading mapping
// rule auto-approve when IsActive flips false→true. Activity log + NATS
// reload publishes happen inside the handler so the audit row is atomic
// with the DB writes.
type UpdateRegistryCommand struct {
	ports.SyncCommandMixin
	ID             uint    `json:"id"`
	SyncEngine     *string `json:"sync_engine,omitempty"`
	SyncInterval   *string `json:"sync_interval,omitempty"`
	Priority       *string `json:"priority,omitempty"`
	IsActive       *bool   `json:"is_active,omitempty"`
	Notes          *string `json:"notes,omitempty"`
	TimestampField *string `json:"timestamp_field,omitempty"`
	UpdatedBy      string  `json:"updated_by,omitempty"`
}

func (UpdateRegistryCommand) Type() string { return "registry.update" }

var (
	ErrRegistryNotFound       = errors.New("registry_not_found")
	ErrRegistryNoFields       = errors.New("no fields to update")
	ErrRegistryInvalidTSField = errors.New("invalid_timestamp_field")
)

func (c UpdateRegistryCommand) Validate() error {
	if c.ID == 0 {
		return errors.New("invalid_registry_id")
	}
	if c.SyncEngine == nil && c.SyncInterval == nil && c.Priority == nil &&
		c.IsActive == nil && c.Notes == nil && c.TimestampField == nil {
		return ErrRegistryNoFields
	}
	if c.TimestampField != nil && !validTimestampField(*c.TimestampField) {
		return ErrRegistryInvalidTSField
	}
	return nil
}

type UpdateRegistryHandler struct {
	sourceRepo ports.SourceRepo
	logger     ports.ActivityLogger
	nats       ports.ReloadPublisher
	log        *zap.Logger
}

func NewUpdateRegistryHandler(sourceRepo ports.SourceRepo, logger ports.ActivityLogger, nats ports.ReloadPublisher, log *zap.Logger) *UpdateRegistryHandler {
	return &UpdateRegistryHandler{sourceRepo: sourceRepo, logger: logger, nats: nats, log: log}
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

func validTimestampField(f string) bool {
	s := strings.TrimSpace(f)
	if s == "" {
		return false
	}
	// allow common patterns: updated_at, lastUpdatedAt, createdAt etc.
	// must start with a letter/underscore and contain only alphanumeric/underscore.
	for i, r := range s {
		if i == 0 {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_') {
				return false
			}
		} else {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_') {
				return false
			}
		}
	}
	return true
}
```

---

## 2. File `internal/server/server.go`

Khôi phục đăng ký `"registry.update"` về nguyên bản:

```go
	cmdBus.RegisterSync("registry.update", sourceCmd.NewUpdateRegistryHandler(sourceObjectRepo, activityLogger, reloadPublisher, logger))
```
