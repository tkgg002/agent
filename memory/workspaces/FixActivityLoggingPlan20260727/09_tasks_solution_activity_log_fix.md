# 09_tasks_solution_activity_log_fix.md

## Hồ sơ Giải pháp Kỹ thuật Chuẩn xác: Dynamic Master Schema (`master_scheduler_service`) từ `cdc_system.master_binding`

### 1. Phân tích Naming Convention Thực tế trong Hệ thống CDC

- Trong hệ thống CDC V2, các Master Schema không dùng `public` mặc định mà được quy hoạch theo từng domain/service:
  - **Source Service:** `scheduler-service`
  - **Shadow Schema:** `shadow_testss` (hoặc `shadow_scheduler_service`)
  - **Master Schema:** `master_scheduler_service` (được lưu động tại cột `master_schema` của `cdc_system.master_binding`).
  - **Master Table:** `schedule_histories`

- Khi `activity_log_read_repo_gorm.go` gọi câu SQL Read:
  ```sql
  LEFT JOIN cdc_system.master_binding mb
    ON mb.shadow_binding_id = sb.shadow_binding_id
   AND mb.is_active = TRUE
  ```
  Giá trị `mb.master_schema` sẽ được đọc **hoàn toàn động từ DB** (trả về `"master_scheduler_service"` cho `schedule_histories`), tuyệt đối KHÔNG set tĩnh / hardcode bất kỳ chuỗi nào trong code Go!

---

### 2. Code Specifications

#### File 1: `cdc-cms-service/internal/app/queries/system/activity_log_read_models.go`
```go
type ActivityLogRow struct {
	ID              uint64          `json:"id"`
	Operation       string          `json:"operation"`
	TargetTable     string          `json:"target_table"`
	SourceDatabase  *string         `json:"source_database,omitempty"`
	SourceSchema    *string         `json:"source_schema,omitempty"`
	SourceNamespace *string         `json:"source_namespace,omitempty"`
	SourceTable     *string         `json:"source_table,omitempty"`
	ShadowSchema    *string         `json:"shadow_schema,omitempty"`
	ShadowTable     *string         `json:"shadow_table,omitempty"`
	MasterSchema    *string         `json:"master_schema,omitempty"` // Đọc động mb.master_schema từ DB
	MasterTable     *string         `json:"master_table,omitempty"`  // Đọc động mb.master_table từ DB
	ScopeAmbiguous  bool            `json:"scope_ambiguous"`
	Status          string          `json:"status"`
	RowsAffected    int64           `json:"rows_affected"`
	DurationMs      *int            `json:"duration_ms"`
	Details         *string         `json:"details"`
	ErrorMessage    *string         `json:"error_message"`
	TriggeredBy     string          `json:"triggered_by"`
	StartedAt       string          `json:"started_at"`
	CompletedAt     *string         `json:"completed_at"`
}
```

#### File 2: `cdc-cms-service/internal/infra/persistence/system/activity_log_read_repo_gorm.go`
```sql
-- baseFromClause():
FROM cdc_activity_log al
LEFT JOIN LATERAL (
	SELECT
		sb.id AS shadow_binding_id,
		sb.source_object_id,
		sb.shadow_schema,
		sb.shadow_table
	FROM cdc_system.shadow_binding sb
	WHERE al.target_table IS NOT NULL
	  AND al.target_table <> '*'
	  AND sb.shadow_table = al.target_table
	  AND sb.is_active = TRUE
	  AND (
	        NULLIF(al.details->>'shadow_binding_id', '') IS NULL
	     OR sb.id = (al.details->>'shadow_binding_id')::bigint
	  )
	ORDER BY sb.updated_at DESC, sb.id DESC
	LIMIT 1
) sb ON TRUE
LEFT JOIN cdc_system.master_binding mb
  ON mb.shadow_binding_id = sb.shadow_binding_id
 AND mb.is_active = TRUE
LEFT JOIN LATERAL (
	SELECT COUNT(*)::int AS binding_count
	FROM cdc_system.shadow_binding sb
	WHERE al.target_table IS NOT NULL
	  AND al.target_table <> '*'
	  AND sb.shadow_table = al.target_table
	  AND sb.is_active = TRUE
) scope_counts ON TRUE
LEFT JOIN cdc_system.source_object_registry so
  ON so.id = sb.source_object_id
WHERE 1=1

-- projectionColumns():
SELECT
	al.id,
	al.operation,
	al.target_table,
	so.source_database,
	so.source_schema,
	so.source_namespace,
	so.source_object_name AS source_table,
	sb.shadow_schema,
	sb.shadow_table,
	mb.master_schema,
	mb.master_table,
	COALESCE(scope_counts.binding_count, 0) > 1 AS scope_ambiguous,
	al.status,
	...
```

#### File 3: `cdc-cms-web/src/pages/ActivityLog.tsx`
```tsx
// ActivityLog.tsx - Render cột Scope cho Transmute log
{
  title: 'Scope', dataIndex: 'target_table', width: 260,
  render: (v, r) => {
    if (v === '*') return <Tag>ALL</Tag>;

    if (r.operation === 'transmute') {
      const masterFqn = r.master_schema && r.master_table 
        ? `${r.master_schema}.${r.master_table}` 
        : (r.master_schema ? `${r.master_schema}.${v}` : v);
      return (
        <Space orientation="vertical" size={0}>
          <Text style={{ fontSize: 12 }} type="secondary">
            {r.shadow_schema ? `${r.shadow_schema}.${r.shadow_table}` : r.source_table}
          </Text>
          <Space size={4}>
            <Tag color="blue">Master: {masterFqn}</Tag>
            {r.scope_ambiguous ? <Tag color="orange">Ambiguous</Tag> : null}
          </Space>
        </Space>
      );
    }

    if (r.source_database && r.source_table && r.shadow_schema && r.shadow_table) {
      return (
        <Space orientation="vertical" size={0}>
          <Text>{r.source_database}.{r.source_table}</Text>
          <Space size={6}>
            <Text type="secondary" code>{r.shadow_schema}.{r.shadow_table}</Text>
            {r.scope_ambiguous ? <Tag color="orange">Ambiguous</Tag> : null}
          </Space>
        </Space>
      );
    }
    return <strong>{v}</strong>;
  },
}
```

---

### 3. API Output JSON Thực tế (Dynamic Data từ DB)

#### Log #28232 (`kafka-consumer` - Ingest Shadow):
```json
{
    "id": 28232,
    "operation": "kafka-consumer",
    "target_table": "schedule_histories",
    "source_database": "scheduler-service",
    "source_namespace": "scheduler-service",
    "source_table": "schedule_histories",
    "shadow_schema": "shadow_testss",
    "shadow_table": "schedule_histories",
    "master_schema": "master_scheduler_service",
    "master_table": "schedule_histories",
    "scope_ambiguous": false,
    "status": "success",
    "rows_affected": 2,
    "duration_ms": 57,
    "details": "{\"batch_size\":4,\"written\":2}",
    "error_message": null,
    "triggered_by": "kafka-consumer",
    "started_at": "2026-07-27T06:48:39.559Z",
    "completed_at": "2026-07-27T06:48:39.617Z"
}
```

#### Log #28233 (`transmute` - Master Transmute):
```json
{
    "id": 28233,
    "operation": "transmute",
    "target_table": "schedule_histories",
    "source_database": "scheduler-service",
    "source_namespace": "scheduler-service",
    "source_table": "schedule_histories",
    "shadow_schema": "shadow_testss",
    "shadow_table": "schedule_histories",
    "master_schema": "master_scheduler_service",
    "master_table": "schedule_histories",
    "scope_ambiguous": false,
    "status": "success",
    "rows_affected": 2,
    "duration_ms": 26,
    "details": "{\"active_gate\":\"\",\"correlation_id\":\"corr-cdc-9821a\",\"inserted\":2,\"rule_misses\":0,\"scanned\":2,\"skipped\":0,\"type_errors\":0,\"updated\":0}",
    "error_message": null,
    "triggered_by": "kafka-consumer-hook",
    "started_at": "2026-07-27T06:48:39.738Z",
    "completed_at": "2026-07-27T06:48:39.764Z"
}
```
*(Trên UI hiển thị đúng Master FQN: `<Tag color="blue">Master: master_scheduler_service.schedule_histories</Tag>`)*
