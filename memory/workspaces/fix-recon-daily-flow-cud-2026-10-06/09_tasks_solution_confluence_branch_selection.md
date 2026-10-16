# Hồ Sơ Giải Pháp Kỹ Thuật: Bộ Chọn Nhánh Cụ Thể (Source ➔ Shadow ➔ Master) Trên UI & Đối Soát Chính Xác 3 Trạm Trên Confluence Report

- **Thời gian:** 2026-10-15T10:55:00+07:00
- **Trạng thái:**PROPOSE ONLY (Chờ lệnh `APPROVE` / `"sửa đi"` từ User theo Rule #1 & Rule #13)
- **Phạm vi tác động:**
  1. `cdc-cms-web/src/pages/ActivityManager.tsx`
  2. `centralized-data-service/internal/service/confluence/confluence_runner.go`

---

## 1. Mục Tiêu Thiết Kế (Theo Đúng Yêu Cầu Của User)

1. **Trên UI (`ActivityManager.tsx`):**
   - Người dùng chọn **cụ thể đến từng nhánh (`1. Source` ➔ `2. Shadow` ➔ `3. Master`)** để đưa vào báo cáo Confluence.
   - Không hiển thị tràn lan toàn bộ các nhánh trong hệ thống, và không chọn mù mờ theo tên bảng trần trụi (`export_jobs`).
2. **Dưới Backend (`confluence_runner.go`):**
   - Tuyệt đối xóa bỏ câu lệnh `LIMIT 1` bốc mò và xóa bỏ truy vấn vào bảng không tồn tại `cdc_system.recon_hourly_pipeline_stats`.
   - Với mỗi nhánh `Source ➔ Shadow ➔ Master` được chọn, truy vấn chính xác số bản ghi ngày tạo (`creation_count`) của **cả 3 trạm (`source`, `shadow`, `master`)** trong `cdc_system.daily_source_counts` theo bộ khóa 5 thành phần `(report_date, layer_type, connection_code, container_namespace, object_name)`, kết hợp lưu lượng CDC thực tế từ `cdc_system.cdc_activity_log`.

---

## 2. Thiết Kế Chi Tiết Tầng Frontend (`ActivityManager.tsx`)

### 2.1. Component `ConfluenceBranchSelector` (Bộ chọn phân tầng 3 bước + Danh sách nhánh đã chọn)
Sử dụng trực tiếp dữ liệu metadata đã nạp sẵn trong `ActivityManager`:
- `sourceObjects` (`/api/v1/source-objects`)
- `rawShadowBindings` (`/api/v1/shadow-bindings`)
- `rawMasterBindings` (`/api/v1/masters`)

**Luồng thao tác 3 bước để chọn nhánh cụ thể:**
1. **Ô 1 — Chọn Nguồn (`Source`):**
   - Hiển thị: `[source_connection_code] source_db.source_table (source_type)`
   - Ví dụ: `[traitestces] centrallized-export-service.export-jobs (mongodb)` (`source_object_id = 201`)
2. **Ô 2 — Chọn Bảng Trung Gian (`Shadow` thuộc Source đã chọn):**
   - Tự động lọc từ `rawShadowBindings` có `sb.source_object_id === selectedSourceId`.
   - Hiển thị: `[default_shadow] shadow_schema.shadow_table`
   - Ví dụ: `[default_shadow] shadow_traitestces.export_jobs` (`shadow_binding_id = 201`)
3. **Ô 3 — Chọn Bảng Đích (`Master` thuộc Shadow đã chọn - Hỗ trợ chọn 1 hoặc nhiều Master của Shadow đó):**
   - Tự động lọc từ `rawMasterBindings` có `mb.shadow_binding_id === selectedShadowId`.
   - Hiển thị: `[effectiveConnCode] master_schema.master_table`
   - Ví dụ với `shadow_traitestces.export_jobs` (`shadow_binding_id = 201`), hiển thị 3 nhánh Master đang active:
     + `[default_master] master_centrallized_export_service.export_jobs` (`id = 48`)
     + `[default_master] master_centrallized_export_service.export_jobs_1` (`id = 52`)
     + `[master_2] master_centrallized_export_service.export_jobs_2` (`id = 51`)
4. **Nút `+ Thêm nhánh vào báo cáo` & Bảng danh sách nhánh đã chọn:**
   - Khi người dùng chọn xong và bấm `+ Thêm nhánh`, nhánh đó được thêm vào danh sách `selected_tables` của Form.
   - Bên dưới hiển thị danh sách các nhánh đã chọn dạng dòng trực quan:
     `[traitestces] centrallized-export-service.export-jobs ➔ [default_shadow] shadow_traitestces.export_jobs ➔ [master_2] master_centrallized_export_service.export_jobs_2` kèm nút **Xoá (`DeleteOutlined`)**.

### 2.2. Cấu trúc Payload `params.selected_tables` gửi xuống Backend
Mỗi nhánh được chọn lưu đầy đủ thông tin của cả 3 trạm:
```json
{
  "branch_key": "src:201|shd:201|mst:51",
  "source_object_id": 201,
  "source_connection_code": "traitestces",
  "source_namespace": "centrallized-export-service",
  "source_object_name": "export-jobs",
  "source_object_type": "collection",
  "shadow_binding_id": 201,
  "shadow_connection_code": "default_shadow",
  "shadow_schema": "shadow_traitestces",
  "shadow_table": "export_jobs",
  "master_binding_id": 51,
  "master_connection_code": "master_2",
  "master_schema": "master_centrallized_export_service",
  "master_table": "export_jobs_2",
  "display_name": "[traitestces] centrallized-export-service.export-jobs ➔ [default_shadow] shadow_traitestces.export_jobs ➔ [master_2] master_centrallized_export_service.export_jobs_2"
}
```

---

## 3. Thiết Kế Chi Tiết Tầng Backend (`confluence_runner.go`)

### 3.1. Struct `ConfluenceBranchConfig` & `confluenceTableRow`
```go
type ConfluenceBranchConfig struct {
	BranchKey            string `json:"branch_key"`
	SourceObjectID       int64  `json:"source_object_id"`
	SourceConnectionCode string `json:"source_connection_code"`
	SourceNamespace      string `json:"source_namespace"`
	SourceObjectName     string `json:"source_object_name"`
	SourceObjectType     string `json:"source_object_type"`

	ShadowBindingID      int64  `json:"shadow_binding_id"`
	ShadowConnectionCode string `json:"shadow_connection_code"`
	ShadowSchema         string `json:"shadow_schema"`
	ShadowTable          string `json:"shadow_table"`

	MasterBindingID      int64  `json:"master_binding_id"`
	MasterConnectionCode string `json:"master_connection_code"`
	MasterSchema         string `json:"master_schema"`
	MasterTable          string `json:"master_table"`

	// Backward-compatible fields
	ConnectionCode     string `json:"connection_code,omitempty"`
	ContainerNamespace string `json:"container_namespace,omitempty"`
	ObjectName         string `json:"object_name,omitempty"`
	ObjectType         string `json:"object_type,omitempty"`
	DisplayName        string `json:"display_name,omitempty"`
}

type confluenceTableRow struct {
	SourceLabel  string
	ShadowLabel  string
	MasterLabel  string
	SourceCount  string
	ShadowCount  string
	MasterCount  string
	CDCFlowLabel string
	Status       string
}
```

### 3.2. Hàm tra cứu chính xác `lookupLayerDailyCount` (Không `LIMIT 1` bừa bãi)
```go
func (r *ConfluenceReportRunner) lookupLayerDailyCount(
	ctx context.Context,
	reportDateStr string,
	layerType string,
	connectionCode string,
	namespace string,
	objectName string,
) (int64, bool) {
	if objectName == "" {
		return 0, false
	}
	var rec struct {
		CreationCount int64  `gorm:"column:creation_count"`
		LayerType     string `gorm:"column:layer_type"`
	}
	res := r.dbControl.WithContext(ctx).Raw(`
		SELECT creation_count, layer_type
		FROM cdc_system.daily_source_counts
		WHERE report_date = ?
		  AND layer_type = ?
		  AND connection_code = ?
		  AND container_namespace = ?
		  AND object_name = ?
		ORDER BY updated_at DESC
		LIMIT 1
	`, reportDateStr, layerType, connectionCode, namespace, objectName).Scan(&rec)
	if res.Error == nil && res.RowsAffected > 0 && rec.LayerType != "" {
		return rec.CreationCount, true
	}
	return 0, false
}
```

### 3.3. Bảng XHTML Xuất Lên Atlassian Confluence
Khi chọn nhánh `[traitestces] centrallized-export-service.export-jobs ➔ [default_shadow] shadow_traitestces.export_jobs ➔ [master_2] master_centrallized_export_service.export_jobs_2`, bảng trên Confluence hiển thị:

| Trạm Nguồn (Source) | Trạm Trung Gian (Shadow) | Trạm Đích (Master) | Số Đếm Ngày Tạo (Source) | Số Đếm Ngày Tạo (Shadow) | Số Đếm Ngày Tạo (Master) | Lưu Lượng CDC (Kafka / Shadow / Master) | Trạng Thái |
| :--- | :--- | :--- | ---: | ---: | ---: | ---: | :--- |
| **`[traitestces]`**<br>`centrallized-export-service.export-jobs` | **`[default_shadow]`**<br>`shadow_traitestces.export_jobs` | **`[master_2]`**<br>`master_centrallized_export_service.export_jobs_2` | **0** | **0** | **0** | 0 / 0 / 0 | 🟢 **MATCH (Khớp 100%)** |
