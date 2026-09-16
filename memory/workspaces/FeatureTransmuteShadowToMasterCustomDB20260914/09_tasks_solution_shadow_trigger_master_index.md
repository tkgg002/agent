# 09_tasks_solution_shadow_trigger_master_index.md
# HỒ SƠ GIẢI PHÁP KỸ THUẬT VÀ DEMO CODE CHI TIẾT (FULL TECHNICAL SOLUTIONS & DEMO CODE)

---

## I. TỔNG QUAN GIẢI PHÁP
Giải pháp bao gồm 2 phần độc lập nhưng chặt chẽ về kiến trúc:
1. **Phần 1: Đảm bảo 100% Shadow Table sở hữu Trigger Sonyflake Fallback** (ngăn chặn vĩnh viễn `_gpay_id = NULL` khi drop/create table hoặc snapshot ingest).
2. **Phần 2: Chuẩn hóa Master Index & Recommendation Suite theo Multi-Connection** (đưa index recommendation lên Master Table, truyền đúng Target Connection qua 3 tầng: Frontend -> CMS API -> Worker Engine).

---

## II. PHẦN 1: TỰ ĐỘNG GẮN SONYFLAKE TRIGGER TRÊN SHADOW TABLE (CDS WORKER)

### 1. Phân tích phản biện & Yêu cầu kỹ thuật
- **Nguyên nhân gốc rễ**: Khi bấm nút `Create` trên UI (`TableRegistry.tsx`), NATS gọi `cdc.cmd.create-default-columns` -> `HandleCreateDefaultColumns` -> chỉ gọi `CreateEmptyTable`, `EnsureCDCColumnsInSchema`, `AddPrimaryKeyConstraint`. **Hoàn toàn không có bước gắn trigger Sonyflake**.
- **Tiêu chí Simplicity First & Minimal Impact**:
  - Không sinh ID trên Go worker buffer để tránh làm chậm ingestion pipeline.
  - Tái sử dụng cơ chế trigger `tg_sonyflake_fallback()` và sequence `fencing_token_seq` trong schema của Shadow.
  - **Kỷ luật chống nuốt lỗi (No Silent Failure)**: Tuyệt đối không dùng `_ = sa.EnsureSonyflakeTrigger(...)`. Bắt buộc kiểm tra `err != nil` và trả về lỗi nếu tạo trigger thất bại!

### 2. Demo Code Chi Tiết

#### File 1.1: `centralized-data-service/internal/service/shadow/schema_adapter.go`
Bổ sung hàm `EnsureSonyflakeTrigger` và gọi trong `EnsureCDCColumnsInSchema` + `createShadowTableV1WithCols`:

```go
// EnsureSonyflakeTrigger đảm bảo sequence, function và trigger sonyflake
// luôn tồn tại trên shadow table để tự động populate _gpay_id khi NULL hoặc 0.
// Hàm này idempotent: chạy nhiều lần không gây lỗi.
func (sa *SchemaAdapter) EnsureSonyflakeTrigger(ctx context.Context, schemaName, tableName string) error {
	if strings.TrimSpace(schemaName) == "" {
		schemaName = "public"
	}
	schemaIdent := sqlutil.QuoteIdent(schemaName)
	tableIdent := sqlutil.QuoteIdent(tableName)
	triggerName := sqlutil.QuoteIdent(fmt.Sprintf("trg_%s_sonyflake_fallback", tableName))

	stmts := []string{
		// 1. Tạo sequence fencing_token_seq trong schema nếu chưa có
		fmt.Sprintf(`CREATE SEQUENCE IF NOT EXISTS %s.fencing_token_seq`, schemaIdent),

		// 2. Tạo hoặc thay thế hàm sinh sonyflake id
		fmt.Sprintf(`CREATE OR REPLACE FUNCTION %s.gen_sonyflake_id()
RETURNS BIGINT AS $fn$
DECLARE
  v_ts_ms   BIGINT;
  v_machine INTEGER;
  v_seq     BIGINT;
BEGIN
  v_ts_ms := (EXTRACT(EPOCH FROM clock_timestamp()) * 1000)::BIGINT - 1767225600000;
  BEGIN
    v_machine := COALESCE(NULLIF(current_setting('cdc.machine_id', true), '')::INTEGER, 0) & 65535;
  EXCEPTION WHEN OTHERS THEN
    v_machine := 0;
  END;
  v_seq := nextval('%s.fencing_token_seq') & 65535;
  RETURN ((v_ts_ms & 4398046511103) << 22) | ((v_machine::BIGINT & 65535) << 6) | (v_seq & 63);
END;
$fn$ LANGUAGE plpgsql VOLATILE`, schemaIdent, schemaName),

		// 3. Tạo hoặc thay thế trigger function fallback
		fmt.Sprintf(`CREATE OR REPLACE FUNCTION %s.tg_sonyflake_fallback()
RETURNS TRIGGER AS $fn$
BEGIN
  IF NEW._gpay_id IS NULL OR NEW._gpay_id = 0 THEN
    NEW._gpay_id := %s.gen_sonyflake_id();
  END IF;
  RETURN NEW;
END;
$fn$ LANGUAGE plpgsql`, schemaIdent, schemaIdent),

		// 4. Drop trigger cũ nếu có để tránh xung đột định nghĩa
		fmt.Sprintf(`DROP TRIGGER IF EXISTS %s ON %s.%s`, triggerName, schemaIdent, tableIdent),

		// 5. Tạo trigger BEFORE INSERT gọi function fallback
		fmt.Sprintf(`CREATE TRIGGER %s BEFORE INSERT ON %s.%s
FOR EACH ROW EXECUTE FUNCTION %s.tg_sonyflake_fallback()`,
			triggerName, schemaIdent, tableIdent, schemaIdent),
	}

	for _, stmt := range stmts {
		if err := sa.db.WithContext(ctx).Exec(stmt).Error; err != nil {
			return fmt.Errorf("ensure sonyflake trigger on %s.%s: %w", schemaName, tableName, err)
		}
	}
	return nil
}
```

Và tại cuối hàm `EnsureCDCColumnsInSchema` (khoảng dòng 1025 của `schema_adapter.go`):
```go
	deletedIdxName := fmt.Sprintf("idx_%s_deleted_partial", tableName)
	if err := sa.db.WithContext(ctx).Exec(fmt.Sprintf(
		`CREATE INDEX IF NOT EXISTS %s ON %s.%s (_deleted) WHERE _deleted = true`,
		sqlutil.QuoteIdent(deletedIdxName),
		sqlutil.QuoteIdent(schemaName), sqlutil.QuoteIdent(tableName),
	)).Error; err != nil {
		return err
	}

	// [MỚI] Tự động gắn trigger Sonyflake khi bổ sung metadata columns — BẮT BUỘC bắt lỗi không nuốt lỗi!
	if err := sa.EnsureSonyflakeTrigger(ctx, schemaName, tableName); err != nil {
		return fmt.Errorf("ensure sonyflake trigger failed: %w", err)
	}

	return nil
```

Và tại hàm `createShadowTableV1WithCols` (khoảng dòng 315 của `schema_adapter.go`):
```go
	if err := sa.db.Exec(ddl).Error; err != nil {
		return fmt.Errorf("create table: %w", err)
	}

	// [MỚI] Gắn trigger cho bảng tự động tạo ở luồng CDC event V1
	if err := sa.EnsureSonyflakeTrigger(context.Background(), schema, table); err != nil {
		return fmt.Errorf("ensure sonyflake trigger v1: %w", err)
	}
	return nil
```

---

## III. PHẦN 2: MASTER INDEX & RECOMMENDATION MULTI-CONNECTION SUITE

### 1. Phân tích phản biện & Yêu cầu kỹ thuật
- **Nguyên nhân gốc rễ**:
  1. Frontend `TableIndexManager.tsx:127`: Điều kiện `if (plane === 'shadow')` bao trùm toàn bộ recommendations (kể cả `backendRecommendations`). Khi ở trang Master (`plane="master"`), mảng `recommendations` luôn rỗng `[]`.
  2. Frontend `TableIndexManager.tsx`: Không nhận và không truyền `connectionKey` khi gọi API `GET /api/introspection/indexes/:table`, `POST /api/introspection/indexes`, và `DELETE /api/introspection/indexes/:name`.
  3. CMS Backend `introspection_handler.go`: Không đọc query param `connection_key` và không đóng gói vào NATS payload `cdc.cmd.introspect-indexes`, `cdc.cmd.create-index`, `cdc.cmd.drop-index`.
  4. CDS Worker `index_handler.go:58, 116, 181`: Toàn bộ 3 handler hardcode `"default"`: `h.connMgr.GetMasterDB(ctx, "default")`. Nếu Master Table nằm trên custom connection (ví dụ `gpay-postgres-master-2`), Worker kết nối nhầm DB default -> không tìm thấy bảng -> trả về 0 index!
  5. CDS Worker `index_manager.go:171`: Tự động đề xuất tạo index trên `_source_id`. Bảng Master **không có cột `_source_id`** (Master dùng `_gpay_id` PK và OCC `_source_ts`). Đề xuất này gây lỗi `column "_source_id" does not exist` khi user bấm tạo! Master cần đề xuất `_updated_at` (cho delta audit), `_source_ts`, và `_deleted`.

---

### 2. Demo Code Chi Tiết

#### File 2.1: `cdc-cms-web/src/components/TableIndexManager.tsx`
Cập nhật interface props, nhận `connectionKey`, truyền vào các API calls và mở rộng logic đề xuất:

```tsx
interface TableIndexManagerProps {
  schema: string;
  table: string;
  plane: 'shadow' | 'master';
  availableColumns: string[];
  connectionKey?: string; // [MỚI] Target Connection Key cho Master
}

export default function TableIndexManager({ schema, table, plane, availableColumns, connectionKey }: TableIndexManagerProps) {
  const [indexes, setIndexes] = useState<IndexInfo[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [creating, setCreating] = useState(false);
  const [form] = Form.useForm();
  const [isPartial, setIsPartial] = useState(false);
  const [backendRecommendations, setBackendRecommendations] = useState<any[]>([]);

  const fetchIndexes = async () => {
    setLoading(true);
    try {
      const { data } = await cmsApi.get(`/api/introspection/indexes/${table}`, {
        params: { 
          schema, 
          plane,
          connection_key: connectionKey // [MỚI] Truyền connection_key
        }
      });
      setIndexes(data.indexes || []);
      setBackendRecommendations(data.recommendations || []);
    } catch (err) {
      message.error(humanizeApiError(err, 'Không thể tải danh sách index'));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchIndexes();
  }, [schema, table, plane, connectionKey]);

  const handleCreateIndex = async (values: any) => {
    setCreating(true);
    try {
      await cmsApi.post('/api/introspection/indexes', {
        schema,
        table,
        plane,
        connection_key: connectionKey, // [MỚI] Truyền connection_key
        columns: values.columns,
        is_unique: !!values.is_unique,
        is_partial: !!values.is_partial,
        where_clause: values.is_partial ? values.where_clause : '',
      });
      message.success('Index creation command submitted concurrently.');
      setModalVisible(false);
      form.resetFields();
      setIsPartial(false);
      setTimeout(fetchIndexes, 2000);
    } catch (err) {
      message.error(humanizeApiError(err, 'Tạo index thất bại'));
    } finally {
      setCreating(false);
    }
  };

  const handleDropIndex = (indexName: string) => {
    Modal.confirm({
      title: `Bạn có chắc chắn muốn xóa index ${indexName}?`,
      content: 'Thao tác DROP INDEX CONCURRENTLY sẽ được gửi xuống worker và thực thi an toàn.',
      okText: 'Xác nhận xóa',
      okType: 'danger',
      cancelText: 'Hủy',
      onOk: async () => {
        try {
          await cmsApi.delete(`/api/introspection/indexes/${indexName}`, {
            params: { 
              schema, 
              plane,
              connection_key: connectionKey // [MỚI] Truyền connection_key
            }
          });
          message.success('Index drop command submitted concurrently.');
          setTimeout(fetchIndexes, 2000);
        } catch (err) {
          message.error(humanizeApiError(err, 'Xóa index thất bại'));
        }
      }
    });
  };

  const handleCreateRecommended = async (rec: { cols: string[]; isPartial: boolean; where: string }) => {
    setLoading(true);
    try {
      await cmsApi.post('/api/introspection/indexes', {
        schema,
        table,
        plane,
        connection_key: connectionKey, // [MỚI] Truyền connection_key
        columns: rec.cols,
        is_unique: false,
        is_partial: rec.isPartial,
        where_clause: rec.where,
      });
      message.success('Index creation command submitted concurrently.');
      setTimeout(fetchIndexes, 2000);
    } catch (err) {
      message.error(humanizeApiError(err, 'Không thể tạo recommended index'));
    } finally {
      setLoading(false);
    }
  };

  // Recommendations logic — HỖ TRỢ CẢ SHADOW VÀ MASTER (Bỏ chặn if (plane === 'shadow'))
  const recommendations: RecommendationItem[] = [];
  const lowercaseDefs = indexes.map(idx => idx.index_def.toLowerCase());

  // 1. Partial index on _deleted = true (áp dụng cho cả shadow và master)
  if (availableColumns.includes('_deleted')) {
    const hasDeletedPartial = lowercaseDefs.some(
      def => def.includes('_deleted') && (def.includes('where _deleted = true') || def.includes('where (_deleted = true)'))
    );
    if (!hasDeletedPartial) {
      recommendations.push({
        title: 'Tối ưu CountDeletedRows (Partial Index on _deleted)',
        desc: 'Tạo partial index trên cột _deleted để tối ưu hóa truy vấn đối soát dòng đã xóa cho Recon.',
        cols: ['_deleted'],
        isPartial: true,
        where: '_deleted = true',
      });
    }
  }

  // 2. Index on _source_ts (áp dụng cho cả shadow và master)
  if (availableColumns.includes('_source_ts')) {
    const hasSourceTs = lowercaseDefs.some(def => def.includes('(_source_ts)'));
    if (!hasSourceTs) {
      recommendations.push({
        title: 'Tối ưu BucketCounts (Index on _source_ts)',
        desc: 'Tạo index trên _source_ts giúp tăng tốc độ phân tích thời gian theo bucket.',
        cols: ['_source_ts'],
        isPartial: false,
        where: '',
      });
    }
  }

  // 3. Index on _updated_at (chỉ áp dụng cho master table để tối ưu delta-scan)
  if (plane === 'master' && availableColumns.includes('_updated_at')) {
    const hasUpdatedAt = lowercaseDefs.some(def => def.includes('(_updated_at)'));
    if (!hasUpdatedAt) {
      recommendations.push({
        title: 'Tối ưu Delta Query (Index on _updated_at)',
        desc: 'Tạo index trên _updated_at để tối ưu hóa truy vấn audit và quét delta cho Master Table.',
        cols: ['_updated_at'],
        isPartial: false,
        where: '',
      });
    }
  }

  // 4. Index on _source_id (CHỈ áp dụng cho shadow table; master không có cột này)
  if (plane === 'shadow' && availableColumns.includes('_source_id')) {
    const hasSourceId = lowercaseDefs.some(def => def.includes('(_source_id)'));
    if (!hasSourceId) {
      recommendations.push({
        title: 'Tối ưu Query by ID (Index on _source_id)',
        desc: 'Tạo index trên _source_id để tối ưu hóa truy vấn lookup ID cho Shadow Table.',
        cols: ['_source_id'],
        isPartial: false,
        where: '',
      });
    }
  }

  // 5. Tích hợp backend recommendations cho CẢ SHADOW LẪN MASTER
  backendRecommendations.forEach(rec => {
    const alreadyRecommended = recommendations.some((r: RecommendationItem) => 
      r.cols.length === rec.columns.length && 
      r.cols.every((val: string, index: number) => val === rec.columns[index])
    );
    if (!alreadyRecommended) {
      recommendations.push({
        title: `Khuyến nghị: ${rec.index_name}`,
        desc: rec.description,
        cols: rec.columns,
        isPartial: !!rec.is_partial,
        where: rec.where_clause || '',
      });
    }
  });
```

---

#### File 2.2: `cdc-cms-web/src/pages/MasterMappingFieldsPage.tsx`
Cập nhật `interface MasterBinding` và truyền `connectionKey`:

```tsx
interface MasterBinding {
  id: number;
  binding_code: string;
  master_name: string;
  master_schema: string;
  master_connection_code?: string; // [MỚI] Nhận master_connection_code từ API /api/v1/masters
  shadow_schema?: string | null;
  shadow_table?: string | null;
  shadow_binding_id?: number | null;
  transform_type: string;
  is_active: boolean;
  schema_status: 'pending_review' | 'approved' | 'rejected' | 'failed';
}

// ...
// Tại dòng 930:
      {binding && (
        <TableIndexManager
          schema={binding.master_schema || 'public'}
          table={binding.master_name}
          plane="master"
          availableColumns={availableColumns}
          connectionKey={binding.master_connection_code} // [MỚI] Truyền đúng connectionKey
        />
      )}
```

---

#### File 2.3: `cdc-cms-service/internal/api/system/introspection_handler.go`
Nhận query/body param `connection_key` và đưa vào NATS payloads:

```go
// ListIndexes - GET /introspection/indexes/:table
func (h *IntrospectionHandler) ListIndexes(c *fiber.Ctx) error {
	table := c.Params("table")
	schema := c.Query("schema", "public")
	plane := c.Query("plane", "shadow")
	connectionKey := c.Query("connection_key") // [MỚI]

	correlationID := fmt.Sprintf("list-indexes-%s-%d", table, time.Now().UnixNano())
	replySubj := "cdc.evt.introspect-indexes." + correlationID

	payload, _ := json.Marshal(map[string]interface{}{
		"schema":         schema,
		"table":          table,
		"plane":          plane,
		"connection_key": connectionKey, // [MỚI]
		"reply_to":       replySubj,
	})

	msgData, err := h.rpc.Request(c.UserContext(), "cdc.cmd.introspect-indexes", replySubj, payload, 10*time.Second)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Worker timeout or request error: " + err.Error(),
		})
	}

	var res map[string]interface{}
	if err := json.Unmarshal(msgData, &res); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to parse worker response"})
	}
	return c.JSON(res)
}

// CreateIndex - POST /introspection/indexes
func (h *IntrospectionHandler) CreateIndex(c *fiber.Ctx) error {
	var body struct {
		Schema        string   `json:"schema"`
		Table         string   `json:"table"`
		Columns       []string `json:"columns"`
		Plane         string   `json:"plane"`
		ConnectionKey string   `json:"connection_key"` // [MỚI]
		IsUnique      bool     `json:"is_unique"`
		IsPartial     bool     `json:"is_partial"`
		WhereClause   string   `json:"where_clause"`
	}

	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body: " + err.Error()})
	}
	if body.Table == "" || len(body.Columns) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "table and columns are required"})
	}
	if body.Schema == "" {
		body.Schema = "public"
	}
	if body.Plane == "" {
		body.Plane = "shadow"
	}

	correlationID := fmt.Sprintf("create-index-%s-%d", body.Table, time.Now().UnixNano())
	replySubj := "cdc.evt.create-index." + correlationID

	payload, _ := json.Marshal(map[string]interface{}{
		"schema":         body.Schema,
		"table":          body.Table,
		"columns":        body.Columns,
		"plane":          body.Plane,
		"connection_key": body.ConnectionKey, // [MỚI]
		"is_unique":      body.IsUnique,
		"is_partial":     body.IsPartial,
		"where_clause":   body.WhereClause,
		"reply_to":       replySubj,
	})

	msgData, err := h.rpc.Request(c.UserContext(), "cdc.cmd.create-index", replySubj, payload, 30*time.Second)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Worker timeout or request error: " + err.Error(),
		})
	}

	var res map[string]interface{}
	if err := json.Unmarshal(msgData, &res); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to parse worker response"})
	}
	return c.JSON(res)
}

// DropIndex - DELETE /introspection/indexes/:name
func (h *IntrospectionHandler) DropIndex(c *fiber.Ctx) error {
	name := c.Params("name")
	schema := c.Query("schema", "public")
	plane := c.Query("plane", "shadow")
	connectionKey := c.Query("connection_key") // [MỚI]

	correlationID := fmt.Sprintf("drop-index-%s-%d", name, time.Now().UnixNano())
	replySubj := "cdc.evt.drop-index." + correlationID

	payload, _ := json.Marshal(map[string]interface{}{
		"schema":         schema,
		"index_name":     name,
		"plane":          plane,
		"connection_key": connectionKey, // [MỚI]
		"reply_to":       replySubj,
	})

	msgData, err := h.rpc.Request(c.UserContext(), "cdc.cmd.drop-index", replySubj, payload, 30*time.Second)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Worker timeout or request error: " + err.Error(),
		})
	}

	var res map[string]interface{}
	if err := json.Unmarshal(msgData, &res); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to parse worker response"})
	}
	return c.JSON(res)
}
```

---

#### File 2.4: `centralized-data-service/internal/handler/governance/index_handler.go`
Nhận `connection_key`, phân giải đúng target database và truyền `plane` vào `GetRecommendations`:

```go
// resolveDB phân giải đúng instance GORM DB cho shadow hoặc master theo connectionKey
func (h *IndexHandler) resolveDB(ctx context.Context, plane, schema, table, connKey string) (*gorm.DB, error) {
	if strings.ToLower(plane) == "master" {
		key := strings.TrimSpace(connKey)
		if key == "" || key == "default" {
			// Fallback: Tra cứu master_connection_id từ cdc_system.master_binding nếu client chưa truyền
			var mb struct {
				ConnectionCode string `gorm:"column:connection_code"`
			}
			systemDB, _ := h.connMgr.GetSystemDB(ctx)
			if systemDB != nil {
				_ = systemDB.Table("cdc_system.master_binding mb").
					Select("COALESCE(cr.connection_code, 'default') AS connection_code").
					Joins("LEFT JOIN cdc_system.connection_registry cr ON cr.id = mb.master_connection_id").
					Where("(mb.master_table = ? OR mb.master_schema || '.' || mb.master_table = ?) AND mb.is_active = true", table, schema+"."+table).
					Order("mb.id DESC").
					Limit(1).Scan(&mb).Error
				if mb.ConnectionCode != "" {
					key = mb.ConnectionCode
				}
			}
		}
		if key == "" {
			key = "default"
		}
		return h.connMgr.GetMasterDB(ctx, key)
	}
	// Với shadow plane
	return h.connMgr.GetShadowDB(ctx, connKey)
}

func (h *IndexHandler) HandleIntrospectIndexes(msg *nats.Msg) {
	var payload struct {
		Schema        string `json:"schema"`
		Table         string `json:"table"`
		Plane         string `json:"plane"` // "shadow" or "master"
		ConnectionKey string `json:"connection_key"` // [MỚI]
	}
	errPayload := json.Unmarshal(msg.Data, &payload)
	// ... logging & span ...

	targetDB, err := h.resolveDB(ctx, payload.Plane, payload.Schema, payload.Table, payload.ConnectionKey)
	if err != nil {
		h.PublishResult(ctx, msg, base.CommandResult{Command: "introspect-indexes", Status: "error", Error: "failed to get database connection: " + err.Error()})
		return
	}

	indexes, err := h.indexManager.ListIndexes(ctx, targetDB, payload.Schema, payload.Table)
	if err != nil {
		h.PublishResult(ctx, msg, base.CommandResult{Command: "introspect-indexes", Status: "error", Error: err.Error()})
		return
	}

	// [MỚI] Tính toán đề xuất index, truyền payload.Plane để phân biệt shadow vs master
	systemDB, _ := h.connMgr.GetSystemDB(ctx)
	recommendations := h.indexManager.GetRecommendations(ctx, targetDB, systemDB, payload.Schema, payload.Table, payload.Plane, indexes)

	respData, _ := json.Marshal(map[string]interface{}{
		"command":         "introspect-indexes",
		"status":          "success",
		"indexes":         indexes,
		"recommendations": recommendations,
	})
	h.NatsPublish(msg, "cdc.result.introspect-indexes", respData)
}

func (h *IndexHandler) HandleCreateIndex(msg *nats.Msg) {
	var payload struct {
		Schema        string   `json:"schema"`
		Table         string   `json:"table"`
		Columns       []string `json:"columns"`
		Plane         string   `json:"plane"`
		ConnectionKey string   `json:"connection_key"` // [MỚI]
		IsUnique      bool     `json:"is_unique"`
		IsPartial     bool     `json:"is_partial"`
		WhereClause   string   `json:"where_clause"`
	}
	errPayload := json.Unmarshal(msg.Data, &payload)
	// ...

	targetDB, err := h.resolveDB(ctx, payload.Plane, payload.Schema, payload.Table, payload.ConnectionKey)
	if err != nil {
		span.End()
		h.PublishResult(ctx, msg, base.CommandResult{Command: "create-index", Status: "error", Error: "failed to get db: " + err.Error()})
		return
	}
	// ... chạy CreateIndexConcurrently trên targetDB ...
}

func (h *IndexHandler) HandleDropIndex(msg *nats.Msg) {
	var payload struct {
		Schema        string `json:"schema"`
		IndexName     string `json:"index_name"`
		Plane         string `json:"plane"`
		ConnectionKey string `json:"connection_key"` // [MỚI]
	}
	errPayload := json.Unmarshal(msg.Data, &payload)
	// ...

	targetDB, err := h.resolveDB(ctx, payload.Plane, payload.Schema, "", payload.ConnectionKey)
	if err != nil {
		span.End()
		h.PublishResult(ctx, msg, base.CommandResult{Command: "drop-index", Status: "error", Error: "failed to get db: " + err.Error()})
		return
	}
	// ... chạy DropIndexConcurrently trên targetDB ...
}
```

---

#### File 2.5: `centralized-data-service/internal/service/governance/index_manager.go`
Nhận `plane string` trong `GetRecommendations`, loại bỏ đề xuất `_source_id` cho Master, bổ sung `_updated_at`:

```go
func (im *IndexManager) GetRecommendations(ctx context.Context, db *gorm.DB, systemDB *gorm.DB, schema, table, plane string, indexes []IndexInfo) []IndexRecommendation {
	var recs []IndexRecommendation
	isMaster := strings.ToLower(plane) == "master"

	// 1. Kiểm tra index trên _source_id (CHỈ cho SHADOW, master không có cột này)
	if !isMaster {
		sourceIdIndexName := fmt.Sprintf("idx_%s_source_id", table)
		hasSourceIdIndex := false
		for _, idx := range indexes {
			if (idx.IndexName == sourceIdIndexName || strings.Contains(idx.IndexName, "source_id") || strings.Contains(idx.IndexName, "_source_id")) && idx.IsValid {
				hasSourceIdIndex = true
				break
			}
		}

		if !hasSourceIdIndex {
			recs = append(recs, IndexRecommendation{
				IndexName:   sourceIdIndexName,
				Columns:     []string{"_source_id"},
				IsUnique:    false,
				IsPartial:   false,
				Description: "Tối ưu hóa Transmuter: Tạo index trên cột _source_id để tối ưu hóa truy vấn lookup/upsert và transmute cho Shadow.",
			})
		}
	}

	// 2. Kiểm tra partial index trên _deleted = true (cho cả shadow và master)
	deletedIndexName := fmt.Sprintf("idx_%s_deleted_partial", table)
	hasDeletedIndex := false
	for _, idx := range indexes {
		if (idx.IndexName == deletedIndexName || strings.Contains(idx.IndexName, "deleted") || strings.Contains(idx.IndexName, "_deleted")) && idx.IsValid {
			hasDeletedIndex = true
			break
		}
	}

	if !hasDeletedIndex {
		recs = append(recs, IndexRecommendation{
			IndexName:   deletedIndexName,
			Columns:     []string{"_deleted"},
			IsUnique:    false,
			IsPartial:   true,
			WhereClause: "_deleted = true",
			Description: "Tối ưu CountDeletedRows: Tạo partial index trên cột _deleted để tối ưu hóa truy vấn đối soát dòng đã xóa cho Recon.",
		})
	}

	// 3. Nếu là Master: Đề xuất index trên _updated_at nếu chưa có
	if isMaster {
		updatedAtIndexName := fmt.Sprintf("idx_%s_updated_at", table)
		hasUpdatedAtIndex := false
		for _, idx := range indexes {
			if (idx.IndexName == updatedAtIndexName || strings.Contains(idx.IndexName, "updated_at") || strings.Contains(idx.IndexName, "_updated_at")) && idx.IsValid {
				hasUpdatedAtIndex = true
				break
			}
		}
		if !hasUpdatedAtIndex {
			recs = append(recs, IndexRecommendation{
				IndexName:   updatedAtIndexName,
				Columns:     []string{"_updated_at"},
				IsUnique:    false,
				IsPartial:   false,
				Description: "Tối ưu hóa Delta Scan: Tạo index trên cột _updated_at để tối ưu hóa truy vấn audit và delta cho Master Table.",
			})
		}
	}

	// 4. Kiểm tra index trên Timestamp Field nghiệp vụ (giữ nguyên logic tra cứu)
	// ... (logic kiểm tra actualCols trên targetDB)
	return recs
}
```
