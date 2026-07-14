# Hồ sơ Giải pháp Kỹ thuật - Quản lý Index qua CMS UI (Index Manager)

Đây là tài liệu chi tiết các thay đổi code cho cả 3 layer: Worker, CMS API và Frontend.

---

## 1. centralized-data-service (Worker)

### 1.1. Service: `internal/service/governance/index_manager.go` [NEW]
Định nghĩa struct `IndexManager` để query metadata index và chạy câu lệnh DDL.
```go
package governance

import (
	"context"
	"fmt"
	"strings"
	"centralized-data-service/pkgs/sqlutil"
	"gorm.io/gorm"
)

type IndexInfo struct {
	IndexName  string `json:"index_name"`
	IndexDef   string `json:"index_def"`
	IndexSize  string `json:"index_size"`
	ScanCount  int64  `json:"scan_count"`
	IsValid    bool   `json:"is_valid"`
}

type IndexManager struct {
	// Không giữ cứng db connection, vì DB phụ thuộc vào plane (shadow/master) truyền từ runtime
}

func NewIndexManager() *IndexManager {
	return &IndexManager{}
}

func (im *IndexManager) ListIndexes(ctx context.Context, db *gorm.DB, schema, table string) ([]IndexInfo, error) {
	if !sqlutil.IsSafeIdent(schema) || !sqlutil.IsSafeIdent(table) {
		return nil, fmt.Errorf("invalid schema or table identifier")
	}

	query := `
		SELECT 
			c.relname AS index_name,
			pg_get_indexdef(i.indexrelid) AS index_def,
			pg_size_pretty(pg_relation_size(i.indexrelid)) AS index_size,
			coalesce(stat.idx_scan, 0) AS scan_count,
			i.indisvalid AS is_valid
		FROM pg_index i
		JOIN pg_class c ON c.oid = i.indexrelid
		JOIN pg_class t ON t.oid = i.indrelid
		JOIN pg_namespace n ON n.oid = t.relnamespace
		LEFT JOIN pg_stat_user_indexes stat ON stat.indexrelid = i.indexrelid
		WHERE n.nspname = ? AND t.relname = ?
	`

	var results []IndexInfo
	rows, err := db.WithContext(ctx).Raw(query, schema, table).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var info IndexInfo
		if err := rows.Scan(&info.IndexName, &info.IndexDef, &info.IndexSize, &info.ScanCount, &info.IsValid); err != nil {
			return nil, err
		}
		results = append(results, info)
	}

	return results, nil
}

func (im *IndexManager) CreateIndexConcurrently(ctx context.Context, db *gorm.DB, schema, table string, columns []string, isUnique bool, isPartial bool, whereClause string) error {
	if !sqlutil.IsSafeIdent(schema) || !sqlutil.IsSafeIdent(table) {
		return fmt.Errorf("invalid schema or table identifier")
	}
	if len(columns) == 0 {
		return fmt.Errorf("columns list cannot be empty")
	}

	var safeCols []string
	var colPartName []string
	for _, col := range columns {
		col = strings.TrimSpace(col)
		if !sqlutil.IsSafeIdent(col) {
			return fmt.Errorf("invalid column identifier: %s", col)
		}
		safeCols = append(safeCols, fmt.Sprintf(`"%s"`, col))
		colPartName = append(colPartName, col)
	}

	// Generate a unique index name if not provided
	idxType := "idx"
	if isUnique {
		idxType = "ux"
	}
	indexName := fmt.Sprintf("%s_%s_%s", idxType, table, strings.Join(colPartName, "_"))
	if len(indexName) > 63 {
		indexName = indexName[:63]
	}

	uniqueStr := ""
	if isUnique {
		uniqueStr = "UNIQUE"
	}

	whereStr := ""
	if isPartial && whereClause != "" {
		// Basic sanitization check for partial WHERE. Simple whitelist/blacklist approach
		upperWhere := strings.ToUpper(whereClause)
		if strings.Contains(upperWhere, ";") || strings.Contains(upperWhere, "DROP") || strings.Contains(upperWhere, "DELETE") {
			return fmt.Errorf("unsafe characters detected in WHERE clause")
		}
		whereStr = " WHERE " + whereClause
	}

	ddl := fmt.Sprintf(`CREATE %s INDEX CONCURRENTLY IF NOT EXISTS "%s" ON "%s"."%s" (%s)%s`,
		uniqueStr, indexName, schema, table, strings.Join(safeCols, ", "), whereStr)

	// CONCURRENTLY cannot run inside transaction, we must use raw connection
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	_, err = sqlDB.ExecContext(ctx, ddl)
	return err
}

func (im *IndexManager) DropIndexConcurrently(ctx context.Context, db *gorm.DB, schema, indexName string) error {
	if !sqlutil.IsSafeIdent(schema) || !sqlutil.IsSafeIdent(indexName) {
		return fmt.Errorf("invalid schema or index identifier")
	}

	// Prevent dropping primary key or unique system index starting with pk_ or ux_ by default, unless requested
	if strings.HasPrefix(indexName, "pk_") || strings.HasPrefix(indexName, "ux_") {
		return fmt.Errorf("protected index cannot be dropped: %s", indexName)
	}

	ddl := fmt.Sprintf(`DROP INDEX CONCURRENTLY IF EXISTS "%s"."%s"`, schema, indexName)

	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	_, err = sqlDB.ExecContext(ctx, ddl)
	return err
}
```

### 1.2. Handler: `internal/handler/governance/index_handler.go` [NEW]
Xử lý các request NATS.
```go
package governance

import (
	"context"
	"encoding/json"
	"strings"
	base "centralized-data-service/internal/handler/base"
	"centralized-data-service/internal/service/source"
	"centralized-data-service/pkgs/observability"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type IndexHandler struct {
	base.BaseHandler
	connMgr      *source.ConnectionManager
	indexManager *IndexManager
}

func NewIndexHandler(db *gorm.DB, natsConn *nats.Conn, logger *zap.Logger, connMgr *source.ConnectionManager, im *IndexManager) *IndexHandler {
	return &IndexHandler{
		BaseHandler:  base.NewBaseHandler(db, natsConn, logger),
		connMgr:      connMgr,
		indexManager: im,
	}
}

func (h *IndexHandler) HandleIntrospectIndexes(msg *nats.Msg) {
	ctx := observability.ExtractNATSHeader(context.Background(), msg.Header)
	ctx, span := observability.ChildSpan(ctx, "nats.HandleIntrospectIndexes")
	defer span.End()

	var payload struct {
		Schema string `json:"schema"`
		Table  string `json:"table"`
		Plane  string `json:"plane"` // "shadow" or "master"
	}
	if err := json.Unmarshal(msg.Data, &payload); err != nil {
		h.PublishResult(ctx, msg, base.CommandResult{Command: "introspect-indexes", Status: "error", Error: "invalid payload"})
		return
	}

	var targetDB *gorm.DB
	var err error
	if strings.ToLower(payload.Plane) == "master" {
		targetDB, err = h.connMgr.GetMasterDB(ctx, "default")
	} else {
		targetDB, err = h.connMgr.GetShadowDB(ctx, "default")
	}

	if err != nil {
		h.PublishResult(ctx, msg, base.CommandResult{Command: "introspect-indexes", Status: "error", Error: "failed to get database connection: " + err.Error()})
		return
	}

	indexes, err := h.indexManager.ListIndexes(ctx, targetDB, payload.Schema, payload.Table)
	if err != nil {
		h.PublishResult(ctx, msg, base.CommandResult{Command: "introspect-indexes", Status: "error", Error: err.Error()})
		return
	}

	respData, _ := json.Marshal(map[string]interface{}{
		"command": "introspect-indexes",
		"status":  "success",
		"indexes": indexes,
	})
	h.NatsPublish(msg, "cdc.result.introspect-indexes", respData)
}

func (h *IndexHandler) HandleCreateIndex(msg *nats.Msg) {
	ctx := observability.ExtractNATSHeader(context.Background(), msg.Header)
	ctx, span := observability.ChildSpan(ctx, "nats.HandleCreateIndex")
	defer span.End()

	var payload struct {
		Schema      string   `json:"schema"`
		Table       string   `json:"table"`
		Columns     []string `json:"columns"`
		Plane       string   `json:"plane"`
		IsUnique    bool     `json:"is_unique"`
		IsPartial   bool     `json:"is_partial"`
		WhereClause string   `json:"where_clause"`
	}

	if err := json.Unmarshal(msg.Data, &payload); err != nil {
		h.PublishResult(ctx, msg, base.CommandResult{Command: "create-index", Status: "error", Error: "invalid payload"})
		return
	}

	var targetDB *gorm.DB
	var err error
	if strings.ToLower(payload.Plane) == "master" {
		targetDB, err = h.connMgr.GetMasterDB(ctx, "default")
	} else {
		targetDB, err = h.connMgr.GetShadowDB(ctx, "default")
	}

	if err != nil {
		h.PublishResult(ctx, msg, base.CommandResult{Command: "create-index", Status: "error", Error: "failed to get db: " + err.Error()})
		return
	}

	err = h.indexManager.CreateIndexConcurrently(ctx, targetDB, payload.Schema, payload.Table, payload.Columns, payload.IsUnique, payload.IsPartial, payload.WhereClause)
	if err != nil {
		h.PublishResult(ctx, msg, base.CommandResult{Command: "create-index", Status: "error", Error: err.Error()})
		return
	}

	h.PublishResult(ctx, msg, base.CommandResult{Command: "create-index", TargetTable: payload.Table, Status: "success"})
}

func (h *IndexHandler) HandleDropIndex(msg *nats.Msg) {
	ctx := observability.ExtractNATSHeader(context.Background(), msg.Header)
	ctx, span := observability.ChildSpan(ctx, "nats.HandleDropIndex")
	defer span.End()

	var payload struct {
		Schema    string `json:"schema"`
		IndexName string `json:"index_name"`
		Plane     string `json:"plane"`
	}

	if err := json.Unmarshal(msg.Data, &payload); err != nil {
		h.PublishResult(ctx, msg, base.CommandResult{Command: "drop-index", Status: "error", Error: "invalid payload"})
		return
	}

	var targetDB *gorm.DB
	var err error
	if strings.ToLower(payload.Plane) == "master" {
		targetDB, err = h.connMgr.GetMasterDB(ctx, "default")
	} else {
		targetDB, err = h.connMgr.GetShadowDB(ctx, "default")
	}

	if err != nil {
		h.PublishResult(ctx, msg, base.CommandResult{Command: "drop-index", Status: "error", Error: "failed to get db: " + err.Error()})
		return
	}

	err = h.indexManager.DropIndexConcurrently(ctx, targetDB, payload.Schema, payload.IndexName)
	if err != nil {
		h.PublishResult(ctx, msg, base.CommandResult{Command: "drop-index", Status: "error", Error: err.Error()})
		return
	}

	h.PublishResult(ctx, msg, base.CommandResult{Command: "drop-index", Status: "success"})
}
```
