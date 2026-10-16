# 09_tasks_solution_recon_multiconn_isolation.md
## Hồ sơ Giải pháp Kỹ thuật: Khắc phục Lỗi Hiển thị Nhật ký Đối soát & Cô lập Đa Kết nối

---

### 1. Giải pháp Task 1: Sửa lỗi Boolean Logic & Placeholders trong `GetTableHistory`
**Vị trí:** `cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go`
**Mã nguồn cũ (Lỗi):**
```go
	where := "(shadow_table = ? OR master_table = ?)"
	args := []interface{}{table, table}
	if shadowSchema != "" {
		where = "((shadow_schema = ? AND shadow_table = ?) OR (segment = 'shadow_master' AND master_table = ?))"
		args = []interface{}{shadowSchema, table, table}
		if masterTable != "" {
			where += " AND master_table = ?"
			args = append(args, masterTable)
		}
	}
```
**Mã nguồn mới (Chuẩn hóa):**
```go
	var where string
	var args []interface{}

	if shadowSchema != "" {
		if masterTable != "" {
			// Segment A: lọc theo (shadow_schema, shadow_table) và segment != 'shadow_master'
			// Segment B: lọc theo (shadow_schema, master_table) và segment = 'shadow_master'
			where = "((shadow_schema = ? AND shadow_table = ? AND (segment IS NULL OR segment != 'shadow_master')) OR (segment = 'shadow_master' AND shadow_schema = ? AND master_table = ?))"
			args = []interface{}{shadowSchema, table, shadowSchema, masterTable}
		} else {
			where = "((shadow_schema = ? AND shadow_table = ?) OR (segment = 'shadow_master' AND shadow_schema = ?))"
			args = []interface{}{shadowSchema, table, shadowSchema}
		}
	} else {
		if masterTable != "" {
			where = "(shadow_table = ? OR master_table = ?)"
			args = []interface{}{table, masterTable}
		} else {
			where = "(shadow_table = ? OR master_table = ?)"
			args = []interface{}{table, table}
		}
	}
```
**Phân tích kỹ thuật:**
- Nhánh Segment A: `shadow_schema = ? AND shadow_table = ? AND (segment IS NULL OR segment != 'shadow_master')`. Nhánh này KHÔNG bị áp điều kiện `master_table = ?`, cho phép 100% record chặng Source $\rightarrow$ Shadow (cả Smoke lẫn Manual Recon) hiển thị chính xác.
- Nhánh Segment B: `segment = 'shadow_master' AND shadow_schema = ? AND master_table = ?`. Nhánh này lọc đúng master table đích của pipeline mà không bị mâu thuẫn placeholder.

---

### 2. Giải pháp Task 2: Gỡ bỏ Master Binding Gate trong `validatePipelineConnections`
**Vị trí:** `centralized-data-service/internal/service/recon/recon_engine_run.go`
**Mã nguồn mới:**
```go
func (rc *ReconCore) validatePipelineConnections(ctx context.Context, targetSchema, targetTable string) (bool, error) {
	var check pipelineConnCheck
	err := rc.db.WithContext(ctx).Raw(`
		SELECT 
			cr_src.id AS source_conn_id,
			cr_sh.id AS shadow_conn_id
		FROM cdc_system.shadow_binding sb
		JOIN cdc_system.source_object_registry sor ON sor.id = sb.source_object_id AND sor.is_active = true
		JOIN cdc_system.connection_registry cr_src ON cr_src.id = sor.source_connection_id AND cr_src.status = 'active'
		JOIN cdc_system.connection_registry cr_sh ON cr_sh.id = sb.shadow_connection_id AND cr_sh.status = 'active'
		WHERE sb.shadow_schema = ? AND sb.shadow_table = ? AND sb.is_active = true
		LIMIT 1
	`, targetSchema, targetTable).Scan(&check).Error

	if err != nil {
		return false, err
	}

	if check.SourceConnID <= 0 || check.ShadowConnID <= 0 {
		return false, nil
	}

	return true, nil
}
```
**Phân tích:** Segment A là chặng Ingest độc lập. Chỉ cần nguồn và shadow active là hợp lệ để chạy Smoke Check.

---

### 3. Giải pháp Task 3: Chuẩn hóa Qualified Target Table cho `recon_jobs`
**Vị trí:** `centralized-data-service/internal/handler/recon/recon_check_handler.go`
**Mã nguồn mới:**
```go
	targetTable := payload.Table
	if payload.ShadowSchema != "" && !strings.Contains(targetTable, ".") {
		targetTable = payload.ShadowSchema + "." + targetTable
	}

	// pg.create_recon_job — DB insert span
	if h.jobRepo != nil {
		ctxDB, dbSpan := observability.ChildSpan(ctx, "pg.create_recon_job",
			attribute.String("recon.job_id", jobID),
			attribute.String("recon.table", targetTable),
			attribute.String("db.statement", fmt.Sprintf(
				"INSERT INTO cdc_system.recon_jobs (job_id, target_table, start_time, end_time, status) VALUES ('%s', '%s', '%s', '%s', 'PENDING')",
				jobID, targetTable,
				startTime.UTC().Format(time.RFC3339),
				endTime.UTC().Format(time.RFC3339),
			)),
		)
		var jobTraceID *string
		if sc := span.SpanContext(); sc.IsValid() {
			tid := sc.TraceID().String()
			jobTraceID = &tid
		}
		job := &repository.ReconJob{
			JobID:       jobID,
			TargetTable: targetTable,
			StartTime:   startTime,
			EndTime:     endTime,
			Status:      servicerecon.StatusPending,
			TraceID:     jobTraceID,
		}
		var createErr error
		createErr = h.jobRepo.Create(ctxDB, job)
		observability.EndSpan(dbSpan, &createErr)
		if createErr != nil {
			observability.Ctx(ctx, h.logger).Error("failed creating recon job", zap.Error(createErr))
		}
	}
```

---

### 4. Giải pháp Task 4: Chống duplicate dòng trong `ListLatest` (CMS Service)
**Vị trí:** `cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go`
Thay thế:
```sql
LEFT JOIN cdc_system.cdc_table_registry reg ON reg.target_table = s.shadow_table AND (COALESCE(s.source_db, '') = '' OR reg.source_db = s.source_db) AND reg.is_active = TRUE
```
bằng:
```sql
LEFT JOIN LATERAL (
    SELECT * FROM cdc_system.cdc_table_registry 
    WHERE target_table = s.shadow_table 
      AND (COALESCE(s.source_db, '') = '' OR source_db = s.source_db) 
      AND is_active = TRUE 
    ORDER BY id DESC LIMIT 1
) reg ON TRUE
```

---

### 5. Giải pháp Task 5: Phân lập Multi-Connector trên UI
**Vị trí:** `cdc-cms-web/src/components/ReconPipelineGrid.tsx`
Khóa gom nhóm `uniqueMap`:
```ts
    const schema = r.shadow_schema || '';
    const table = r.target_table;
    const segment = r.segment || '';
    const srcConn = r.source_connection_code || '';
    const key = segment === 'shadow_master'
      ? `${srcConn}::${schema}::${table}::${r.master_schema || ''}::${segment}`
      : `${srcConn}::${schema}::${table}::${segment}`;
```
**Vị trí:** `cdc-cms-web/src/pages/DataIntegrity.tsx`
Khi mở modal kiểm tra / heal:
```ts
const row = action.record || reportList.find(r => r.target_table === action.table && (!action.record?.shadow_schema || r.shadow_schema === action.record.shadow_schema));
```
