# Technical Solution: Fix Log Trace Connection

Tài liệu thiết kế chi tiết các thay đổi mã nguồn trong dự án `centralized-data-service`.

## 1. base_handler.go
**File**: `internal/handler/base/base_handler.go`

- Thêm import `"centralized-data-service/pkgs/observability"`.
- Cập nhật hàm `NatsPublish` để tự động trích xuất `ctx` từ `msg.Header` và dùng `observability.Ctx(ctx, h.Logger)` khi ghi logs, đồng thời pass `ctx` vào `LogCommandResult`.
- Cập nhật hàm `PublishResultWithSubject`:
  ```go
  observability.Ctx(ctx, h.Logger).Error("command failed", zap.String("command", result.Command), zap.String("error", result.Error))
  ```
- Cập nhật hàm `PublishResult`:
  ```go
  h.LogCommandResult(ctx, safeResult)
  ```
- Cập nhật hàm `LogCommandResult`:
  ```go
  func (h *BaseHandler) LogCommandResult(ctx context.Context, res CommandResult, extraFields ...zap.Field) {
      // ...
      if res.Status == "error" {
          observability.Ctx(ctx, h.Logger).Error("command result status: error", fields...)
      } else {
          observability.Ctx(ctx, h.Logger).Info("command result status: success", fields...)
      }
  }
  ```
- Cập nhật hàm `ConnectCall`:
  ```go
  observability.Ctx(ctx, h.Logger).Warn("HTTP request failed", ...)
  observability.Ctx(ctx, h.Logger).Warn("HTTP request returned error status", ...)
  ```

## 2. bridge_handler.go & bridge_mongo.go
**Files**: `internal/handler/source/bridge_handler.go`, `internal/handler/source/bridge_mongo.go`

- Sửa đổi hàm `resolveCollection` trong `bridge_handler.go` để nhận `ctx context.Context`:
  ```go
  func (h *BridgeHandler) resolveCollection(ctx context.Context, coll BridgeCollection, payloadConnString string) resolvedCollection
  ```
  Và gọi log bằng `observability.Ctx(ctx, h.logger)`.
- Ghi log trong `processBridge`, `processCollection`, `batchUpsert`, `triggerTransmute` bằng `observability.Ctx(ctx, h.logger)`.
- Ghi log trong `MongoBridgeStrategy.ReadChanges` bằng `observability.Ctx(ctx, s.logger)`.

## 3. recon_execute_heal_handler.go & recon_check_handler.go & recon_sysops_handler.go & recon_heal_fetch.go
**Files**: `internal/handler/recon/...`

- Tất cả các logs dùng `h.logger` trong `recon_execute_heal_handler.go` (cả sub-methods) chuyển sang bọc `observability.Ctx(ctx, h.logger)`.
- Logs trong `HandleReconCheck` bọc bằng `observability.Ctx(ctx, h.logger)`.
- Logs trong `recon_sysops_handler.go` (ví dụ `HandleDebeziumSignal`, `HandleRetryFailedLog`, `HandleBackfillData`, `HandleDetectTimestampField`) bọc bằng `observability.Ctx(ctx, h.logger)`.
- Đổi chữ ký `fetchSourceDocsForHeal` nhận `ctx context.Context` và bọc logs bằng `observability.Ctx`.

## 4. schema_ddl_handler.go & batch_transform_handler.go
**Files**: `internal/handler/shadow/...`

- Logs trong các methods của `schema_ddl_handler.go` chuyển sang dùng `observability.Ctx(ctx, h.logger)`.
- Logs trong `batch_transform_handler.go` chuyển sang dùng `observability.Ctx(ctx, h.logger)`.

## 5. transmute_handler.go
**File**: `internal/handler/master/transmute_handler.go`

- Logs trong `HandleTransmuteShadow`, `executeTransmute` chuyển sang dùng `observability.Ctx(ctx, h.logger)`.

## 6. scan_handler.go
**File**: `internal/handler/scan/scan_handler.go`

- Logs trong `HandleScanRawData`, `HandleScanArrayFields`, `HandlePeriodicScan` chuyển sang dùng `observability.Ctx(ctx, h.Logger)`.

## 7. snapshot_runner_handler.go
**File**: `internal/handler/orchestration/snapshot_runner_handler.go`

- Logs trong `HandleRunSnapshot`, `processSnapshotRun` chuyển sang dùng `observability.Ctx(ctx, r.logger)`.

## 8. Sửa đổi các file test để tương thích với các thay đổi query DB
**Files**: 
- `internal/handler/shadow/batch_transform_handler_test.go`
- `internal/handler/scan/scan_handler_test.go`

### Thay đổi trong `batch_transform_handler_test.go`
Cập nhật WithArgs của `ExpectQuery` cho `mapping_rule_v2` nhận 4 tham số `"users", "users", true, "approved"` thay vì 3 tham số để khớp với repository query thực tế:
```go
	mock.ExpectQuery(`SELECT .* FROM "cdc_system"\."mapping_rule_v2" JOIN cdc_system\.source_object_registry so .* WHERE .*so\.source_object_name = \$1 .*`).
		WithArgs("users", "users", true, "approved").
		WillReturnRows(rulesRows)
```
Áp dụng cho cả `TestHandleBatchTransform_Success` và `TestHandleBatchTransform_UnchunkedFallback`.

### Thay đổi trong `scan_handler_test.go`
Thêm các mock query bị thiếu vào `TestHandleScanArrayFields_ReplyToAndUnmarshalOrder` trước câu truy vấn `jsonb_typeof` (đặt `hasAfter` trả về `false` để giữ nguyên pgPath `{items}`):
```go
	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM "public"\."payment_bills" WHERE _raw_data \? 'after'\)`).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	mock.ExpectQuery(`SELECT transform_type FROM "cdc_system"\."master_binding" WHERE id = .*`).
		WithArgs(int64(100)).
		WillReturnRows(sqlmock.NewRows([]string{"transform_type"}).AddRow("array_explode"))
```

### Thay đổi trong `recon_job_handler_test.go`
Bổ sung mock method `GetActiveJobs` cho `mockJobRepoHandler` để sửa lỗi biên dịch:
```go
func (m *mockJobRepoHandler) GetActiveJobs(ctx context.Context, targetTable string) ([]repository.ReconJob, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var list []repository.ReconJob
	for _, job := range m.jobs {
		if job.TargetTable == targetTable && job.Status == "RUNNING" {
			list = append(list, *job)
		}
	}
	return list, nil
}
```
