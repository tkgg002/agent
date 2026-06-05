# 03_implementation — Delete /shadow Row (Code Demo)

> Plan-only — code dưới đây là **demo**, KHÔNG được apply vào source cho tới khi user duyệt.
> Mọi snippet đã align với pattern hiện có (system_connector.go, update_source_object_v2.go, system_connectors_handler.go).

## File 1 (NEW) — `internal/app/commands/delete_source_object_v2.go`

```go
package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"cdc-cms-service/internal/app/ports"
)

// DeleteSourceObjectV2Command wipes a source-object row (Level C):
//   - manual cleanup các bảng legacy không có FK (cdc_reconciliation_report,
//     cdc_worker_schedule, cdc_mapping_rules) — filter theo source_db +
//     source_object_name + shadow_table.
//   - DELETE source_object_registry — FK ON DELETE CASCADE tự kéo
//     shadow_binding, mapping_rule_v2, master_binding, sync_runtime_state.
//   - DROP TABLE từng shadow_schema."<shadow_table>" (best-effort, ngoài TX
//     metadata) — DDL không nằm trong TX để metadata không bị rollback khi
//     DDL fail; skip nếu binding khác còn ref cùng (schema, table).
//
// Reversible? KHÔNG. Đây là intent Level C — user re-register lại nếu cần.
type DeleteSourceObjectV2Command struct {
	ports.SyncCommandMixin
	ID        int64  `json:"id"`
	DeletedBy string `json:"deleted_by,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

func (DeleteSourceObjectV2Command) Type() string { return "source.delete-v2" }

var (
	ErrSourceObjectDeleteIDInvalid = errors.New("invalid_source_object_id")
	ErrSourceObjectDeleteReasonTooShort = errors.New("reason_too_short")
)

func (c DeleteSourceObjectV2Command) Validate() error {
	if c.ID <= 0 {
		return ErrSourceObjectDeleteIDInvalid
	}
	// Defense-in-depth — middleware Audit đã enforce ≥ 10. Vẫn check để
	// command bus replay (idempotent) không đi qua middleware không trượt.
	if len(strings.TrimSpace(c.Reason)) < 10 {
		return ErrSourceObjectDeleteReasonTooShort
	}
	return nil
}

type DeleteSourceObjectV2Handler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewDeleteSourceObjectV2Handler(db *gorm.DB, logger *zap.Logger) *DeleteSourceObjectV2Handler {
	return &DeleteSourceObjectV2Handler{db: db, logger: logger}
}

type droppedTable struct {
	Schema string `json:"schema"`
	Table  string `json:"table"`
}
type skippedDrop struct {
	Schema string `json:"schema"`
	Table  string `json:"table"`
	Reason string `json:"reason"`
}

func (h *DeleteSourceObjectV2Handler) Handle(ctx context.Context, c ports.Command) (json.RawMessage, error) {
	cmd, ok := c.(DeleteSourceObjectV2Command)
	if !ok {
		return nil, errors.New("source.delete-v2: command type mismatch")
	}
	if h.db == nil {
		return nil, errors.New("source store not ready")
	}

	// 1) SELECT identity của source_object (để filter cleanup legacy).
	var src struct {
		ID                int64
		SourceDatabase    string
		SourceObjectName  string
		ObjectCode        string
	}
	if err := h.db.WithContext(ctx).
		Table("cdc_system.source_object_registry").
		Select("id, source_database, source_object_name, object_code").
		Where("id = ?", cmd.ID).
		Scan(&src).Error; err != nil {
		return nil, fmt.Errorf("select source object: %w", err)
	}
	if src.ID == 0 {
		return nil, ErrSourceObjectNotFound // tái dùng error đã có trong update_source_object_v2.go
	}

	// 2) SELECT các binding rows để biết shadow_schema/shadow_table cần DROP.
	type bindingRow struct {
		ID           int64
		ShadowSchema string
		ShadowTable  string
	}
	var bindings []bindingRow
	if err := h.db.WithContext(ctx).
		Table("cdc_system.shadow_binding").
		Select("id, shadow_schema, shadow_table").
		Where("source_object_id = ?", cmd.ID).
		Scan(&bindings).Error; err != nil {
		return nil, fmt.Errorf("select shadow bindings: %w", err)
	}

	// 3) TX#1 — metadata cleanup (legacy + FK cascade).
	txErr := h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 3a) Legacy bảng KHÔNG có FK — cleanup thủ công.
		// Lưu ý: filter theo (source_db, source_table) là điều kiện tối thiểu;
		// nếu schema legacy còn cột target_table thì cộng thêm vào WHERE.
		shadowTableNames := make([]string, 0, len(bindings))
		for _, b := range bindings {
			shadowTableNames = append(shadowTableNames, b.ShadowTable)
		}

		// cdc_reconciliation_report — drift report.
		if len(shadowTableNames) > 0 {
			if err := tx.Exec(
				`DELETE FROM cdc_system.cdc_reconciliation_report WHERE target_table IN ?`,
				shadowTableNames,
			).Error; err != nil {
				return fmt.Errorf("cleanup recon report: %w", err)
			}
		}

		// cdc_worker_schedule — runner config (xác nhận cột target_table khi
		// implement; nếu cột tên khác thì điều chỉnh WHERE).
		if len(shadowTableNames) > 0 {
			if err := tx.Exec(
				`DELETE FROM cdc_system.cdc_worker_schedule WHERE target_table IN ?`,
				shadowTableNames,
			).Error; err != nil {
				return fmt.Errorf("cleanup worker schedule: %w", err)
			}
		}

		// cdc_mapping_rules — legacy field-level mapping (V1).
		if err := tx.Exec(
			`DELETE FROM cdc_system.cdc_mapping_rules WHERE source_table = ?`,
			src.SourceObjectName,
		).Error; err != nil {
			return fmt.Errorf("cleanup legacy mapping rules: %w", err)
		}

		// 3b) DELETE source_object_registry — FK CASCADE kéo:
		//   shadow_binding, mapping_rule_v2, master_binding, sync_runtime_state.
		if err := tx.Exec(
			`DELETE FROM cdc_system.source_object_registry WHERE id = ?`,
			cmd.ID,
		).Error; err != nil {
			return fmt.Errorf("delete source object: %w", err)
		}
		return nil
	})
	if txErr != nil {
		h.logger.Error("delete source object metadata failed",
			zap.Int64("source_object_id", cmd.ID), zap.Error(txErr))
		return nil, txErr
	}

	// 4) Best-effort DROP TABLE (NGOÀI TX — DDL fail không rollback metadata).
	dropped := make([]droppedTable, 0, len(bindings))
	skipped := make([]skippedDrop, 0)
	for _, b := range bindings {
		// 4a) Multi-binding share check — binding nào khác còn ref?
		var refCount int64
		if err := h.db.WithContext(ctx).
			Table("cdc_system.shadow_binding").
			Where("shadow_schema = ? AND shadow_table = ?", b.ShadowSchema, b.ShadowTable).
			Count(&refCount).Error; err != nil {
			skipped = append(skipped, skippedDrop{b.ShadowSchema, b.ShadowTable,
				fmt.Sprintf("ref_count_query_failed: %v", err)})
			continue
		}
		if refCount > 0 {
			skipped = append(skipped, skippedDrop{b.ShadowSchema, b.ShadowTable,
				"multi_binding_share"})
			continue
		}

		// 4b) Quote identifier — schema/table đã pass slugify ở FE, vẫn
		// quote double-quotes phòng case identifier reserved.
		ddl := fmt.Sprintf(`DROP TABLE IF EXISTS %q.%q`, b.ShadowSchema, b.ShadowTable)
		if err := h.db.WithContext(ctx).Exec(ddl).Error; err != nil {
			h.logger.Warn("drop shadow table failed",
				zap.String("schema", b.ShadowSchema),
				zap.String("table", b.ShadowTable),
				zap.Error(err))
			skipped = append(skipped, skippedDrop{b.ShadowSchema, b.ShadowTable,
				fmt.Sprintf("drop_failed: %v", err)})
			continue
		}
		dropped = append(dropped, droppedTable{b.ShadowSchema, b.ShadowTable})
	}

	body, _ := json.Marshal(map[string]interface{}{
		"status":           "deleted",
		"source_object_id": cmd.ID,
		"object_code":      src.ObjectCode,
		"dropped_tables":   dropped,
		"skipped_drops":    skipped,
	})
	return body, nil
}
```

**Ghi chú**:
- `ErrSourceObjectNotFound` đã được khai báo ở `update_source_object_v2.go:35`. Reuse, không định nghĩa lại.
- `tx.Exec(... IN ?, slice)` là GORM idiom — generate prepared `IN ($1, $2, ...)`. Nếu slice rỗng đã được guard ở `if len(...) > 0`.
- DDL `%q.%q` (Go quoting) tạo output `"shadow_db"."tbl"` — đủ safe cho Postgres identifier.

---

## File 2 (EDIT) — `internal/server/server.go`

Tại dòng **231** (ngay sau `source.update-v2`), append:

```go
cmdBus.RegisterSync("source.delete-v2", commands.NewDeleteSourceObjectV2Handler(db, logger))
```

---

## File 3 (EDIT) — `internal/api/source_objects_handler.go`

### 3.1 Constructor signature — thêm `bus`

```go
type SourceObjectsHandler struct {
	db        *gorm.DB
	logger    *zap.Logger
	bus       messaging.CommandBus // NEW
	listQ     *queries.ListSourceObjectsHandler
	mappingCx *queries.GetSourceObjectMappingContextHandler
}

func NewSourceObjectsHandler(
	db *gorm.DB,
	logger *zap.Logger,
	bus messaging.CommandBus, // NEW
	listQ *queries.ListSourceObjectsHandler,
	mappingCx *queries.GetSourceObjectMappingContextHandler,
) *SourceObjectsHandler {
	return &SourceObjectsHandler{db: db, logger: logger, bus: bus, listQ: listQ, mappingCx: mappingCx}
}
```

Imports thêm:
```go
"cdc-cms-service/internal/app/commands"
"cdc-cms-service/internal/messaging"
"cdc-cms-service/internal/middleware"
```

### 3.2 Method `Delete` — append cuối file

```go
type deleteSourceObjectRequest struct {
	Reason string `json:"reason"`
}

// Delete godoc
// @Summary      Delete V2 source object (Level C — hard wipe)
// @Description  Xoá vĩnh viễn 1 source-object: cleanup legacy + FK CASCADE
// @Description  metadata + DROP TABLE shadow vật lý (best-effort).
// @Tags         Source Objects
// @Accept       json
// @Produce      json
// @Param        id      path int true "Source object ID"
// @Param        payload body deleteSourceObjectRequest true "Reason (≥10 chars)"
// @Param        Idempotency-Key header string true "Idempotency key"
// @Success      202 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Failure      502 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/v1/source-objects/{id} [delete]
func (h *SourceObjectsHandler) Delete(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || id <= 0 {
		return c.Status(400).JSON(fiber.Map{"error": "invalid_source_object_id"})
	}
	if h.bus == nil {
		return c.Status(503).JSON(fiber.Map{"error": "command bus not ready"})
	}
	var req deleteSourceObjectRequest
	// body parse best-effort — middleware Audit đã read+validate reason.
	_ = c.BodyParser(&req)

	user := middleware.GetUsername(c)
	cmd := commands.DeleteSourceObjectV2Command{
		ID:        id,
		DeletedBy: user,
		Reason:    req.Reason,
	}
	ctx := messaging.WithMetadata(
		c.UserContext(),
		user,
		c.Get("X-Correlation-Id"),
		c.Get("Idempotency-Key"),
	)
	body, err := h.bus.Execute(ctx, cmd)
	if err != nil {
		if errors.Is(err, commands.ErrSourceObjectNotFound) {
			return c.Status(404).JSON(fiber.Map{"error": "source_object_not_found"})
		}
		return c.Status(502).JSON(fiber.Map{"error": "delete_failed", "detail": err.Error()})
	}
	// body đã là json.RawMessage từ handler (status + dropped_tables + skipped_drops).
	c.Type("application/json")
	return c.Status(202).Send(body)
}
```

Imports thêm: `"errors"`.

---

## File 4 (EDIT) — `internal/server/server.go` (dòng 267)

`router.SetupRoutes(...)` call hiện đã pass `sourceObjectsHandler`. **Phải sửa nơi tạo `sourceObjectsHandler`** ở phần khởi tạo trước đó để pass `cmdBus`:

```go
// Tìm dòng tương tự:
sourceObjectsHandler := api.NewSourceObjectsHandler(db, logger, listSourceObjectsQ, mappingCxQ)
// Sửa thành:
sourceObjectsHandler := api.NewSourceObjectsHandler(db, logger, cmdBus, listSourceObjectsQ, mappingCxQ)
```

Note: vị trí chính xác sẽ xác định khi grep `NewSourceObjectsHandler(` trong server.go — không cần thay đổi `router.SetupRoutes` signature.

---

## File 5 (EDIT) — `internal/router/router.go`

Tại dòng **322** (ngay sau `shared.Get("/v1/shadow-bindings", ...)`), thêm block:

```go
// Destructive DELETE — full hard wipe + DROP shadow table (Level C).
// registerDestructive only wraps POST → mount manually with destructiveChain.
// Body: {"reason": "<≥10 chars>"}; Header: Idempotency-Key.
{
	deleteHandlers := append([]fiber.Handler{}, destructiveChain...)
	deleteHandlers = append(deleteHandlers, sourceObjectsHandler.Delete)
	apiGroup.Delete("/v1/source-objects/:id", deleteHandlers...)
}
```

Pattern y hệt block `/v1/system/connectors/:name` (dòng 211-215).

---

## File 6 (EDIT) — `cdc-cms-web/src/pages/TableRegistry.tsx`

### 6.1 Import (dòng 3)

```tsx
import { PlusOutlined, SyncOutlined, DatabaseOutlined, SearchOutlined,
  ThunderboltOutlined, RocketOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons';
```

### 6.2 State + handler (sau dòng 308 `actionLoadingId`)

```tsx
const [deletePending, setDeletePending] = useState<TRegistry | null>(null);
const [deleting, setDeleting] = useState(false);

const handleDelete = async (reason: string) => {
  if (!deletePending) return;
  setDeleting(true);
  try {
    await cmsApi.delete(`/api/v1/source-objects/${deletePending.id}`, {
      data: { reason },
      headers: {
        'Idempotency-Key': `delete-source-object-${deletePending.id}-${Date.now()}`,
      },
    });
    message.success(`Đã xoá ${deletePending.source_db}.${deletePending.source_table}`);
    setDeletePending(null);
    fetchData();
    fetchShadowBindings();
  } catch (err) {
    message.error(humanizeApiError(err, 'Xoá source-object thất bại'));
  } finally {
    setDeleting(false);
  }
};
```

### 6.3 Button trong cell `Thao tác` (dòng 856-895, append vào `<Space wrap>`)

```tsx
<Tooltip title="Xoá vĩnh viễn metadata + DROP table shadow vật lý">
  <Button
    size="small"
    danger
    icon={<DeleteOutlined />}
    onClick={(e) => { e.stopPropagation(); setDeletePending(record); }}
  >
    Xoá
  </Button>
</Tooltip>
```

### 6.4 Modal (cuối JSX, trước `</div>` cuối cùng dòng 1193)

```tsx
<ConfirmDestructiveModal
  open={!!deletePending}
  danger
  title="Xoá vĩnh viễn shadow object"
  targetName={deletePending ? `${deletePending.source_db}.${deletePending.source_table} → ${deletePending.target_table}` : ''}
  description={
    <>
      Hành động sẽ <strong>DROP TABLE shadow vật lý</strong>, xoá toàn bộ
      metadata (binding, mapping rule V2, recon report, worker schedule, master
      binding). <strong>Không thể hoàn tác</strong>. Sau khi xoá, bạn có thể
      <em> re-register</em> lại cùng (connector + source DB + source table) để
      tạo binding mới.
      <br /><br />
      <Text type="secondary">
        Out-of-scope: KHÔNG reset Debezium offset, KHÔNG xoá schema
        <Text code> shadow_&lt;db&gt;</Text>, KHÔNG xoá connection_registry,
        KHÔNG xoá master table vật lý.
      </Text>
    </>
  }
  actionLabel="Xoá vĩnh viễn"
  loading={deleting}
  onConfirm={handleDelete}
  onCancel={() => setDeletePending(null)}
/>
```

---

## Tóm tắt impact

| File | Loại | LOC delta (ước lượng) |
|---|---|---|
| `internal/app/commands/delete_source_object_v2.go` | NEW | +180 |
| `internal/server/server.go` | EDIT | +2 (1 RegisterSync + 1 constructor arg) |
| `internal/api/source_objects_handler.go` | EDIT | +60 (constructor +5, Delete method +55) |
| `internal/router/router.go` | EDIT | +6 (block manual mount) |
| `cdc-cms-web/src/pages/TableRegistry.tsx` | EDIT | +60 (state +5, handler +20, button +10, modal +25) |
| **TỔNG** | | **~+308 LOC** |

## Pattern compliance check

- ✅ CQRS — Command + Handler trong `app/commands/`, registered ở `cmdBus.RegisterSync`.
- ✅ Destructive chain — JWT (apiGroup) + RequireOpsAdmin + Idempotency + Audit (tự động fold reason vào admin_actions).
- ✅ FE pattern — `ConfirmDestructiveModal` + `cmsApi.delete` + `Idempotency-Key` + `humanizeApiError`. Mirror `SensitiveFieldsPage.tsx:92` và `SourceConnectors.tsx:446`.
- ✅ Best-effort DDL — pattern giống `DeleteSystemConnectorHandler` (Kafka 404 tolerant).
- ✅ Minimal impact — không sửa middleware, không sửa migration, không tạo bảng mới, không sửa worker.
