# HỒ SƠ GIẢI PHÁP KỸ THUẬT: CENTRALIZED RESOLVERS & PHÂN LẬP BÁO CÁO RECON/HEAL ĐA KẾT NỐI
**Workspace:** `FeatureTransmuteShadowToMasterCustomDB20260914`  
**Mã hồ sơ:** `SOL-HEAL-PIPELINE-ISOLATION-V1`

---

## I. MỤC TIÊU KỸ THUẬT

1. **Chấm dứt hoàn toàn tình trạng "Code rối nùi, mỗi nơi parse một kiểu":**
   Xây dựng bộ hàm xài chung (Centralized Helpers/Resolvers) để nhận diện và truy vấn connect, schema, table cho toàn bộ hệ thống.
2. **Sửa dứt điểm Lỗi 1 (Báo cáo hiển thị chéo pipeline):**
   Đảm bảo khi mở "Phiên đã xử lý" của Master Table `export_jobs_2`, báo cáo hiển thị chính xác các phiên của `export_jobs_2`, không bị biến mất và không bị nhảy sang bảng `export_jobs`.
3. **Sửa dứt điểm Lỗi 2 (Nguy cơ sync/prune nhầm database trong Heal Segment B):**
   Đảm bảo lệnh Heal luôn gửi kèm `master_binding_id` trong NATS event `cdc.cmd.transmute` và luồng Prune Orphan Master kết nối đúng dynamic database (`master_2`) qua `ConnectionManager`.

---

## II. THIẾT KẾ GIẢI PHÁP CHI TIẾT

### 1. Centralized Helper trên Backend CMS Service (`cdc-cms-service`)

Tạo helper function dùng chung trong `internal/infra/persistence/recon/recon_read_repo_gorm.go`:

```go
// ReconTargetScope đại diện cho phạm vi lọc dữ liệu báo cáo đối soát
type ReconTargetScope struct {
	Table        string // input param
	ShadowSchema string
	ShadowTable  string
	MasterSchema string
	MasterTable  string
	Segment      string // 'source_shadow' hoặc 'shadow_master'
}

// buildReconReportWhere sinh câu WHERE và arguments chuẩn xác 100%,
// không bao giờ ghi đè làm mất điều kiện lọc của Master Table.
func buildReconReportWhere(scope ReconTargetScope) (string, []interface{}) {
	var clauses []string
	var args []interface{}

	// 1. Phân giải tên bảng nếu có dấu chấm (schema.table)
	cleanTable := scope.Table
	if strings.Contains(cleanTable, ".") {
		parts := strings.Split(cleanTable, ".")
		if len(parts) > 1 {
			if scope.ShadowSchema == "" {
				scope.ShadowSchema = parts[0]
			}
			cleanTable = parts[len(parts)-1]
		}
	}

	// 2. Xử lý trường hợp có cả Shadow Schema và Shadow Table rõ ràng
	if scope.ShadowSchema != "" {
		clauses = append(clauses, "shadow_schema = ?")
		args = append(args, scope.ShadowSchema)

		shadowTbl := scope.ShadowTable
		if shadowTbl == "" && scope.MasterTable == "" {
			// Fallback: nếu không chỉ định rõ, cleanTable có thể là shadow hoặc master
			shadowTbl = cleanTable
		}

		if scope.MasterTable != "" {
			// Đây là chặng Shadow -> Master cụ thể!
			clauses = append(clauses, "segment = 'shadow_master'")
			if shadowTbl != "" && shadowTbl != scope.MasterTable {
				clauses = append(clauses, "shadow_table = ?")
				args = append(args, shadowTbl)
			}
			clauses = append(clauses, "master_table = ?")
			args = append(args, scope.MasterTable)
			if scope.MasterSchema != "" {
				clauses = append(clauses, "COALESCE(NULLIF(master_schema, ''), 'public') = COALESCE(NULLIF(?, ''), 'public')")
				args = append(args, scope.MasterSchema)
			}
		} else {
			// Trường hợp chỉ có shadow_schema và table:
			// Match hoặc là shadow_table = cleanTable (ở Segment A)
			// HOẶC master_table = cleanTable (ở Segment B)
			clauses = append(clauses, "(shadow_table = ? OR master_table = ?)")
			args = append(args, cleanTable, cleanTable)
		}
	} else {
		// Không có shadow schema: lọc fallback
		clauses = append(clauses, "(shadow_table = ? OR master_table = ?)")
		args = append(args, cleanTable, cleanTable)
	}

	return strings.Join(clauses, " AND "), args
}
```

Áp dụng helper này vào cả `GetTableHistory` và `ListUnhealedReports`:
- Triệt tiêu hoàn toàn dòng gán cứng tai hại: `where = "shadow_schema = ? AND shadow_table = ?"`.
- Khi client truyền `table = "export_jobs_2"` và `shadow_schema = "shadow_traitestces"`:
  Helper sẽ match đúng `master_table = 'export_jobs_2' AND shadow_schema = 'shadow_traitestces'` $\rightarrow$ Trả về đúng bản ghi 230!

---

### 2. Centralized Helper trên CDS Worker Engine (`centralized-data-service`)

Trong `internal/handler/recon/recon_execute_heal_handler.go`:

1. **Phân giải Master Target xài chung:**
```go
func (h *ExecuteHealHandler) resolveMasterTargetInfo(ctx context.Context, rpt *modelrecon.ReconciliationReport) (int64, string, error) {
	// Query cdc_system.master_binding để lấy master_binding_id và master_connection_key
	type bindingInfo struct {
		ID                  int64  `gorm:"column:id"`
		MasterConnectionKey string `gorm:"column:master_conn_key"`
	}
	var info bindingInfo
	err := h.reconCore.ControlPlane().WithContext(ctx).Raw(`
		SELECT mb.id, COALESCE(cr.connection_code, 'default') as master_conn_key
		FROM cdc_system.master_binding mb
		JOIN cdc_system.shadow_binding sb ON mb.shadow_binding_id = sb.id
		LEFT JOIN cdc_system.connection_registry cr ON mb.master_connection_id = cr.id
		WHERE sb.shadow_schema = ? AND sb.shadow_table = ? AND mb.master_table = ? AND mb.is_active = true
		ORDER BY mb.id DESC LIMIT 1
	`, rpt.ShadowSchema, rpt.ShadowTable, rpt.MasterTable).Scan(&info).Error

	if err != nil {
		return 0, "", err
	}
	return info.ID, info.MasterConnectionKey, nil
}
```

2. **Cập nhật `publishTransmuteChunked`**:
Gửi kèm `master_binding_id`:
```go
func (h *ExecuteHealHandler) publishTransmuteChunked(ctx context.Context, table string, masterBindingID int64, sourceIDs []string, triggeredBy string) (int, error) {
	...
	payload, _ := json.Marshal(map[string]any{
		"master_table":      table,
		"master_binding_id": masterBindingID,
		"_source_ids":       sourceIDs[start:end],
		"triggered_by":      triggeredBy,
	})
	...
}
```

3. **Cập nhật Prune Orphan Master**:
Xóa bỏ hardcode `masterDB := h.reconCore.MasterPlane()`:
```go
// Thay vì dùng masterDB cũ của default_master:
masterDB, errDB := h.reconCore.GetMasterAgent(ctx, masterConnectionKey)
if errDB != nil || masterDB == nil {
    observability.Ctx(ctx, h.logger).Error("[execute-heal-b] cannot resolve dynamic master agent", zap.String("key", masterConnectionKey))
    return healed
}
// Chạy DELETE trên agent chuẩn của connection đó
```

---

### 3. Centralized Helper trên CMS Web Frontend (`cdc-cms-web`)

Tạo file `src/utils/pipelineIdentity.ts`:

```typescript
export interface PipelineIdentity {
  shadowSchema: string;
  shadowTable: string;
  masterSchema?: string;
  masterTable?: string;
  sourceDb?: string;
}

export function extractPipelineIdentity(r: any): PipelineIdentity {
  const shadowSchema = r.shadow_schema || r.shadowSchema || '';
  const shadowTable = r.shadow_table || r.shadowTable || (r.segment !== 'shadow_master' ? r.target_table : '') || '';
  const masterSchema = r.master_schema || r.masterSchema || undefined;
  const masterTable = r.master_table || r.masterTable || (r.segment === 'shadow_master' ? r.target_table : undefined);
  return { shadowSchema, shadowTable, masterSchema, masterTable };
}
```

Cập nhật `ExecuteHealModal.tsx`:
- Nhận props: `shadowSchema`, `shadowTable`, `masterTable`, `masterSchema`.
- Khi gọi `useTableHistory`:
  ```typescript
  const { data: historyData } = useTableHistory(
    open ? (shadowTable || table) : null,
    shadowSchema,
    masterTable,
    100,
    true
  );
  ```
- Khi gọi `useUnhealedReports`:
  Truyền cả `shadow_schema` và `master_table` vào query params!

---

## III. DEFINITION OF DONE (DOD GATES)

1. **G1 (Truy vết yêu cầu):** Khắc phục toàn bộ 2 vấn đề user chỉ ra, không còn lỗi nhảy chéo pipeline.
2. **G3 (Test thật):** Compile pass 100% cả 3 repo (`go build`, `npm run build`).
3. **G6 (Tính chính xác dữ liệu):** Mở Modal Heal ở dòng `export_jobs_2` trên UI, tab "Phiên đã xử lý" phải hiển thị đúng phiên ID 230 ("thiếu 2 • đã heal 2").
