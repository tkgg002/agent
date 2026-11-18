---
tier: T2
suffix: sync_fields_alter_type
workspace: fix-shadow-sync-fields-alter-type-2026-11-16
---
# 09 Tasks Solution — Khắc Phục Triệt Để Lỗi "Sync Fields" Không Chuyển Đổi Kiểu Cột `TEXT` → `JSONB` Trên Bảng Shadow

## 0. Bảng Tổng Hợp Đối Chiếu Phát Hiện & Giải Pháp Core Thuần Khiết (`F1..F4` — Tuân thủ `L097`: Không Vá/Khôi Phục Data Cũ)

> **Nguyên tắc thiết kế (`L097` — Chỉ thị của User):** Không đụng vào dữ liệu cũ, không viết logic khôi phục từ `_raw_data` hay regex lọc chuỗi rác `map[...]` trong câu lệnh DDL (các bảng thử nghiệm cũ bị bẩn có thể xóa làm lại). Giải pháp **chỉ tập trung chuẩn hóa sạch tận gốc luồng Core từ hiện tại trở về sau**.

| Mã | Hạng mục Core | Bằng chứng Vật lý (`file:line` & Runtime Log) | Nguyên nhân Gốc rễ (`VERIFIED`) | Giải pháp Core Thuần Khiết (Ánh xạ 1-1) |
| :-: | :--- | :--- | :--- | :--- |
| **`F1`** | **Core Ingest/Coerce ghi hỏng Object/Array thành chuỗi Go `map[...]` trên cột `TEXT`/`VARCHAR`** | [`dynamic_mapper.go:233-235`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/shadow/dynamic_mapper.go#L233-L235)<br/>[`schema_adapter_coerce.go:31-38`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/shadow/schema_adapter_coerce.go#L31-L38)<br/>Log `gpay-postgres-shadow`: `Token "map" is invalid` | Khi field nguồn là JSON Object/Array (`map[string]interface{}` / `[]interface{}`) nhưng cột Shadow đang là `TEXT`/`VARCHAR`, cả `convertType` và `CoerceValue` đều gọi `fmt.Sprintf("%v", val)` tạo ra chuỗi Go `"map[k:v]"` thay vì JSON hợp lệ — làm hỏng dữ liệu ngay từ khâu ghi và khiến mọi lệnh `ALTER COLUMN TYPE JSONB` về sau gãy `SQLSTATE 22P02`. | Trong cả `convertType` và `CoerceValue`, nếu `val` là `map[string]interface{}` hoặc `[]interface{}`, dùng `json.Marshal(val)` để lưu chuỗi JSON chuẩn ngay từ đầu; giữ nguyên 100% hành vi với kiểu nguyên thủy (`string`, `int`, `float`, `bool`, `nil`). |
| **`F2`** | **Core DDL `SchemaAdapter.AlterColumn` thiếu chuẩn hóa quote, whitelist, `lock_timeout` và rỗng `""`** | [`schema_adapter.go:822-840`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/shadow/schema_adapter.go#L822-L840) | `AlterColumn` nối chuỗi `"%s"` thô, chưa kiểm tra `IsTypeWhitelisted`, không đặt `lock_timeout` (dễ treo `AccessExclusiveLock` chặn `BatchBuffer`), và ép thẳng `"col"::JSONB` (kể cả khi gặp chuỗi rỗng `""`). | Chuẩn hóa trực tiếp trong `SchemaAdapter.AlterColumn` (giữ nguyên chữ ký, **0 logic vá data cũ**): dùng `sqlutil.QualifiedTable` + `sqlutil.QuoteIdent`, kiểm tra `IsTypeWhitelisted`, bọc `SET LOCAL lock_timeout = '10s'`, dùng `NULLIF(btrim(col::text), '')::JSONB` cho `JSONB`/`JSON` (chuẩn fail-fast nếu dữ liệu không phải JSON). |
| **`F3`** | **Core Worker `HandleCreateDefaultColumns` nuốt lỗi DDL, thiếu đếm `columnsAlteredType` & quên xóa cache Schema** | [`schema_ddl_handler.go:308-348`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/schema_ddl_handler.go#L308-L348)<br/>[`schema_ddl_handler.go:404,443-449`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/schema_ddl_handler.go#L443-L449) | (1) Khi `AlterColumn` lỗi, handler chỉ log rồi vẫn trả `Status: "success"` và bỏ sót `columnsAlteredType` khỏi `RowsAffected`.<br/>(2) Nếu bảng không có `TimestampField`, `InvalidateCacheInSchema` và `schema.config.reload` không được gọi → `BatchBuffer` giữ kiểu `text` cũ trong RAM. | (1) Gom `ddlErrors`, cộng đủ `columnsAlteredType` vào `RowsAffected`, trả trung thực `Status: "error"` kèm `Error` khi có cột lỗi.<br/>(2) Khi `columnsAdded > 0 \|\| columnsAlteredType > 0` (và ở `HandleAlterColumn`), gọi ngay `InvalidateCacheInSchema` + phát `schema.config.reload` để xóa cache toàn cụm. |
| **`F4`** | **Core CMS & UI `Sync Fields` bắn bất đồng bộ không chờ kết quả và không refresh `Data Type Current`** | [`MappingFieldsPage.tsx:165-185`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/MappingFieldsPage.tsx#L165-L185)<br/>[`source_object_actions_handler.go:197-274`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/source/source_object_actions_handler.go#L197-L274) | `handleSyncFields` nhận `202 Accepted` trước khi Worker chạy DDL, hiện ngay toast xanh giả, không nhận được lỗi khi DDL thất bại và không gọi lại `fetchShadowColumns()`. | Bổ sung CQRS `SyncShadowColumnsCommand` (`app/commands/source/sync_shadow_columns.go`) chờ `msg.Reply` qua `ports.RPCClient` khi có `?wait=true` (giữ nguyên `202` mặc định cho caller cũ). Trên `MappingFieldsPage.tsx`, gọi `?wait=true`, hiện lỗi thật nếu fail và gọi `Promise.all([fetchRules(), fetchShadowColumns()])` ngay khi xong. |

---

## 1. [S2.5 Pre-Plan Critique — DG0] Kiểm Tra Thực Tế & Loại Bỏ Các Hướng Vá Tạm

1. **Loại bỏ hoàn toàn tư duy vá dữ liệu cũ (`L097`):** Không nhét `_raw_data #> path` hay regex `map[...]` vào `AlterColumn`. Khi `F1` (`json.Marshal` cho `map`/`slice` trên cột `TEXT`) đã được khóa ở đầu vào Core, mọi dòng dữ liệu ghi mới vào cột `TEXT` đều là chuỗi JSON hợp lệ (`{"...":...}` / `[...]`), nên khi đổi `TEXT → JSONB` bằng `ALTER COLUMN` chuẩn sẽ thành công tự nhiên 100%; còn nếu người vận hành đổi một cột `TEXT` chứa chữ thuần (`"pending"`) sang `JSONB`, PostgreSQL phải báo lỗi fail-fast minh bạch qua `F3` + `F4` thay vì tự ý ép mù.
2. **Tương thích môi trường vật lý (`gpay-postgres-shadow` = `postgres:15-alpine`):** Chỉ sử dụng cú pháp SQL chuẩn ANSI / PostgreSQL 15 (`NULLIF`, `btrim`, `SET LOCAL lock_timeout = '10s'`), không dùng hàm độc quyền của PG16+.

---

## 2. Quy Trình 5 Bước Kiểm Tra Thiết Kế (`Blast-Radius = 0` — Mục 5 `system_overview_and_design_pattern.md`)

1. **Nhóm Chức Năng & Phân Tầng:**
   - `centralized-data-service`: Nhóm `shadow` — `internal/service/shadow/dynamic_mapper.go`, `schema_adapter_coerce.go`, `schema_adapter.go` và `internal/handler/shadow/schema_ddl_handler.go`.
   - `cdc-cms-service`: Nhóm `source` — `internal/api/source/source_object_actions_handler.go` $\rightarrow$ `internal/app/commands/source/sync_shadow_columns.go` $\rightarrow$ `ports.RPCClient`, đăng ký tại `internal/server/server.go`.
   - `cdc-cms-web`: `src/pages/MappingFieldsPage.tsx`.
2. **Canonical Pattern Reference (Luồng mẫu đang chạy tốt):**
   - Khóa `SET LOCAL lock_timeout`: theo khuôn [`master_ddl_generator.go:609-622`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/master_ddl_generator.go#L609-L622).
   - NATS RPC `Request` qua `ports.RPCClient` + `msg.Reply` của `BaseHandler`: theo khuôn [`introspection_handler.go:407-428`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/system/introspection_handler.go#L407-L428) và [`base_handler.go:67-70`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/base/base_handler.go#L67-L70).
3. **Đối chiếu 5 Shared Chokepoints & Đa hình 3 Chiều:**
   - Giữ nguyên 100% chữ ký `SchemaAdapter.AlterColumn`, `NewSourceObjectActionsHandler`, và `CreateDefaultColumnsCommand`.
   - Các caller cũ không truyền `?wait=true` (`TableRegistry.tsx:1057`, `registry_handler_bulk.go:65`, `registry_handler_register.go:70`) giữ nguyên 100% luồng `202 Accepted` qua `h.bus.Dispatch`.
   - Ở `CoerceValue` & `convertType`, kiểu nguyên thủy (`string`, `int`, `float`, `bool`, `nil`) giữ nguyên 100% kết quả cũ; chỉ `map[string]interface{}` và `[]interface{}` được marshal thành chuỗi JSON chuẩn.
4. **Kiểm tra Hợp đồng Liên Service (Cross-Service Contract):**
   - `BaseHandler.PublishResult` (`base_handler.go:95-105`) khi nhận `msg.Reply != ""` tự phản hồi JSON `CommandResult{command, target_table, rows_affected, status, error}` về đúng `replySubject` của `natsPublisher.Request`.
5. **Phạm vi Build & Test dự kiến khi Thực thi (S4):**
   - `centralized-data-service`: `go build ./internal/... ./cmd/...`, `go vet ./internal/... ./cmd/...`, `go test ./internal/service/shadow/... ./internal/handler/shadow/... ./test/internal/service/...`.
   - `cdc-cms-service`: `go build ./internal/... ./cmd/...`, `go vet ./internal/... ./cmd/...`, `go test ./internal/app/commands/source/... ./internal/api/source/... ./test/internal/app/commands/...`.
   - `cdc-cms-web`: `npm run build` (TypeScript `tsc -b` + Vite build).

---

## 3. Code Demo Chi Tiết (Phương Án Core Thuần Khiết)

### 3.1. Worker — Chuẩn Hóa Tuần Tự Hóa JSON Ở Đầu Vào Cột `TEXT`/`VARCHAR` (`F1`)

Tại [`dynamic_mapper.go:233-235`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/shadow/dynamic_mapper.go#L233-L235) và [`schema_adapter_coerce.go:31-38`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/shadow/schema_adapter_coerce.go#L31-L38):
```go
// dynamic_mapper.go:233-235
	default: // TEXT, VARCHAR, etc.
		switch val.(type) {
		case map[string]interface{}, []interface{}:
			if b, err := json.Marshal(val); err == nil {
				return string(b), nil
			}
		}
		return fmt.Sprintf("%v", val), nil

// schema_adapter_coerce.go:31-38
	if colType == "character varying" || colType == "text" || colType == "varchar" {
		switch val.(type) {
		case string:
			// pass through
		case map[string]interface{}, []interface{}:
			if b, err := json.Marshal(val); err == nil {
				return string(b)
			}
			return fmt.Sprintf("%v", val)
		default:
			return fmt.Sprintf("%v", val)
		}
	}
```

### 3.2. Worker — Chuẩn Hóa `SchemaAdapter.AlterColumn` Tinh Gọn, An Toàn Khóa & Fail-Fast (`F2`)

Tại [`schema_adapter.go:822-840`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/shadow/schema_adapter.go#L822-L840):
```go
func (sa *SchemaAdapter) AlterColumn(ctx context.Context, schemaName, tableName, columnName, dataType, action string) error {
	if strings.TrimSpace(schemaName) == "" {
		schemaName = "public"
	}
	if !sqlutil.IsSafeIdent(schemaName) || !sqlutil.IsSafeIdent(tableName) || !sqlutil.IsSafeIdent(columnName) {
		return fmt.Errorf("invalid identifier: %s.%s.%s", schemaName, tableName, columnName)
	}
	act := strings.ToLower(strings.TrimSpace(action))
	cleanType := strings.TrimSpace(dataType)
	if (act == "add" || act == "alter_type") && !IsTypeWhitelisted(cleanType) {
		return fmt.Errorf("disallowed data_type: %q", cleanType)
	}

	qualifiedTable := sqlutil.QualifiedTable(schemaName, tableName)
	colIdent := sqlutil.QuoteIdent(columnName)

	var ddl string
	switch act {
	case "add":
		ddl = fmt.Sprintf(`ALTER TABLE %s ADD COLUMN IF NOT EXISTS %s %s`, qualifiedTable, colIdent, cleanType)
	case "drop":
		ddl = fmt.Sprintf(`ALTER TABLE %s DROP COLUMN IF EXISTS %s`, qualifiedTable, colIdent)
	case "alter_type":
		upperType := strings.ToUpper(cleanType)
		usingExpr := fmt.Sprintf(`%s::%s`, colIdent, cleanType)
		if upperType == "JSONB" || upperType == "JSON" {
			usingExpr = fmt.Sprintf(`NULLIF(btrim(%s::text), '')::%s`, colIdent, cleanType)
		}
		ddl = fmt.Sprintf(`ALTER TABLE %s ALTER COLUMN %s TYPE %s USING %s`,
			qualifiedTable, colIdent, cleanType, usingExpr)
	default:
		return fmt.Errorf("invalid alter action: %s", action)
	}

	return sa.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`SET LOCAL lock_timeout = '10s'`).Error; err != nil {
			return err
		}
		return tx.Exec(ddl).Error
	})
}
```

### 3.3. Worker — Trả Đúng Lỗi DDL & Xóa Cache Schema Toàn Cụm Trong `HandleCreateDefaultColumns` (`F3`)

Tại [`schema_ddl_handler.go:291-348,443-450`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/schema_ddl_handler.go#L291-L348):
```go
	var ddlErrors []string
	columnsAlteredType := 0
	// ... trong vòng lặp for _, rule := range rules:
	if existingType, ok := existingColsWithType[colLower]; ok {
		if normalizePGType(rule.DataType) == normalizePGType(existingType) {
			columnsAlreadyExist++
			continue
		}
		if err := h.schemaAdapter.AlterColumn(ctx, schemaName, payload.TargetTable, rule.TargetColumn, rule.DataType, "alter_type"); err != nil {
			observability.Ctx(ctx, h.Logger).Error("failed to ALTER COLUMN TYPE (drift fix)",
				zap.String("column", rule.TargetColumn),
				zap.String("from_type", existingType),
				zap.String("to_type", rule.DataType),
				zap.Error(err),
			)
			ddlErrors = append(ddlErrors, fmt.Sprintf("%s (%s -> %s): %v", rule.TargetColumn, existingType, rule.DataType, err))
			columnsSkipped++
			continue
		}
		columnsAlteredType++
		existingColsWithType[colLower] = strings.ToLower(rule.DataType)
		continue
	}
	if err := h.schemaAdapter.AlterColumn(ctx, schemaName, payload.TargetTable, rule.TargetColumn, rule.DataType, "add"); err != nil {
		ddlErrors = append(ddlErrors, fmt.Sprintf("%s (add %s): %v", rule.TargetColumn, rule.DataType, err))
		columnsSkipped++
		continue
	}
	// ... ngay sau vòng lặp rules:
	if columnsAdded > 0 || columnsAlteredType > 0 {
		h.schemaAdapter.InvalidateCacheInSchema(schemaName, payload.TargetTable)
		if h.NatsConn != nil {
			_ = h.NatsConn.Publish("schema.config.reload", []byte(payload.TargetTable))
		}
	}
	// ... cuối hàm HandleCreateDefaultColumns:
	finalStatus := "success"
	finalErr := ""
	if len(ddlErrors) > 0 {
		finalStatus = "error"
		finalErr = strings.Join(ddlErrors, "; ")
	}
	h.PublishResult(ctx, msg, base.CommandResult{
		Command:      "create-default-columns",
		RegistryID:   payload.RegistryID,
		TargetTable:  payload.TargetTable,
		RowsAffected: columnsAdded + columnsAlteredType + count,
		Status:       finalStatus,
		Error:        finalErr,
	})
```

### 3.4. CMS Backend & Frontend — Chế Độ Đồng Bộ `?wait=true` & Tự Động Refresh `Data Type Current` (`F4`)

1. **`cdc-cms-service/internal/app/commands/source/sync_shadow_columns.go` (file mới — CQRS SyncCommand):**
```go
package source

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"cdc-cms-service/internal/app/ports"
)

type SyncShadowColumnsCommand struct {
	ports.SyncCommandMixin
	CreateDefaultColumnsCommand
}

func (SyncShadowColumnsCommand) Type() string { return "source.sync-shadow-columns" }

func (c SyncShadowColumnsCommand) Validate() error {
	return c.CreateDefaultColumnsCommand.Validate()
}

type SyncShadowColumnsHandler struct {
	rpc ports.RPCClient
}

func NewSyncShadowColumnsHandler(rpc ports.RPCClient) *SyncShadowColumnsHandler {
	return &SyncShadowColumnsHandler{rpc: rpc}
}

func (h *SyncShadowColumnsHandler) Handle(ctx context.Context, c ports.Command) (json.RawMessage, error) {
	cmd, ok := c.(SyncShadowColumnsCommand)
	if !ok {
		return nil, errors.New("source.sync-shadow-columns: command type mismatch")
	}
	if h.rpc == nil {
		return nil, errors.New("rpc client not configured")
	}
	payload, err := json.Marshal(cmd.CreateDefaultColumnsCommand)
	if err != nil {
		return nil, err
	}
	replySubj := fmt.Sprintf("cdc.evt.create-default-columns.%d", time.Now().UnixNano())
	rawResp, err := h.rpc.Request(ctx, "cdc.cmd.create-default-columns", replySubj, payload, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("worker timeout hoặc mất kết nối khi đồng bộ cột shadow: %w", err)
	}
	var workerRes struct {
		Status       string `json:"status"`
		Error        string `json:"error"`
		RowsAffected int    `json:"rows_affected"`
		TargetTable  string `json:"target_table"`
	}
	if err := json.Unmarshal(rawResp, &workerRes); err != nil {
		return nil, fmt.Errorf("phản hồi worker không hợp lệ: %w", err)
	}
	if workerRes.Status == "error" {
		if workerRes.Error != "" {
			return nil, errors.New(workerRes.Error)
		}
		return nil, errors.New("đồng bộ cột shadow thất bại tại worker")
	}
	return rawResp, nil
}
```
*(Đăng ký `cmdBus.RegisterSync("source.sync-shadow-columns", sourceCmd.NewSyncShadowColumnsHandler(natsRPCClient))` tại `server.go`; trong `CreateDefaultColumnsV2`, khi `c.QueryBool("wait")` bật thì gọi `h.bus.Execute(ctx, source.SyncShadowColumnsCommand{CreateDefaultColumnsCommand: cmd})` và trả `200` hoặc `422` nếu lỗi — các caller không có `wait=true` giữ nguyên 100% luồng `202 Accepted`).*

2. **`cdc-cms-web/src/pages/MappingFieldsPage.tsx:166-185` (`handleSyncFields`):**
```tsx
  const handleSyncFields = async () => {
    if (!registry) return;
    setSyncFieldsLoading(true);
    const trace = createActionTrace('sync_fields_to_shadow');
    try {
      const scopedBindingID = bindingIdParam ? Number(bindingIdParam) : registry.shadow_binding_id;
      const queryParts = ['wait=true'];
      if (scopedBindingID != null && scopedBindingID > 0) {
        queryParts.push(`binding_id=${scopedBindingID}`);
      }
      const endpoint = `/api/v1/source-objects/${registry.id}/create-default-columns?${queryParts.join('&')}`;
      const { data } = await cmsApi.post(endpoint, actionTraceBody(trace), {
        headers: actionTraceHeaders(trace),
      });
      await Promise.all([fetchRules(), fetchShadowColumns()]);
      message.success(
        `Đã đồng bộ schema bảng shadow thành công (${data?.rows_affected ?? 0} thay đổi). trace=${data?.trace_id || trace.traceId}`,
      );
    } catch (err) {
      await fetchShadowColumns();
      message.error(humanizeApiError(err, 'Đồng bộ kiểu dữ liệu / cột shadow thất bại'));
    } finally {
      setSyncFieldsLoading(false);
    }
  };
```
*(Và tại Tooltip cột `Data Type Current` dòng `482`, nếu `record.status !== 'approved' || !record.is_active` thì hiển thị rõ: `Rule đang "${record.status}" — cần Duyệt (approved) trước rồi bấm "Sync Fields" để ALTER COLUMN TYPE`).*
