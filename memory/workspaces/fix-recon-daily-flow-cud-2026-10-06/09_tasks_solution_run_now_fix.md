# HỒ SƠ GIẢI PHÁP KỸ THUẬT: ĐA KẾT NỐI RUNNER & ACTIVITY LOGGING TOÀN TRÌNH

## 1. Giải pháp cho `SourceCountRunner` (`source_count_runner.go`)

### 1.1 Khai báo cấu trúc và khởi tạo
```go
type SourceCountRunner struct {
	dbControl      *gorm.DB
	dbShadow       *gorm.DB
	dbMaster       *gorm.DB
	dbReg          *database.Registry
	connMgr        *source.ConnectionManager
	activityLogger *governance.ActivityLogger
	logger         *zap.Logger
}

func NewSourceCountRunner(
	dbControl *gorm.DB,
	dbShadow *gorm.DB,
	dbMaster *gorm.DB,
	dbReg *database.Registry,
	connMgr *source.ConnectionManager,
	activityLogger *governance.ActivityLogger,
	logger *zap.Logger,
) *SourceCountRunner {
	return &SourceCountRunner{
		dbControl:      dbControl,
		dbShadow:       dbShadow,
		dbMaster:       dbMaster,
		dbReg:          dbReg,
		connMgr:        connMgr,
		activityLogger: activityLogger,
		logger:         logger,
	}
}
```

### 1.2 Phân giải kết nối động theo Connection Code
```go
// Trong switch params.LayerType:
case "master":
	targetDB := r.dbMaster
	if r.connMgr != nil && params.ConnectionCode != "" {
		if dynDB, err := r.connMgr.GetMasterDB(queryCtx, params.ConnectionCode); err == nil && dynDB != nil {
			targetDB = dynDB
		} else if err != nil {
			r.logger.Warn("cannot resolve dynamic master DB, fallback to default",
				zap.String("conn_code", params.ConnectionCode), zap.Error(err))
		}
	}
	countResult, meta, countErr = r.countSQL(queryCtx, targetDB, params, startOfDay, nextDayStart)

case "shadow":
	targetDB := r.dbShadow
	if r.connMgr != nil && params.ConnectionCode != "" {
		if dynDB, err := r.connMgr.GetShadowDB(queryCtx, params.ConnectionCode); err == nil && dynDB != nil {
			targetDB = dynDB
		}
	}
	countResult, meta, countErr = r.countSQL(queryCtx, targetDB, params, startOfDay, nextDayStart)
```

### 1.3 Ghi nhận Activity Log đầy đủ
```go
// Xác định triggered_by
triggeredBy := activity.TriggeredByScheduler.String()
if sched.NextRunAt != nil && !sched.NextRunAt.After(now) && (sched.LastRunAt == nil || sched.LastRunAt.Before(*sched.NextRunAt)) {
	triggeredBy = activity.TriggeredByManual.String()
}

targetName := params.ObjectName
if targetName == "" && sched.TargetTable != nil {
	targetName = *sched.TargetTable
}

// Khi lỗi:
if countErr != nil {
	// ... cập nhật last_error trên cdc_worker_schedule ...
	if r.activityLogger != nil {
		r.activityLogger.Quick("source_count", targetName, triggeredBy, "error", 0, meta, countErr.Error())
	}
	return
}

// Khi thành công:
if r.activityLogger != nil {
	r.activityLogger.Quick("source_count", targetName, triggeredBy, "success", countResult, meta, "")
}
```

---

## 2. Giải pháp cho `ConfluenceReportRunner` (`confluence_runner.go`)

### 2.1 Cấu trúc & Khởi tạo
```go
type ConfluenceReportRunner struct {
	dbControl        *gorm.DB
	confluenceClient *confluenceclient.Client
	activityLogger   *governance.ActivityLogger
	logger           *zap.Logger
}

func NewConfluenceReportRunner(
	dbControl *gorm.DB,
	client *confluenceclient.Client,
	activityLogger *governance.ActivityLogger,
	logger *zap.Logger,
) *ConfluenceReportRunner {
	if client == nil {
		client = confluenceclient.NewClient()
	}
	return &ConfluenceReportRunner{
		dbControl:        dbControl,
		confluenceClient: client,
		activityLogger:   activityLogger,
		logger:           logger,
	}
}
```

### 2.2 Ghi nhận Activity Log
```go
triggeredBy := activity.TriggeredByScheduler.String()
if sched.NextRunAt != nil && !sched.NextRunAt.After(now) && (sched.LastRunAt == nil || sched.LastRunAt.Before(*sched.NextRunAt)) {
	triggeredBy = activity.TriggeredByManual.String()
}

logTarget := "all_selected_tables"
if sched.TargetTable != nil && *sched.TargetTable != "" {
	logTarget = *sched.TargetTable
}

// Khi lỗi:
if err != nil {
	// ... cập nhật last_error ...
	if r.activityLogger != nil {
		r.activityLogger.Quick("confluence", logTarget, triggeredBy, "error", 0, map[string]interface{}{"table_count": len(rows)}, err.Error())
	}
	return
}

// Khi thành công:
if r.activityLogger != nil {
	r.activityLogger.Quick("confluence", logTarget, triggeredBy, "success", int64(len(rows)), map[string]interface{}{
		"page_url":    pageURL,
		"table_count": len(rows),
	}, "")
}
```

---

## 3. Tối ưu Ticker Scheduler (`server_scheduler.go`)
```go
ticker := time.NewTicker(2 * time.Second)
```
- Phản ứng tức thời (trong 0–2 giây) ngay khi người dùng nhấn "Chạy ngay" trên giao diện CMS.
