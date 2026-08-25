# 09_tasks_solution_sftp_scan_fix.md

## Hồ sơ Giải pháp Kỹ thuật Chi tiết (Technical Solution)

### 1. Thay đổi tại `centralized-data-service/internal/handler/source/discover_handler.go`

Thêm helper function `isFileOrStreamSource(sourceType string)`:
```go
func isFileOrStreamSource(sourceType string) bool {
	st := strings.ToLower(strings.TrimSpace(sourceType))
	return st == "sftp" || st == "file" || st == "csv" || st == "json" || st == "kafka"
}
```

Cập nhật `ScanFieldsDebezium`:
```go
	if !h.discoverSvc.TableExistsInShadow(ctx, schemaName, targetTable) || !h.discoverSvc.HasColumnInShadow(ctx, schemaName, targetTable, "_raw_data") {
		if sourceType == "mongodb" {
			return h.scanFieldsMongoSource(ctx, v2ObjectID, shadowBindingID, sourceTable, autoApprove)
		}
		if isSQLSource(sourceType) {
			return h.scanFieldsSQLSource(ctx, v2ObjectID, shadowBindingID, sourceTable, autoApprove)
		}
		if isFileOrStreamSource(sourceType) {
			return 0, 0, fmt.Errorf("SFTP/File source '%s' chưa có dữ liệu trong shadow DB. Vui lòng kiểm tra file CSV mẫu trong thư mục SFTP input", targetTable)
		}
		return 0, 0, fmt.Errorf("table %s has no _raw_data column in shadow db", targetTable)
	}

	rows, err := h.discoverSvc.GetShadowSampleRows(ctx, schemaName, targetTable, 100)
	if err != nil {
		return 0, 0, fmt.Errorf("sample raw_data: %w", err)
	}

	if len(rows) == 0 {
		if sourceType == "mongodb" {
			return h.scanFieldsMongoSource(ctx, v2ObjectID, shadowBindingID, sourceTable, autoApprove)
		}
		if isSQLSource(sourceType) {
			return h.scanFieldsSQLSource(ctx, v2ObjectID, shadowBindingID, sourceTable, autoApprove)
		}
		if isFileOrStreamSource(sourceType) {
			return 0, 0, fmt.Errorf("SFTP/File source '%s' shadow table đang rỗng. Vui lòng đảm bảo Kafka Connect đã đọc file CSV và đẩy record vào shadow table", targetTable)
		}
		h.Logger.Info("scan-fields: shadow table is empty, returning success with 0 fields",
			zap.String("table", targetTable))
		return 0, 0, nil
	}
```

### 2. Tạo File CSV mẫu tại `./docker/data/reconcile_final/reconcile_final_20260811.csv`

```csv
id,trans_id,amount,status,created_at
1,TRX1001,50000,SUCCESS,2026-08-11 10:00:00
2,TRX1002,100000,SUCCESS,2026-08-11 10:05:00
```
