# Hồ Sơ Giải Pháp Kỹ Thuật: Edit, Delete và Danh Mục Tác Vụ trên Activity Manager

---

## 1. BACKEND CMS SERVICE (`cdc-cms-service`)

### A. Repository Interface & Implementation
```go
// File: internal/app/ports/repository.go
type WorkerScheduleRepo interface {
	Create(ctx context.Context, schedule *schedulermodel.WorkerSchedule) error
	GetByID(ctx context.Context, id int64) (*schedulermodel.WorkerSchedule, error)
	Update(ctx context.Context, id int64, updates map[string]interface{}) error
	Delete(ctx context.Context, id int64) error // Bổ sung method Delete
	CreateActivityLog(ctx context.Context, log *systemmodel.ActivityLog) error
	ResolveTargetTable(ctx context.Context, filter WorkerScheduleResolveFilter) (*WorkerScheduleScopeCandidate, error)
}
```

```go
// File: internal/infra/persistence/scheduler/worker_schedule_repo_gorm.go
func (r *workerScheduleRepoGorm) Delete(ctx context.Context, id int64) error {
	res := r.db.WithContext(ctx).Delete(&schedmodel.WorkerSchedule{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ports.ErrRecordNotFound
	}
	return nil
}
```

### B. CQRS Command Delete
```go
// File: internal/app/commands/scheduler/delete_worker_schedule.go
package scheduler

import (
	"context"
	"encoding/json"
	"errors"

	"cdc-cms-service/internal/app/ports"
)

var ErrWorkerScheduleDeleteNotFound = errors.New("worker_schedule_delete_not_found")

type DeleteWorkerScheduleCommand struct {
	ports.SyncCommandMixin
	ID int64 `json:"id"`
}

func (DeleteWorkerScheduleCommand) Type() string { return "worker_schedule.delete" }

func (c DeleteWorkerScheduleCommand) Validate() error {
	if c.ID <= 0 {
		return errors.New("invalid schedule id")
	}
	return nil
}

type DeleteWorkerScheduleHandler struct {
	repo ports.WorkerScheduleRepo
}

func NewDeleteWorkerScheduleHandler(repo ports.WorkerScheduleRepo) *DeleteWorkerScheduleHandler {
	return &DeleteWorkerScheduleHandler{repo: repo}
}

func (h *DeleteWorkerScheduleHandler) Handle(ctx context.Context, c ports.Command) (json.RawMessage, error) {
	cmd, ok := c.(DeleteWorkerScheduleCommand)
	if !ok {
		return nil, errors.New("worker_schedule.delete: command type mismatch")
	}

	if err := h.repo.Delete(ctx, cmd.ID); err != nil {
		if errors.Is(err, ports.ErrRecordNotFound) {
			return nil, ErrWorkerScheduleDeleteNotFound
		}
		return nil, err
	}

	body, _ := json.Marshal(map[string]interface{}{
		"status": "deleted",
		"id":     cmd.ID,
	})
	return body, nil
}
```

### C. Nâng cấp Update Command & Handler
```go
// File: internal/app/commands/scheduler/update_schedule.go
type UpdateScheduleCommand struct {
	ports.SyncCommandMixin
	ID              int64           `json:"id"`
	TargetTable     *string         `json:"target_table"` // Cho phép đổi bảng
	IntervalMinutes *int            `json:"interval_minutes"`
	IsEnabled       *bool           `json:"is_enabled"`
	Notes           *string         `json:"notes"`
	UpdatedBy       string          `json:"updated_by"`
	RunAtTime       *string         `json:"run_at_time"`
	Params          json.RawMessage `json:"params"`
	NextRunAt       *time.Time      `json:"next_run_at"`
	RunNow          *bool           `json:"run_now"`
}

// Trong Handle:
if cmd.TargetTable != nil {
	updates["target_table"] = *cmd.TargetTable
}
```

### D. Schedule Handler & Router
```go
// File: internal/api/scheduler/schedule_handler.go
type WorkerScheduleUpdateRequest struct {
	TargetTable     *string         `json:"target_table"`
	IntervalMinutes *int            `json:"interval_minutes"`
	IsEnabled       *bool           `json:"is_enabled"`
	Notes           *string         `json:"notes"`
	RunAtTime       *string         `json:"run_at_time"`
	Params          json.RawMessage `json:"params"`
	NextRunAt       *time.Time      `json:"next_run_at"`
	RunNow          *bool           `json:"run_now"`
}

func (h *ScheduleHandler) Delete(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return c.Status(400).JSON(fiber.Map{"error": "invalid schedule id"})
	}

	if h.bus == nil {
		return c.Status(503).JSON(fiber.Map{"error": "command bus not ready"})
	}

	user := middleware.GetUsername(c)
	cmd := schedulerCmd.DeleteWorkerScheduleCommand{ID: int64(id)}
	ctx := messaging.WithMetadata(c.UserContext(), user, c.Get("X-Correlation-Id"), c.Get("Idempotency-Key"))

	if _, err := h.bus.Execute(ctx, cmd); err != nil {
		if errors.Is(err, schedulerCmd.ErrWorkerScheduleDeleteNotFound) {
			return c.Status(404).JSON(fiber.Map{"error": "schedule_not_found"})
		}
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "schedule deleted", "id": id})
}
```

```go
// File: internal/router/router.go
dual("PATCH", admin, "/worker-schedule/:id", h.Scheduler.WorkerSchedule.Update)
dual("POST", admin, "/worker-schedule", h.Scheduler.WorkerSchedule.Create)
dual("DELETE", admin, "/worker-schedule/:id", h.Scheduler.WorkerSchedule.Delete) // Thêm route DELETE
```

---

## 2. FRONTEND CMS WEB (`cdc-cms-web/src/pages/ActivityManager.tsx`)

### A. Chuẩn hóa Danh mục Tác vụ
- **Loại bỏ hoàn toàn:** `drop-gin-index` và `create-default-columns`
- **Bổ sung mới:** `reconcile` (Đối soát dữ liệu) và `bridge` (Đồng bộ cầu nối)

```tsx
const ALL_OPERATIONS = [
  { value: 'source_count', label: 'Đếm bản ghi ngày tạo (Source Count)' },
  { value: 'confluence', label: 'Báo cáo Confluence Daily (Confluence Report)' },
  { value: 'reconcile', label: 'Đối soát dữ liệu (Reconcile)' },
  { value: 'bridge', label: 'Đồng bộ cầu nối (Bridge)' },
  { value: 'transform', label: 'Chuyển đổi field (Transform)' },
  { value: 'field-scan', label: 'Quét field mới (Field Scan)' },
  { value: 'partition-check', label: 'Kiểm tra partition' },
];

const opColors: Record<string, string> = {
  'source_count': 'cyan',
  'confluence': 'blue',
  'reconcile': 'green',
  'bridge': 'orange',
  'transform': 'purple',
  'field-scan': 'geekblue',
  'partition-check': 'lime',
};
```

---

### B. CASCADING SELECTOR 5 TẦNG KẾ THỪA 100% METADATA HUYẾT THỐNG (LINEAGE)

Để khắc phục triệt để lỗi selector giả tạo, hardcode container, và không tự động suy diễn engine / object type:

1. **Nạp đầy đủ các nguồn Metadata từ Backend:**
   - `connections`: Lấy từ `/api/v1/master-connections` (hoặc `/api/v1/registry/connections`)
   - `sourceObjects`: Lấy từ `/api/v1/source-objects`
   - `shadowBindings`: Lấy từ `/api/v1/shadow-bindings`
   - `masterBindings`: Lấy từ `/api/v1/masters`

2. **Quy tắc Liên kết Động 5 Tầng:**
   - **Tầng 1: `layer_type` (`source` | `shadow` | `master`):**
     Khi đổi Layer, chỉ nạp Connection của đúng layer đó, reset toàn bộ các tầng 2, 3, 4, 5.
   - **Tầng 2: `connection_code`:**
     - Nguồn: Danh sách Connection thuộc `role_type === layer_type` (hoặc match tương ứng).
     - **Tự động suy diễn Engine Type:** Gán `engine_type = conn.engine_type`. Khóa read-only (Tag hiển thị), không cho user chọn thủ công!
     - **Tự động suy diễn Object Type:**
       - Engine là `mongodb` -> `object_type = 'collection'`
       - Engine là SQL (`postgresql`, `mysql`, `clickhouse`) -> `object_type = 'table'`
       - Engine là `sftp` -> `object_type = 'file_batch'`
     - Reset Tầng 3, 4, 5.
   - **Tầng 3: `container_namespace`:**
     - **DẠNG DROPDOWN `<Select showSearch>`**, KHÔNG dùng Input tự gõ tự do hay hardcode:
       - Nếu `layer_type === 'source'`: Lấy các `source_db` / `source_schema` từ `sourceObjects` theo connection đó.
       - Nếu `layer_type === 'shadow'`: Lấy các `shadow_schema` thực tế từ `shadowBindings` (vd: `shadow_traitestpbs`, `shadow_traitestces`...).
       - Nếu `layer_type === 'master'`: Lấy các `master_schema` thực tế từ `masterBindings` (vd: `master_payment_bill_service`, `master_core_trans_proxy_history_service`...).
     - Reset Tầng 4, 5.
   - **Tầng 4: `object_name`:**
     - Dropdown `<Select showSearch>` chỉ hiển thị các đối tượng thuộc đúng Namespace ở Tầng 3:
       - Nếu `source`: Danh sách `source_table` / collection name của DB đó.
       - Nếu `shadow`: Danh sách `shadow_table` của schema đó.
       - Nếu `master`: Danh sách `master_name` của schema đó.
   - **Tầng 5: `date_field`:**
     - Tự động điền `timestamp_field` từ metadata của object đã chọn ở Tầng 4 (vd: `createdAt`, `created_at`, `updatedAt`).

#### Code Demo Component Cascading Selector:
```tsx
// 1. Filter Connections theo Layer Type
const filteredConnections = useMemo(() => {
  if (selectedLayer === 'source') {
    return (connections || []).filter(c => c.role_type === 'source' || (!c.role_type && c.connection_code !== 'default_shadow' && c.connection_code !== 'default_master'));
  }
  if (selectedLayer === 'shadow') {
    return (connections || []).filter(c => c.role_type === 'shadow' || c.connection_code === 'default_shadow');
  }
  if (selectedLayer === 'master') {
    return (connections || []).filter(c => c.role_type === 'master' || c.connection_code.includes('master'));
  }
  return connections || [];
}, [connections, selectedLayer]);

// 2. Filter Namespaces theo Layer và Connection
const availableNamespaces = useMemo(() => {
  const set = new Set<string>();
  if (selectedLayer === 'source') {
    sourceObjects.forEach(s => {
      if (!selectedConn || s.source_connection_code === selectedConn || s.source_db === selectedConn) {
        if (s.source_db) set.add(s.source_db);
      }
    });
  } else if (selectedLayer === 'shadow') {
    shadowBindings.forEach(s => {
      if (s.shadow_schema) set.add(s.shadow_schema);
    });
    // Fallback từ sourceObjects nếu shadowBindings chưa tải kịp
    sourceObjects.forEach(s => {
      if (s.shadow_schema) set.add(s.shadow_schema);
    });
  } else if (selectedLayer === 'master') {
    masterBindings.forEach(m => {
      if (m.master_schema) set.add(m.master_schema);
    });
  }
  return Array.from(set);
}, [selectedLayer, selectedConn, sourceObjects, shadowBindings, masterBindings]);

// 3. Filter Objects theo Layer và Namespace
const availableObjects = useMemo(() => {
  if (!selectedNamespace) return [];
  if (selectedLayer === 'source') {
    return sourceObjects
      .filter(s => s.source_db === selectedNamespace)
      .map(s => ({
        name: s.source_table,
        timestamp_field: s.timestamp_field || 'created_at',
      }));
  }
  if (selectedLayer === 'shadow') {
    const list: { name: string; timestamp_field: string }[] = [];
    shadowBindings.forEach(s => {
      if (s.shadow_schema === selectedNamespace) {
        list.push({ name: s.shadow_table, timestamp_field: 'created_at' });
      }
    });
    if (list.length === 0) {
      sourceObjects.forEach(s => {
        if (s.shadow_schema === selectedNamespace) {
          list.push({ name: s.target_table, timestamp_field: s.timestamp_field || 'created_at' });
        }
      });
    }
    return list;
  }
  if (selectedLayer === 'master') {
    return masterBindings
      .filter(m => m.master_schema === selectedNamespace)
      .map(m => ({
        name: m.master_name,
        timestamp_field: 'created_at',
      }));
  }
  return [];
}, [selectedLayer, selectedNamespace, sourceObjects, shadowBindings, masterBindings]);
```

---

### C. CỘT THAO TÁC (CHẠY NGAY, SỬA, XOÁ)
```tsx
{
  title: 'Thao tác', key: 'action', width: 220, fixed: 'right',
  render: (_, record) => (
    <Space size={6}>
      <Button
        type="link"
        size="small"
        icon={<PlayCircleOutlined />}
        loading={runningNowId === record.id}
        onClick={(e) => {
          e.stopPropagation();
          handleRunNow(record);
        }}
      >
        Chạy ngay
      </Button>

      <Button
        type="link"
        size="small"
        icon={<EditOutlined />}
        onClick={(e) => {
          e.stopPropagation();
          handleOpenEdit(record);
        }}
      >
        Sửa
      </Button>

      <Popconfirm
        title="Xoá lịch trình"
        description={`Bạn có chắc chắn muốn xoá lịch trình #${record.id} (${record.operation})?`}
        onConfirm={(e) => {
          e?.stopPropagation();
          handleDeleteSchedule(record);
        }}
        onCancel={(e) => e?.stopPropagation()}
        okText="Xoá"
        cancelText="Huỷ"
        okButtonProps={{ danger: true }}
      >
        <Button
          type="link"
          size="small"
          danger
          icon={<DeleteOutlined />}
          onClick={(e) => e.stopPropagation()}
        >
          Xoá
        </Button>
      </Popconfirm>
    </Space>
  ),
},
```

---

### D. MODAL & LOGIC SỬA LỊCH TRÌNH (EDIT SCHEDULE)
```tsx
const handleOpenEdit = (record: Schedule) => {
  setEditingSchedule(record);
  editForm.resetFields();

  if (record.operation === 'source_count') {
    editForm.setFieldsValue({
      interval_minutes: record.interval_minutes,
      run_at_time: record.run_at_time || '00:05',
      is_enabled: record.is_enabled,
      notes: record.notes,
      layer_type: record.params?.layer_type || 'source',
      connection_code: record.params?.connection_code || '',
      engine_type: record.params?.engine_type || 'postgresql',
      container_namespace: record.params?.container_namespace || '',
      object_name: record.params?.object_name || record.target_table || '',
      object_type: record.params?.object_type || 'table',
      date_strategy: record.params?.date_strategy || 'field',
      date_field: record.params?.date_field || 'created_at',
    });
  } else if (record.operation === 'confluence') {
    editForm.setFieldsValue({
      interval_minutes: record.interval_minutes,
      run_at_time: record.run_at_time || '06:00',
      is_enabled: record.is_enabled,
      notes: record.notes,
      confluence_base_url: record.params?.confluence?.base_url || 'https://goopay.atlassian.net',
      confluence_space_key: record.params?.confluence?.space_key || '~712020a3dcebcbe93646fea463f119dd38b08c',
      confluence_parent_page_id: record.params?.confluence?.parent_page_id || '463833042',
      confluence_user_email: record.params?.confluence?.user_email || '',
      confluence_api_token: '', // Để trống nếu không muốn đổi token
      confluence_title_template: record.params?.confluence?.title_template || '[Daily Data Check] {{.ReportDate}} - Core Trans Proxy History Service',
      selected_tables: (record.params?.selected_tables || []).map((t: any) => t.object_name),
    });
  } else {
    editForm.setFieldsValue({
      interval_minutes: record.interval_minutes,
      run_at_time: record.run_at_time,
      is_enabled: record.is_enabled,
      notes: record.notes,
      target_table: record.target_table,
    });
  }
  setEditVisible(true);
};

const handleSaveEdit = async (values: any) => {
  if (!editingSchedule) return;
  setEditLoading(true);
  try {
    let payload: any = {
      interval_minutes: values.interval_minutes || 1440,
      run_at_time: values.run_at_time || null,
      notes: values.notes || null,
      is_enabled: values.is_enabled,
    };

    if (editingSchedule.operation === 'source_count') {
      payload.target_table = values.object_name || editingSchedule.target_table;
      payload.params = {
        layer_type: values.layer_type || 'source',
        connection_code: values.connection_code || '',
        engine_type: values.engine_type || 'postgresql',
        container_namespace: values.container_namespace || '',
        object_name: values.object_name || '',
        object_type: values.object_type || 'table',
        date_strategy: values.date_strategy || 'field',
        date_field: values.date_field || 'created_at',
      };
    } else if (editingSchedule.operation === 'confluence') {
      const selectedTables = (values.selected_tables || []).map((tblName: string) => {
        const found = sourceObjects.find((s) => s.target_table === tblName || s.source_table === tblName);
        return {
          layer_type: 'master',
          connection_code: 'default_master',
          container_namespace: found?.source_db || 'public',
          object_name: tblName,
          object_type: 'table',
          display_name: tblName,
        };
      });

      const apiToken = values.confluence_api_token || editingSchedule.params?.confluence?.api_token || '';

      payload.target_table = values.selected_tables?.[0] || editingSchedule.target_table;
      payload.params = {
        confluence: {
          base_url: values.confluence_base_url || 'https://goopay.atlassian.net',
          space_key: values.confluence_space_key || '~712020a3dcebcbe93646fea463f119dd38b08c',
          parent_page_id: values.confluence_parent_page_id || '463833042',
          user_email: values.confluence_user_email || '',
          api_token: apiToken,
          title_template: values.confluence_title_template || '[Daily Data Check] {{.ReportDate}} - Core Trans Proxy History Service',
        },
        selected_tables: selectedTables,
      };
    } else {
      payload.target_table = values.target_table || editingSchedule.target_table;
    }

    await cmsApi.patch(`/api/worker-schedule/${editingSchedule.id}`, payload);
    message.success(`Đã cập nhật lịch trình #${editingSchedule.id}`);
    setEditVisible(false);
    fetchData();
  } catch (err) {
    message.error(humanizeApiError(err, 'Cập nhật thất bại'));
  } finally {
    setEditLoading(false);
  }
};
```
```
