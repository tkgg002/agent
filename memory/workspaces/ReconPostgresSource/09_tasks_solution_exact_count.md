# Technical Solution: Exact Count for PostgreSQL Source

## 1. Mục tiêu
Thay đổi cuộc gọi `EstimatedCount` thành `CountDocuments` đối với PostgreSQL nguồn để tránh sai số ước lượng gây drift giả.

## 2. Chi tiết thay đổi mã nguồn

### Tệp tin: `internal/service/recon/recon_tier_a.go`

Tại phương thức `RunTier1`, thực hiện thay thế khối mã nguồn sau:

```go
	// Trước khi thay đổi:
	srcEst, errE := rc.sourceAgent.EstimatedCount(fastCtx, entry.SourceURL, entry.SourceDB, entry.SourceTable)
	if errE != nil {
		status = "failed"
		return rc.errorReport(entry, "count", 1, fmt.Errorf("src estimated count: %w", errE))
	}
```

Thay thế bằng khối mã nguồn rẽ nhánh rành mạch:

```go
	// Sau khi thay đổi:
	var srcEst int64
	var errE error
	if isPostgres(entry.SourceURL) {
		srcEst, errE = rc.sourceAgent.CountDocuments(fastCtx, entry.SourceURL, entry.SourceDB, entry.SourceTable)
	} else {
		srcEst, errE = rc.sourceAgent.EstimatedCount(fastCtx, entry.SourceURL, entry.SourceDB, entry.SourceTable)
	}
	if errE != nil {
		status = "failed"
		return rc.errorReport(entry, "count", 1, fmt.Errorf("src count: %w", errE))
	}
```

## 3. Kế hoạch xác minh
- Thực hiện biên dịch thử dự án và chạy bộ unit test của package `recon`:
  ```bash
  go test -v -count=1 ./internal/service/recon/...
  ```
