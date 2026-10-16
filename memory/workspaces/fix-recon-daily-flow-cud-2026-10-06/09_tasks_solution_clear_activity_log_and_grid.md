# Hồ sơ Giải pháp Kỹ thuật: Tính năng Clear Activity Log & Tái cấu trúc Bảng Đối soát (V3 — Multi-Connector Safe)

## 1. Backend CMS Service (`cdc-cms-service`) — Cách ly Triệt để Multi-Connector

### 1.1. Persistence Layer (`internal/infra/persistence/system/activity_log_read_repo_gorm.go`)

Để triệt tiêu hoàn toàn nguy cơ xóa nhầm hoặc lẫn lộn log khi **nhiều connector cùng sync bảng trùng tên** (ví dụ: 2 connector cùng lấy bảng `payment_bills` vào 2 shadow schema khác nhau), hàm `ClearActivity` BẮT BUỘC phải nhận và lọc theo bộ định danh đầy đủ của pipeline:

```go
// ClearActivityParams defines the filter parameters for deleting activity logs safely.
type ClearActivityParams struct {
	ShadowSchema   string // Schema shadow của connector này (ví dụ: "shadow_traitestpbs")
	ShadowTable    string // Tên bảng shadow (ví dụ: "payment_bills")
	SourceDatabase string // Database nguồn (ví dụ: "payment-bill-service")
	MasterTable    string // Master table (ví dụ: "master_payment_bill_service.payment_bills")
	BeforeDate     string // created_at & started_at < beforeDate
	FromDate       string // created_at & started_at >= fromDate
	ToDate         string // created_at & started_at <= toDate
}

// ClearActivity deletes rows from cdc_system.cdc_activity_log with complete pipeline isolation.
func (r *ActivityLogReadRepo) ClearActivity(ctx context.Context, p ClearActivityParams) (int64, error) {
	// Guard Clause: Bắt buộc phải có ít nhất 1 điều kiện lọc để ngăn chặn xóa nhầm toàn hệ thống
	if p.ShadowTable == "" && p.MasterTable == "" && p.BeforeDate == "" && p.FromDate == "" {
		return 0, fmt.Errorf("at least one filter (shadow_table, master_table, before_date, from_date) is strictly required to prevent accidental full table wipe")
	}

	// Xây dựng subquery tìm chính xác các id cần xóa dựa trên lateral join giải mã pipeline
	// Đảm bảo không bao giờ xóa nhầm log của connector khác có cùng tên table!
	idQuery := `
		SELECT al.id 
		FROM cdc_system.cdc_activity_log al
		LEFT JOIN LATERAL (
			SELECT sb.id AS shadow_binding_id, sb.source_object_id, sb.shadow_schema, sb.shadow_table
			FROM cdc_system.shadow_binding sb
			WHERE al.target_table IS NOT NULL AND al.target_table <> '*' 
			  AND sb.shadow_table = al.target_table AND sb.is_active = TRUE
			ORDER BY sb.updated_at DESC, sb.id DESC
			LIMIT 1
		) sb ON TRUE
		LEFT JOIN LATERAL (
			SELECT mb2.id AS tm_mb_id, mb2.master_schema, mb2.master_table, mb2.shadow_binding_id AS tm_sb_id
			FROM cdc_system.master_binding mb2
			WHERE al.operation IN ('transmute', 'transmute_sub')
			  AND al.target_table IS NOT NULL AND al.target_table <> '*'
			  AND ((mb2.master_schema || '.' || mb2.master_table) = al.target_table OR mb2.master_table = al.target_table)
			  AND mb2.is_active = TRUE
			ORDER BY mb2.updated_at DESC, mb2.id DESC
			LIMIT 1
		) tm ON TRUE
		LEFT JOIN cdc_system.shadow_binding tm_sb ON tm_sb.id = tm.tm_sb_id
		LEFT JOIN cdc_system.source_object_registry so ON so.id = sb.source_object_id
		LEFT JOIN cdc_system.source_object_registry tm_so ON tm_so.id = tm_sb.source_object_id
		WHERE 1=1
	`
	var args []interface{}

	// 1. Phân lập chính xác Pipeline theo ShadowSchema & ShadowTable (Chống trùng tên đa connector)
	if p.ShadowSchema != "" {
		idQuery += ` AND (COALESCE(tm_sb.shadow_schema, sb.shadow_schema) = ? OR al.target_table = ?)`
		args = append(args, p.ShadowSchema, p.MasterTable)
	}
	if p.ShadowTable != "" {
		idQuery += ` AND (COALESCE(tm_sb.shadow_table, sb.shadow_table) = ? OR al.target_table = ?)`
		args = append(args, p.ShadowTable, p.MasterTable)
	}
	if p.SourceDatabase != "" {
		idQuery += ` AND COALESCE(tm_so.source_database, so.source_database) = ?`
		args = append(args, p.SourceDatabase)
	}

	// 2. Lọc thời gian kết hợp created_at & started_at để kích hoạt PostgreSQL Partition Pruning
	if p.BeforeDate != "" {
		idQuery += ` AND al.created_at < ? AND al.started_at < ?`
		args = append(args, p.BeforeDate, p.BeforeDate)
	}
	if p.FromDate != "" && p.ToDate != "" {
		idQuery += ` AND al.created_at >= ? AND al.created_at <= ? AND al.started_at >= ? AND al.started_at <= ?`
		args = append(args, p.FromDate, p.ToDate, p.FromDate, p.ToDate)
	} else if p.FromDate != "" {
		idQuery += ` AND al.created_at >= ? AND al.started_at >= ?`
		args = append(args, p.FromDate, p.FromDate)
	}

	// Thực hiện xóa an toàn
	deleteSQL := `DELETE FROM cdc_system.cdc_activity_log WHERE id IN (` + idQuery + `)`
	res := r.db.WithContext(ctx).Exec(deleteSQL, args...)
	return res.RowsAffected, res.Error
}
```

---

### 1.2. HTTP Handler (`internal/api/system/activity_log_handler.go`)

```go
// Clear godoc
// @Summary      Clear activity logs
// @Description  Deletes activity logs with full pipeline scope isolation (shadow_schema, shadow_table, source_database, master_table).
// @Tags         Activity Log
// @Produce      json
// @Param        shadow_schema query string false "Shadow schema of pipeline"
// @Param        shadow_table query string false "Shadow table name"
// @Param        source_database query string false "Source database"
// @Param        master_table query string false "Master table name"
// @Param        before_date query string false "Delete logs started before this date (YYYY-MM-DD)"
// @Param        from_date query string false "Start date (YYYY-MM-DD)"
// @Param        to_date query string false "End date (YYYY-MM-DD)"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/activity-log [delete]
func (h *ActivityLogHandler) Clear(c *fiber.Ctx) error {
	params := persistence.ClearActivityParams{
		ShadowSchema:   strings.TrimSpace(c.Query("shadow_schema")),
		ShadowTable:    strings.TrimSpace(c.Query("shadow_table")),
		SourceDatabase: strings.TrimSpace(c.Query("source_database")),
		MasterTable:    strings.TrimSpace(c.Query("master_table")),
		BeforeDate:     strings.TrimSpace(c.Query("before_date")),
		FromDate:       strings.TrimSpace(c.Query("from_date")),
		ToDate:         strings.TrimSpace(c.Query("to_date")),
	}

	if params.ShadowTable == "" && params.MasterTable == "" && params.BeforeDate == "" && params.FromDate == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "At least one filter (shadow_table, master_table, before_date, from_date) is required",
		})
	}

	rows, err := h.reader.ClearActivity(c.UserContext(), params)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{
		"ok":           true,
		"rows_deleted": rows,
	})
}
```

---

## 2. Frontend CMS Web (`cdc-cms-web`)

### 2.1. Nâng cấp `usePipelineActivityLog` (`src/hooks/useReconStatus.ts`)

Bổ sung param `shadowSchema` để đảm bảo khi đọc log trên UI, log của 2 connector cùng tên table **không bao giờ bị lẫn vào nhau**:

```typescript
export function usePipelineActivityLog(
  table: string | null,
  operation: string,
  sourceDb?: string | null,
  shadowSchema?: string | null, // Bổ sung phân lập theo shadow schema
  pageSize = 30,
) {
  return useQuery<ActivityLogResponse>({
    queryKey: ['pipeline-activity-log', table, operation, sourceDb, shadowSchema, pageSize],
    queryFn: async () => {
      const params: Record<string, string | number> = {
        page: 1,
        page_size: pageSize,
        operation,
      };
      if (table) params.target_table = table;
      if (sourceDb) params.source_database = sourceDb;
      if (shadowSchema) params.shadow_schema = shadowSchema; // Phân lập connector
      const { data } = await cmsApi.get<ActivityLogResponse>('/api/activity-log', { params });
      return { data: data.data || [], total: data.total || 0, page: data.page || 1, page_size: data.page_size || pageSize };
    },
    enabled: Boolean(table),
    refetchInterval: 15_000,
    staleTime: 10_000,
    retry: 1,
  });
}

export interface ClearActivityLogPayload {
  shadow_schema?: string;
  shadow_table?: string;
  source_database?: string;
  master_table?: string;
  before_date?: string;
  from_date?: string;
  to_date?: string;
  reason?: string;
}

export function useClearActivityLogMutation() {
  const queryClient = useQueryClient();
  return useMutation<{ ok: boolean; rows_deleted: number }, Error, ClearActivityLogPayload>({
    mutationFn: async (payload) => {
      const { data } = await cmsApi.delete<{ ok: boolean; rows_deleted: number }>('/api/activity-log', {
        params: payload,
        headers: {
          'X-Action-Reason': payload.reason || 'Dọn dẹp nhật ký hoạt động CDC',
        },
      });
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['pipeline-activity-log'] });
      queryClient.invalidateQueries({ queryKey: ['table-history'] });
      queryClient.invalidateQueries({ queryKey: ['recon-report'] });
    },
  });
}
```

---

### 2.2. Modal Dọn dẹp với DatePicker Động (`ReconPipelineGrid.tsx`)

```tsx
interface ClearActivityLogsModalProps {
  open: boolean;
  shadowSchema: string;
  shadowTable: string;
  sourceDb: string;
  masterTable: string | null;
  onClose: () => void;
}

export const ClearActivityLogsModal: React.FC<ClearActivityLogsModalProps> = ({
  open,
  shadowSchema,
  shadowTable,
  sourceDb,
  masterTable,
  onClose,
}) => {
  const [purgeMode, setPurgeMode] = useState<'before' | 'all'>('before');
  const [beforeDate, setBeforeDate] = useState<dayjs.Dayjs | null>(dayjs().subtract(7, 'day'));
  const clearMutation = useClearActivityLogMutation();

  const handleConfirm = async () => {
    let payload: ClearActivityLogPayload = {
      shadow_schema: shadowSchema,
      shadow_table: shadowTable,
      source_database: sourceDb,
      master_table: masterTable || undefined,
    };

    if (purgeMode === 'before') {
      if (!beforeDate) {
        message.warning('Vui lòng chọn ngày mốc để xóa');
        return;
      }
      payload.before_date = beforeDate.format('YYYY-MM-DD');
      payload.reason = `Xóa nhật ký pipeline ${shadowSchema}.${shadowTable} trước ngày ${payload.before_date}`;
    } else {
      payload.reason = `Xóa TOÀN BỘ nhật ký pipeline ${shadowSchema}.${shadowTable}`;
    }

    try {
      const res = await clearMutation.mutateAsync(payload);
      message.success(`Đã dọn dẹp thành công ${res.rows_deleted.toLocaleString()} bản ghi nhật ký!`);
      onClose();
    } catch (err: any) {
      message.error(`Dọn dẹp thất bại: ${err?.response?.data?.error || err?.message || 'Lỗi không xác định'}`);
    }
  };

  return (
    <Modal
      title={
        <Space>
          <ClearOutlined style={{ color: '#ff4d4f' }} />
          <span>Dọn dẹp nhật ký hoạt động CDC</span>
        </Space>
      }
      open={open}
      onCancel={onClose}
      onOk={handleConfirm}
      okText="Xác nhận dọn dẹp"
      okButtonProps={{ danger: true, loading: clearMutation.isPending }}
      destroyOnClose
      width={540}
    >
      <div style={{ marginBottom: 16 }}>
        <Text type="secondary" style={{ fontSize: 13 }}>
          Pipeline: <b>{shadowSchema}.{shadowTable}</b> (Nguồn: <code>{sourceDb}</code>) {masterTable ? ` ➔ ${masterTable}` : ''}
        </Text>
      </div>

      <Radio.Group
        value={purgeMode}
        onChange={(e) => setPurgeMode(e.target.value)}
        style={{ display: 'flex', flexDirection: 'column', gap: 14 }}
      >
        <Radio value="before">
          <Space>
            <span>Xóa các bản ghi cũ trước ngày:</span>
            <DatePicker
              value={beforeDate}
              onChange={(d) => setBeforeDate(d)}
              format="YYYY-MM-DD"
              disabled={purgeMode !== 'before'}
              allowClear={false}
            />
          </Space>
        </Radio>

        <Radio value="all">
          <Text type="danger" strong>
            Xóa TOÀN BỘ nhật ký của pipeline này (Reset sạch sẽ)
          </Text>
        </Radio>
      </Radio.Group>
    </Modal>
  );
};
```

---

## 3. Tái cấu trúc Bảng Đối soát 3 Trạm (`ReconPipelineGrid.tsx`)

### 3.1. Phân lập Triệt để Snapshot và Kafka CDC Stream
* Bắt trọn các cờ Snapshot: `is_snapshot === true`, `buffer_type === 'snapshot'`, `triggered_by === 'snapshot'`, `operation === 'snapshot.v2'`.
* **Đẩy vào Khối 1: `1. SNAPSHOT / RE-SYNC` (Shadow Nạp, Master Sync)**.
* **TUYỆT ĐỐI KHÔNG CỘNG VÀO CỘT KAFKA CDC, SHADOW VÀ MASTER CỦA DÒNG CHẢY STREAM**.
* Dòng chảy Stream CDC đo lường liên tục: Kafka In ($c+u+d$) $\rightarrow$ Shadow Write ($c+u+d$) $\rightarrow$ Master Transmute ($Ins+Upd+Skip$).

### 3.2. Cấu trúc Grouped Columns 6 Khối
* Cột cố định trái: `Ngày`.
* `1. SNAPSHOT / RE-SYNC` (Shadow Nạp, Master Sync).
* `2. KAFKA CDC BẮN VỀ` (Tổng Ops, c, u, d).
* `3. GHI VÀO DB SHADOW (Từ CDC Stream)` (Tổng Ops & Dòng DB, c, u, d).
* `4. GHI VÀO DB MASTER (Từ CDC Stream)` (Tổng Ops & Ghi DB, Ins, Upd, Skip).
* `5. ĐỐI SOÁT & DLQ (Lỗi Luồng CDC)` (DLQ, Thiếu, Stale, Heal).
* Cột cố định phải: `ĐÁNH GIÁ (FSM Luồng CDC)` (`fixed: 'right'`).
* Bỏ phân trang, cuộn dọc `scroll={{ y: 450, x: 1950 }}`.
