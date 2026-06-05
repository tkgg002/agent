# Báo cáo Khắc phục lỗi SLOW SQL trong Probes và System Health Queries

## 1. Thông tin Chung
- **Workspace**: `bug-cms-slow-sql-probes-2026-05-26`
- **Thời gian thực hiện**: 26/05/2026
- **Vấn đề**: Cảnh báo SLOW SQL (>= 200ms) định kỳ ở cdc-cms-service tại `probes/postgres.go` và `system_health_queries.go` do tranh chấp khóa chuẩn bị statement (prepared statement cache mutex contention) của GORM dưới tải song song cao và việc sử dụng hàm `NOW()` động làm ngăn cản cơ chế tối ưu phân hoạch tĩnh (partition pruning) của PostgreSQL.

---

## 2. Các File Đã Thay Đổi
- **[MODIFY]** [postgres.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/observability/probes/postgres.go)
  - Vô hiệu hóa `PrepareStmt` bằng cách chạy các câu lệnh trên GORM Session không cache statement: `db.Session(&gorm.Session{PrepareStmt: false})`.
- **[MODIFY]** [system_health_queries.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/observability/system_health_queries.go)
  - Thêm các gói import `"time"` và `"gorm.io/gorm"`.
  - Thiết lập `db.Session(&gorm.Session{PrepareStmt: false})` cho tất cả 3 hàm thu thập sức khỏe hệ thống: `queryReconciliation`, `queryFailedCount`, và `queryRecentEvents`.
  - Thay thế việc sử dụng hàm `NOW()` trong SQL bằng mốc thời gian tĩnh được tính toán trước ở tầng ứng dụng bằng Go (`time.Now()`, `time.Now().Add(-1 * time.Hour)`, v.v.) và truyền qua placeholder `?`. Điều này giúp PostgreSQL tối ưu hóa tĩnh các phân hoạch (partition pruning) lúc biên dịch và lập kế hoạch (planning).

---

## 3. Chi tiết Thay Đổi Mã Nguồn

### 3.1. postgres.go
```go
// Trước:
gdb := db.WithContext(ctxQ)

// Sau:
gdb := db.Session(&gorm.Session{PrepareStmt: false}).WithContext(ctxQ)
```

### 3.2. system_health_queries.go - Imports
```go
// Trước:
import (
	"context"
	"fmt"

	"cdc-cms-service/internal/model"

	"go.uber.org/zap"
)

// Sau:
import (
	"context"
	"fmt"
	"time"

	"cdc-cms-service/internal/model"

	"go.uber.org/zap"
	"gorm.io/gorm"
)
```

### 3.3. system_health_queries.go - queryReconciliation
```go
// Trước:
var reports []model.ReconciliationReport
err := c.db.WithContext(ctxQ).Raw(
	`SELECT DISTINCT ON (target_table) * FROM cdc_reconciliation_report ORDER BY target_table, checked_at DESC`,
).Scan(&reports).Error

// Sau:
db := c.db.Session(&gorm.Session{PrepareStmt: false}).WithContext(ctxQ)

var reports []model.ReconciliationReport
err := db.Raw(
	`SELECT DISTINCT ON (target_table) * FROM cdc_reconciliation_report ORDER BY target_table, checked_at DESC`,
).Scan(&reports).Error
```

### 3.4. system_health_queries.go - queryFailedCount
```go
// Trước:
var count24h, count1h int64
c.db.WithContext(ctxQ).Table("cdc_system.failed_sync_logs").
	Where("created_at > NOW() - INTERVAL '24 hours' AND created_at <= NOW()").Count(&count24h)
c.db.WithContext(ctxQ).Table("cdc_system.failed_sync_logs").
	Where("created_at > NOW() - INTERVAL '1 hour' AND created_at <= NOW()").Count(&count1h)

// Sau:
now := time.Now()
oneHourAgo := now.Add(-1 * time.Hour)
twentyFourHoursAgo := now.Add(-24 * time.Hour)

db := c.db.Session(&gorm.Session{PrepareStmt: false}).WithContext(ctxQ)

var count24h, count1h int64
db.Table("cdc_system.failed_sync_logs").
	Where("created_at > ? AND created_at <= ?", twentyFourHoursAgo, now).Count(&count24h)
db.Table("cdc_system.failed_sync_logs").
	Where("created_at > ? AND created_at <= ?", oneHourAgo, now).Count(&count1h)
```

### 3.5. system_health_queries.go - queryRecentEvents
```go
// Trước:
var logs []model.ActivityLog
c.db.WithContext(ctxQ).
	Where("created_at > NOW() - INTERVAL '1 day' AND created_at <= NOW()").
	Order("started_at DESC").Limit(10).Find(&logs)

// Sau:
now := time.Now()
oneDayAgo := now.Add(-24 * time.Hour)

db := c.db.Session(&gorm.Session{PrepareStmt: false}).WithContext(ctxQ)

var logs []model.ActivityLog
db.Where("created_at > ? AND created_at <= ?", oneDayAgo, now).
	Order("started_at DESC").Limit(10).Find(&logs)
```

---

## 4. Kết quả Xác Minh
- Chạy biên dịch dự án thành công: `go build ./...`
- Chạy toàn bộ các ca kiểm thử đơn vị thành công (không dùng cache):
  ```bash
  go test -count=1 ./...
  ```
  Kết quả: `PASS` toàn bộ 100%.
