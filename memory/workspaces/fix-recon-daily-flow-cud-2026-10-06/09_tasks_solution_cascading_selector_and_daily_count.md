# Hồ Sơ Giải Pháp Kỹ Thuật: Cascading Selector & Daily Source Count Đa Hình (Postgres, Mongo, MySQL, SFTP, ClickHouse)

- **Ngày cập nhật:** 2026-10-12
- **Tác giả:** Brain (Chairman & Architect)
- **Mục tiêu:** Cung cấp thiết kế kỹ thuật chi tiết hỗ trợ toàn diện các loại nguồn: PostgreSQL, MongoDB, MySQL, SFTP, ClickHouse, Kafka...

---

## I. DDL MIGRATION TRÊN CƠ SỞ DỮ LIỆU (`cdc_dw`)

### 1. Bổ sung trường cho bảng `cdc_system.cdc_worker_schedule`
```sql
-- Migration 081_enhance_worker_schedule_for_dynamic_jobs.sql
BEGIN;

ALTER TABLE cdc_system.cdc_worker_schedule
  ADD COLUMN IF NOT EXISTS run_at_time VARCHAR(8) NULL,               -- Định dạng 'HH:mm', vd: '00:05'
  ADD COLUMN IF NOT EXISTS params JSONB NOT NULL DEFAULT '{}'::jsonb; -- Lưu chi tiết cấu hình Cascading Selector đa hình

COMMENT ON COLUMN cdc_system.cdc_worker_schedule.run_at_time IS 
  'Mốc giờ cố định chạy hàng ngày (HH:mm). Nếu có giá trị, scheduler sẽ căn chuẩn thời gian này thay vì lặp theo interval.';

COMMENT ON COLUMN cdc_system.cdc_worker_schedule.params IS 
  'Metadata tham số động: layer_type, connection_code, engine_type, container_namespace, object_name, object_type, date_strategy, date_field, window_mode, timezone.';

COMMIT;
```

### 2. Tạo bảng lưu trữ kết quả đếm đa hình `cdc_system.daily_source_counts`
```sql
-- Migration 082_create_polymorphic_daily_source_counts.sql
BEGIN;

CREATE TABLE IF NOT EXISTS cdc_system.daily_source_counts (
    id                      BIGSERIAL PRIMARY KEY,
    schedule_id             INT REFERENCES cdc_system.cdc_worker_schedule(id) ON DELETE SET NULL,
    layer_type              VARCHAR(32) NOT NULL,               -- 'source', 'shadow', 'master'
    connection_code         VARCHAR(128) NOT NULL,              -- Mã kết nối từ connection_registry
    engine_type             VARCHAR(32) NOT NULL,               -- 'postgresql', 'mongodb', 'mysql', 'mariadb', 'sftp', 'clickhouse', 'kafka', 'file'
    container_namespace     VARCHAR(255) NOT NULL DEFAULT '',   -- Postgres: schema | MySQL/Mongo: database | SFTP: remote directory path
    object_name             VARCHAR(255) NOT NULL,              -- SQL: table | Mongo: collection | SFTP: filename pattern | Kafka: topic
    object_type             VARCHAR(32) NOT NULL,               -- 'table', 'collection', 'file', 'file_batch', 'topic', 'view'
    date_strategy           VARCHAR(64) NOT NULL DEFAULT 'column', -- 'column', 'filename_regex', 'file_mtime', 'message_timestamp'
    date_field              VARCHAR(128) NOT NULL,              -- CSDL: cột ('createdAt', 'created_at') | SFTP: pattern/cột
    report_date             DATE NOT NULL,                      -- Ngày đối soát (vd: 2026-10-11)
    creation_count          BIGINT NOT NULL DEFAULT 0,          -- Số lượng bản ghi / số dòng sinh ra trong ngày
    calculated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(), -- Thời điểm Worker thực thi lệnh đếm
    execution_duration_ms   INT NOT NULL DEFAULT 0,             -- Thời gian thực thi (ms)
    metadata_json           JSONB DEFAULT '{}'::jsonb,          -- Chi tiết: danh sách file SFTP, query SQL, line count breakdown, checksum
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (layer_type, connection_code, container_namespace, object_name, report_date)
);

CREATE INDEX IF NOT EXISTS idx_daily_source_counts_lookup 
  ON cdc_system.daily_source_counts(layer_type, container_namespace, object_name, report_date);

CREATE INDEX IF NOT EXISTS idx_daily_source_counts_engine 
  ON cdc_system.daily_source_counts(engine_type, object_type);

CREATE INDEX IF NOT EXISTS idx_daily_source_counts_report_date 
  ON cdc_system.daily_source_counts(report_date);

COMMIT;
```

---

## II. GIAO DIỆN CMS WEB (`cdc-cms-web/src/pages/ActivityManager.tsx`)

### Giao diện Form Modal Cascading thích ứng đa hình theo `engine_type`:
```tsx
{/* BƯỚC 1: CHỌN LAYER TYPE */}
<Form.Item name="layer_type" label="Tầng dữ liệu (Layer Type)" rules={[{ required: true }]}>
  <Select placeholder="Chọn tầng dữ liệu" onChange={handleLayerTypeChange}>
    <Select.Option value="source">1. Source (Cơ sở dữ liệu / Nguồn SFTP / Kafka)</Select.Option>
    <Select.Option value="shadow">2. Shadow (Cơ sở dữ liệu vùng đệm Staging CDC)</Select.Option>
    <Select.Option value="master">3. Master (Kho dữ liệu tập trung Data Warehouse)</Select.Option>
  </Select>
</Form.Item>

{/* BƯỚC 2: CHỌN CONNECTION */}
<Form.Item name="connection_code" label="Kết nối (Connection / DB Connect)" rules={[{ required: true }]}>
  <Select placeholder="Chọn kết nối" onChange={handleConnectionChange}>
    {filteredConnections.map(conn => (
      <Select.Option key={conn.connection_code} value={conn.connection_code}>
        <Space>
          <Tag color={getEngineColor(conn.engine_type)}>{conn.engine_type.toUpperCase()}</Tag>
          <strong>{conn.display_name}</strong>
          <Text type="secondary" style={{ fontSize: 11 }}>({conn.connection_code})</Text>
        </Space>
      </Select.Option>
    ))}
  </Select>
</Form.Item>

{/* BƯỚC 3 & 4 & 5: DYNAMIC FIELDS THEO ENGINE TYPE */}
{selectedEngine === 'sftp' || selectedEngine === 'file' ? (
  <>
    <Form.Item name="container_namespace" label="Đường dẫn thư mục SFTP (Directory Path)" rules={[{ required: true }]} initialValue="/reconciliation/inbox/">
      <Input placeholder="/reconciliation/inbox/vcb/" />
    </Form.Item>
    <Form.Item name="object_name" label="Mẫu tên file / Pattern" rules={[{ required: true }]}>
      <Input placeholder="clearing_vcb_*.csv" />
    </Form.Item>
    <Form.Item name="object_type" label="Loại đối tượng" initialValue="file_batch">
      <Select>
        <Select.Option value="file_batch">File Batch (Quét nhiều file khớp pattern)</Select.Option>
        <Select.Option value="file">Đơn file cố định</Select.Option>
      </Select>
    </Form.Item>
    <Form.Item name="date_strategy" label="Chiến lược nhận diện ngày" initialValue="filename_regex">
      <Select>
        <Select.Option value="filename_regex">Trích xuất ngày từ tên file (Regex)</Select.Option>
        <Select.Option value="file_mtime">Thời điểm sửa đổi file (mtime)</Select.Option>
        <Select.Option value="column">Cột ngày bên trong file CSV/Excel</Select.Option>
      </Select>
    </Form.Item>
    <Form.Item name="date_field" label="Cấu hình ngày (Regex hoặc Tên cột)">
      <Input placeholder="clearing_vcb_(\d{8})\.csv hoặc TransactionDate" />
    </Form.Item>
  </>
) : selectedEngine === 'mongodb' ? (
  <>
    <Form.Item name="container_namespace" label="Database (MongoDB)" rules={[{ required: true }]}>
      <Select placeholder="Chọn Database">{availableDatabases.map(db => <Select.Option key={db} value={db}>{db}</Select.Option>)}</Select>
    </Form.Item>
    <Form.Item name="object_name" label="Collection (MongoDB)" rules={[{ required: true }]}>
      <Select placeholder="Chọn Collection">{availableCollections.map(c => <Select.Option key={c} value={c}>{c}</Select.Option>)}</Select>
    </Form.Item>
    <Form.Item name="object_type" hidden initialValue="collection"><Input /></Form.Item>
    <Form.Item name="date_field" label="Trường ngày tạo" rules={[{ required: true }]} initialValue="createdAt">
      <Select mode="combobox" options={[{ value: 'createdAt' }, { value: 'create_time' }, { value: '_source_ts' }]} />
    </Form.Item>
  </>
) : (
  <>
    <Form.Item name="container_namespace" label={selectedEngine === 'postgresql' ? "Schema (PostgreSQL)" : "Database (MySQL)"} rules={[{ required: true }]}>
      <Select placeholder="Chọn Schema/Database">{availableSchemas.map(s => <Select.Option key={s} value={s}>{s}</Select.Option>)}</Select>
    </Form.Item>
    <Form.Item name="object_name" label="Bảng (Table)" rules={[{ required: true }]}>
      <Select placeholder="Chọn Table">{availableTables.map(t => <Select.Option key={t} value={t}>{t}</Select.Option>)}</Select>
    </Form.Item>
    <Form.Item name="object_type" hidden initialValue="table"><Input /></Form.Item>
    <Form.Item name="date_field" label="Trường ngày tạo" rules={[{ required: true }]} initialValue="created_at">
      <Select mode="combobox" options={[{ value: 'created_at' }, { value: 'create_time' }, { value: 'createdAt' }, { value: '_source_ts' }]} />
    </Form.Item>
  </>
)}
```

---

## III. WORKER RUNNER ĐA HÌNH TRONG `centralized-data-service`

Runner `source-count` định tuyến theo `engine_type`:

```go
func (r *SourceCountRunner) ExecuteCount(ctx context.Context, params SourceCountParams, reportDate time.Time) (int64, map[string]interface{}, error) {
	switch params.EngineType {
	case "sftp":
		return r.countSFTPFiles(ctx, params, reportDate)
	case "mongodb":
		return r.countMongoDocuments(ctx, params, reportDate)
	case "postgresql", "mysql", "mariadb", "clickhouse":
		return r.countSQLRecords(ctx, params, reportDate)
	case "kafka":
		return r.countKafkaMessages(ctx, params, reportDate)
	default:
		return 0, nil, fmt.Errorf("unsupported engine type: %s", params.EngineType)
	}
}
```

---

## IV. TÁC VỤ CONFLUENCE DAILY TRÊN ACTIVITY MANAGER & WORKER RUNNER

### 1. Form Modal trên `ActivityManager.tsx` khi `operation === 'confluence'`:
- **Chọn Cấu hình Confluence đích:** Lấy từ `cdc_system.confluence_report_configs`.
- **Chọn Tables / Pipelines (Multi-Select):** Cho phép tick chọn nhiều tables cùng một lúc.
- **Giờ chạy cố định:** Mặc định `06:00` sáng (sau khi job `source-count` đã chốt sổ lúc `00:05`).

### 2. Handler thực thi tác vụ `confluence` trong `centralized-data-service`:
```go
s.jobs["confluence"] = func(ctx context.Context, now time.Time, sched mastermodel.WorkerSchedule) {
	var params ConfluenceJobParams
	if err := json.Unmarshal(sched.Params, &params); err != nil {
		s.logger.Error("failed to parse confluence job params", zap.Error(err))
		return
	}

	// 1. Tải thông tin Confluence Profile đích
	var config ConfluenceReportConfig
	if err := s.db.First(&config, params.ConfluenceConfigID).Error; err != nil {
		s.logger.Error("confluence config not found", zap.Int64("config_id", params.ConfluenceConfigID))
		return
	}

	tz, _ := time.LoadLocation("Asia/Ho_Chi_Minh")
	yesterday := now.In(tz).AddDate(0, 0, -1)
	reportDateStr := yesterday.Format("2006-01-02")

	// 2. Thu thập dữ liệu cho từng table được chọn
	type TableReportRow struct {
		DisplayName   string
		EngineType    string
		SourceCount   *int64 // Lấy từ cdc_system.daily_source_counts
		KafkaIngest   int64  // Lấy từ recon_hourly_pipeline_stats
		ShadowInsert  int64
		ShadowUpdate  int64
		MasterUpsert  int64
		Status        string
	}

	var rows []TableReportRow
	for _, tbl := range params.SelectedTables {
		row := TableReportRow{
			DisplayName: tbl.DisplayName,
			EngineType:  tbl.EngineType,
		}

		// A. Đọc từ cdc_system.daily_source_counts (nếu có)
		var dailyCount int64
		err := s.db.Table("cdc_system.daily_source_counts").
			Select("creation_count").
			Where("layer_type = ? AND connection_code = ? AND container_namespace = ? AND object_name = ? AND report_date = ?",
				tbl.LayerType, tbl.ConnectionCode, tbl.ContainerNamespace, tbl.ObjectName, reportDateStr).
			Scan(&dailyCount).Error
		if err == nil {
			row.SourceCount = &dailyCount
		}

		// B. Đọc lưu lượng CDC trong ngày từ recon_hourly_pipeline_stats
		s.db.Table("cdc_system.recon_hourly_pipeline_stats").
			Select("COALESCE(SUM(kafka_total), 0) as kafka_ingest, COALESCE(SUM(shadow_insert), 0) as shadow_insert, COALESCE(SUM(shadow_update), 0) as shadow_update, COALESCE(SUM(master_upsert), 0) as master_upsert").
			Where("target_table = ? AND DATE(window_hour AT TIME ZONE 'Asia/Ho_Chi_Minh') = ?", tbl.ObjectName, reportDateStr).
			Row().Scan(&row.KafkaIngest, &row.ShadowInsert, &row.ShadowUpdate, &row.MasterUpsert)

		// C. Đánh giá trạng thái khớp
		if row.SourceCount != nil && *row.SourceCount == row.MasterUpsert {
			row.Status = "MATCH (Khớp 100%)"
		} else if row.SourceCount == nil {
			row.Status = "CDC OK (Source N/A)"
		} else {
			row.Status = fmt.Sprintf("DRIFT (Lệch %d)", *row.SourceCount - row.MasterUpsert)
		}

		rows = append(rows, row)
	}

	// 3. Render XHTML Table theo Confluence Storage Format
	pageBodyXHTML := buildConfluenceStorageFormat(reportDateStr, rows)
	pageTitle := fmt.Sprintf("[Daily Data Check] %s - %s", reportDateStr, config.DisplayTitle)

	// 4. Gọi Atlassian Confluence REST API v1/v2 tạo Child Page
	pageURL, err := s.confluenceClient.CreateChildPage(ctx, ConfluenceCreatePageRequest{
		BaseURL:      config.BaseURL,
		SpaceKey:     config.SpaceKey,
		ParentPageID: config.ParentPageID, // 463833042 (Core Trans Proxy History Service)
		UserEmail:    config.UserEmail,
		APIToken:     config.APIToken,
		Title:        pageTitle,
		BodyXHTML:    pageBodyXHTML,
	})

	if err != nil {
		s.logger.Error("failed to create confluence page", zap.Error(err))
		s.db.Model(&config).Updates(map[string]interface{}{
			"last_run_at": now,
			"last_status": "FAILED",
			"last_error":  err.Error(),
		})
		return
	}

	s.logger.Info("confluence report page created successfully", zap.String("url", pageURL))
	s.db.Model(&config).Updates(map[string]interface{}{
		"last_run_at":   now,
		"last_status":   "SUCCESS",
		"last_page_url": pageURL,
		"last_error":    nil,
	})
}
```
