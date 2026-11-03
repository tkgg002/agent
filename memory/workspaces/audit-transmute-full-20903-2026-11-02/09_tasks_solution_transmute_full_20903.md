# 09. Tasks Solution — `transmute_full_20903`

## 1. Bối cảnh & Bằng chứng Thực tế (VERIFIED 100% từ Code + Runtime API `:8083`)

### 1.1. Giải mã trực diện câu hỏi: Tại sao bấm "Chạy ngay (toàn bộ)" trên `transaction_history_2` (`binding_id=76`) chỉ đồng bộ `20,903 / 20,903` dòng?

1. **Luồng thực thi của nút "Chạy ngay (toàn bộ)":**
   - Trên UI (`cdc-cms-web/src/pages/MasterRegistry.tsx:326-346`), nút **"Chạy ngay (toàn bộ)"** gửi `POST /api/v1/masters/transaction_history_2/transmute?id=76` với body `{ "full_sync": true }`.
   - Badge `Synced (20,903 / 20,903)` (`cdc-cms-web/src/pages/MasterRegistry.tsx:166-200`) hiển thị `processed_rows / total_rows` từ API `GET /api/v1/master-bindings/76/transmute-job-status`.
   - Ở phía Worker (`centralized-data-service/internal/service/master/transmuter.go:209-215` & `1026-1037`), khi chạy `full_sync: true` (`watermark = 0`, `onlySourceIDs` rỗng), `TransmuterModule.Run` gọi `t.countShadowRows(ctx, masterRow, 0)`:
     ```sql
     SELECT COUNT(*) FROM "shadow_testcoretrans"."transaction_history" WHERE ("_gpay_id")::bigint > 0
     ```
     Tức là **Master Full Transmute luôn đếm và quét 100% dữ liệu đang có trong bảng Primary Shadow được gắn với Master Binding đó**, không đọc trực tiếp từ DB nguồn (`source_object_id=217`).

2. **Sự thật trong Runtime: Có 2 cặp bảng Shadow khác nhau cùng trỏ về 1 bảng nguồn `transaction_history` (`250,820` dòng):**
   Đối chiếu từ `GET /api/v1/masters`, `GET /api/v1/shadow-bindings`, `GET /api/v1/reconciliation/report` và `GET /api/v1/activity-log`:

   | Tiêu chí | `transaction_history_1` (`binding_id=75`) | `transaction_history_2` (`binding_id=76` — bảng anh đang mở) |
   | :--- | :--- | :--- |
   | **Bảng nguồn (`source_object_id=217`)** | `core-trans-proxy-history-service.public.transaction_history` (**`250,820` dòng**) | `core-trans-proxy-history-service.public.transaction_history` (**`250,820` dòng**) |
   | **Primary Shadow gắn vào Master** | `shadow_testcoretrans.transaction_history_1` (`shadow_binding_id=231`) | `shadow_testcoretrans.transaction_history` (`shadow_binding_id=222`) |
   | **Secondary Shadow (Payload)** | `shadow_testcoretrans.cdc_transaction_history_payload_1` (`shadow_binding_id=233`) | `shadow_testcoretrans.transaction_history_payload` (`shadow_binding_id=237`) |
   | **Số dòng thực tế trong Primary Shadow** | **`250,820 / 250,820` dòng** (`diff = 0`, `status = "ok"`, Recon `id=538785`) | **`20,903` dòng** lúc `11:44:12` (`20,909` dòng lúc `11:46:31`, thiếu **`229,911` dòng**, `drift_pct = 91.66%`, Recon `id=538783`) |
   | **Kết quả "Chạy ngay (toàn bộ)"** | Quét 100% Shadow `#231` -> **`250,820` dòng** (Recon `id=538782`) | Quét 100% Shadow `#222` -> **`20,903 / 20,903` dòng** (Activity Log `id=142943`: `"scanned": 20903, "inserted": 20903, "run_type": "full"`) |

   **=> Kết luận nguyên nhân 1 (Cấu hình & Dữ liệu Shadow):**
   Worker Transmuter **không hề giới hạn hay bỏ sót dòng nào của Shadow**. Sở dĩ `transaction_history_2` (`binding_id=76`) chỉ ra `20,903 / 20,903` là vì khi tạo `transaction_history_2`, nó được gắn vào bảng Primary Shadow cũ `shadow_testcoretrans.transaction_history` (`shadow_binding_id=222`) — bảng Shadow này **chưa chạy Heal/Snapshot lịch sử** nên trong bảng vật lý chỉ có đúng `20,903` dòng CDC mới (trong khi bảng Shadow đã có đủ `250,820` dòng là `shadow_testcoretrans.transaction_history_1` — `shadow_binding_id=231`).

---

### 1.2. Phát hiện Lỗi Code Nghiêm trọng trên chính `binding_id=76` (`http://localhost:5173/masters/transaction_history_2/mappings?binding_id=76`)

Khi kiểm tra danh sách Mapping Rules của `binding_id=76` (`GET /api/v1/master-mapping-rules?master_binding_id=76`) và đối chiếu Recon Report `id=538783` / `id=538793` (`shadow_total = 20,909` nhưng `master_total = 20,916`, lệch `-7` dòng trùng), phát hiện chuỗi bug trực tiếp trong code:

- **Finding `F1` (Bug sinh rule PK `_id` mồ côi rỗng `source_field=""` & `data_type=""` trong `cdc-cms-service`):**
  - Tại `cdc-cms-service/internal/infra/persistence/master/master_multi_shadow_repo_gorm.go:67-80` (`cloneAllShadowRulesTx`), khi tạo/approve Master Binding cho đích MongoDB (`binding_id=76`, `spec.pk = "_id"`), hàm `resolveMasterPKTargetColumn` trả về `pkCol = "_id"`.
  - Vì Primary Shadow `shadow_binding_id=222` có nguồn PostgreSQL (`primary_key_field = "id"`), rule PK được clone từ `mapping_rule_v2` (`v2.id=935`) có `target_column = "id"` (`rule id=1852`, trạng thái `pending`).
  - Do chưa có rule nào mang `target_column = "_id"`, khối lệnh ở dòng `69-78` tự động chèn thêm 1 dòng PK vào `cdc_system.mapping_rule_master`:
    ```sql
    INSERT INTO cdc_system.mapping_rule_master
      (master_binding_id, mapping_v2_id, target_column, is_active, status, created_by, updated_by)
    SELECT ?, NULL, ?, true, 'pending', ?, ?
    ```
  - Câu lệnh này chèn `mapping_v2_id = NULL` nhưng **không truyền `source_field` và `data_type`**, tạo ra rule `id=1862` trên trang `/masters/transaction_history_2/mappings?binding_id=76` với:
    `target_column = "_id"`, `mapping_v2_id = 0`, `source_field = ""`, `data_type = ""`.

- **Finding `F2` (Bug `loadRules` đánh rớt rule `_id` rỗng `data_type`, làm MongoDB Master mất khóa chính và nhân bản document khi CDC Update trong `centralized-data-service`):**
  - Mặc dù anh đã bấm **Approve** cho rule `id=1862` (`target_column = "_id"`, `status = "approved"`), khi Worker chạy `TransmuterModule.loadRules` (`centralized-data-service/internal/service/master/transmuter.go:918-955`), câu SQL `COALESCE(m.data_type, v2.data_type)` trả về `""` (do `m.data_type` NULL và `m.mapping_v2_id` NULL).
  - Tại dòng `946`: `if !t.typeRes.Validate(r.DataType)` trả về `false` với `""`, khiến Worker **âm thầm skip luôn rule `_id` (`id=1862`)**!
  - Hệ quả dây chuyền:
    1. Trong `colTypes` không có `_id` (và rule `id=1852` đang `pending`), nên `resolveConflictTarget` (`centralized-data-service/internal/service/master/transmuter_identity.go:85-98`) fallback khóa upsert về `"_gpay_id"`.
    2. Khi chạy Full Sync (`transmuter.go:536`), `_gpay_id` lấy `row.GpayID` của Shadow; nhưng khi nhận sự kiện CDC `UPDATE` realtime (`transmuter_events_batch.go:169`), `rec["_gpay_id"] = nextMasterSonyflakeGpayID()` sinh Sonyflake ID mới tinh cho mỗi lần update!
    3. Kết quả là mỗi bản ghi `UPDATE` từ CDC không tìm thấy `_gpay_id` cũ trên MongoDB `devtest.transaction_history_2` mà **insert thêm document mới gây trùng lặp** (lệch `20,916` vs `20,909`, `diff = -7` trên Recon Report `id=538793`).

---

## 2. Ràng buộc Kỹ thuật & Bảng Kiểm tra Toàn diện Call-Site (`L007`, `L023`)

- **Technical Constraints (từ `GEMINI.md` & `lessons/INDEX.md`):**
  - `L007` / `L023`: Khi sửa schema/query của `mapping_rule_master`, bắt buộc kiểm tra mọi call-site đọc/ghi (`cloneAllShadowRulesTx`, `syncRulesFromAllShadowsTx`, `masterRuleSelectJoin`, `TransmuterModule.loadRules`).
  - **Cạm bẫy quan trọng đã phát hiện khi đối chiếu call-site:** Không được gán chung `mapping_v2_id = v2_pk.id` cho rule `target_column = "_id"` nếu rule `target_column = "id"` cũng đang giữ `mapping_v2_id = v2_pk.id`, vì `syncRulesFromAllShadowsTx` (`master_multi_shadow_repo_gorm.go:171-204`) sẽ tự động chạy `UPDATE mapping_rule_master SET target_column = v2.target_column WHERE m.mapping_v2_id = v2.id AND m.target_column <> v2.target_column` làm đụng độ `UNIQUE (master_binding_id, target_column)`!
  - Do đó, rule PK alias (`_id` với `mapping_v2_id = NULL`) phải lưu trực tiếp `source_field` (tên cột PK ở Primary Shadow / `source_object_registry.primary_key_field`, vd `"id"`) và `data_type` (kiểu dữ liệu từ `v2` hoặc `'TEXT'`) ngay trên dòng `mapping_rule_master`.

| STT | Call-site (`file:line`) | Hành vi hiện tại | Tác động sau khi áp dụng giải pháp |
| :--- | :--- | :--- | :--- |
| 1 | `cdc-cms-service/internal/infra/persistence/master/master_multi_shadow_repo_gorm.go:67-80` (`cloneAllShadowRulesTx`) | Chèn `(master_binding_id, NULL, pkCol)` với `source_field = NULL`, `data_type = NULL` | Điền sẵn `source_field` (từ PK của Primary Shadow `v2.target_column` / `so.primary_key_field` / `pkCol`) và `data_type` (từ `v2.data_type` hoặc `'TEXT'`), đồng thời tự backfill nếu dòng `pkCol` đã tồn tại nhưng đang rỗng `source_field`/`data_type` |
| 2 | `cdc-cms-service/internal/infra/persistence/master/master_multi_shadow_repo_gorm.go:171-204` (`syncRulesFromAllShadowsTx`) | Rename các rule có `m.mapping_v2_id = v2.id` theo `v2.target_column` | Giữ `mapping_v2_id = NULL` cho rule PK alias (`_id`), đảm bảo `syncRulesFromAllShadowsTx` không bao giờ ghi đè `_id` thành `id` |
| 3 | `cdc-cms-service/internal/infra/persistence/master/master_mapping_rule_repo_gorm.go:99-101` (`masterRuleSelectJoin`) | `COALESCE(v2.target_column, m.source_field)` và `COALESCE(m.data_type, v2.data_type)` | Dùng `COALESCE(NULLIF(v2.target_column, ''), NULLIF(m.source_field, ''))` và `COALESCE(NULLIF(m.data_type, ''), NULLIF(v2.data_type, ''))` để hiển thị chính xác `source_field="id"`, `data_type="TEXT"` trên UI |
| 4 | `centralized-data-service/internal/service/master/transmuter.go:918-955` (`loadRules`) | Đọc `COALESCE(m.source_field, v2.source_field)` & `COALESCE(m.data_type, v2.data_type)` và skip rule nếu `DataType == ""` | Dùng `NULLIF` trong SQL và bổ sung fallback cho rule PK (`strings.EqualFold(r.TargetColumn, masterPK)`): nếu `r.SourceField == ""` thì lấy `row.SourcePK` (hoặc `r.TargetColumn`), nếu `r.DataType == ""` thì gán `"TEXT"` trước khi `Validate` |

---

## 3. Phương án Duy nhất Tối ưu & Code Demo (Ánh xạ 1-1 `F1` & `F2`)

### 3.1. Xử lý `F1`: Tự động điền & backfill `source_field`, `data_type` cho rule PK trong `cdc-cms-service`

**File:** `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/master/master_multi_shadow_repo_gorm.go` (`L66-L81`)

```go
		// Chỉ tự động đảm bảo cột PK trên Primary Shadow (idx == 0)
		if idx == 0 {
			if pkCol := r.resolveMasterPKTargetColumn(ctx, db, masterBindingID, sbID); pkCol != "" {
				pkSourceField, pkDataType := r.resolvePrimaryShadowPKMeta(ctx, db, sbID, pkCol)
				_ = db.WithContext(ctx).Exec(`
					INSERT INTO `+masterRuleTable+`
					  (master_binding_id, mapping_v2_id, target_column, source_field, data_type, is_active, status, created_by, updated_by)
					SELECT ?, NULL, ?, ?, ?, true, 'pending', ?, ?
					 WHERE NOT EXISTS (
					     SELECT 1 FROM `+masterRuleTable+` mm
					      WHERE mm.master_binding_id = ? AND lower(mm.target_column) = lower(?))
					ON CONFLICT (master_binding_id, target_column) DO NOTHING`,
					masterBindingID, pkCol, pkSourceField, pkDataType, createdBy, updatedBy, masterBindingID, pkCol,
				)
				_ = db.WithContext(ctx).Exec(`
					UPDATE `+masterRuleTable+`
					   SET source_field = COALESCE(NULLIF(btrim(source_field), ''), ?),
					       data_type    = COALESCE(NULLIF(btrim(data_type), ''), ?),
					       updated_at   = NOW()
					 WHERE master_binding_id = ?
					   AND lower(target_column) = lower(?)
					   AND mapping_v2_id IS NULL
					   AND (COALESCE(btrim(source_field), '') = '' OR COALESCE(btrim(data_type), '') = '')`,
					pkSourceField, pkDataType, masterBindingID, pkCol,
				)
			}
		}
```

Và helper `resolvePrimaryShadowPKMeta` trong cùng file `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/master/master_multi_shadow_repo_gorm.go`:

```go
func (r *masterRepoGorm) resolvePrimaryShadowPKMeta(
	ctx context.Context,
	db *gorm.DB,
	shadowBindingID int64,
	fallbackPKCol string,
) (sourceField string, dataType string) {
	sourceField = strings.TrimSpace(fallbackPKCol)
	dataType = "TEXT"
	if db == nil || shadowBindingID <= 0 {
		return sourceField, dataType
	}
	shadowTable := "cdc_system.shadow_binding"
	sourceObjTable := "cdc_system.source_object_registry"
	v2Table := "cdc_system.mapping_rule_v2"
	if os.Getenv("GO_ENV") == "test" {
		shadowTable = "shadow_binding"
		sourceObjTable = "source_object_registry"
		v2Table = "mapping_rule_v2"
	}
	var srcPK string
	_ = db.WithContext(ctx).Raw(`
		SELECT COALESCE(NULLIF(btrim(so.primary_key_field), ''), '')
		  FROM `+shadowTable+` sb
		  JOIN `+sourceObjTable+` so ON so.id = sb.source_object_id
		 WHERE sb.id = ? LIMIT 1`, shadowBindingID).Scan(&srcPK).Error
	srcPK = strings.TrimSpace(srcPK)
	if srcPK != "" {
		sourceField = srcPK
	}
	var row struct {
		TargetColumn string `gorm:"column:target_column"`
		DataType     string `gorm:"column:data_type"`
	}
	_ = db.WithContext(ctx).Raw(`
		SELECT COALESCE(NULLIF(btrim(target_column), ''), '') AS target_column,
		       COALESCE(NULLIF(btrim(data_type), ''), '')     AS data_type
		  FROM `+v2Table+`
		 WHERE shadow_binding_id = ?
		   AND COALESCE(is_active, true) = true
		   AND (lower(source_field) = lower(?) OR lower(target_column) = lower(?))
		 ORDER BY id ASC LIMIT 1`,
		shadowBindingID, sourceField, sourceField,
	).Scan(&row).Error
	if strings.TrimSpace(row.TargetColumn) != "" {
		sourceField = strings.TrimSpace(row.TargetColumn)
	}
	if strings.TrimSpace(row.DataType) != "" {
		dataType = strings.TrimSpace(row.DataType)
	}
	return sourceField, dataType
}
```

### 3.2. Xử lý `F2`: Bảo vệ `TransmuterModule.loadRules` không bao giờ drop rule PK (`_id`) khi `source_field`/`data_type` trống trong DB

**File:** `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go` (`L918-L955`)

```go
	var rules []mappingRuleRow
	err := t.systemDB.WithContext(ctx).Raw(`
		SELECT m.id, ?::bigint AS source_object_id, m.master_binding_id,
		       COALESCE(NULLIF(btrim(m.source_field), ''), v2.source_field) AS source_field,
		       m.target_column, COALESCE(NULLIF(btrim(m.data_type), ''), v2.data_type) AS data_type,
		       COALESCE(v2.source_format, '') AS source_format,
		       COALESCE(NULLIF(btrim(m.source_path), ''), v2.source_path) AS source_path,
		       COALESCE(NULLIF(m.transform_fn, ''), v2.transform_fn) AS transform_fn,
		       COALESCE(v2.is_nullable, true) AS is_nullable, v2.default_value
		  FROM cdc_system.mapping_rule_master m
		  LEFT JOIN cdc_system.mapping_rule_v2 v2 ON v2.id = m.mapping_v2_id
		 WHERE m.master_binding_id = ? AND m.is_active = true AND m.status = 'approved'
		 ORDER BY m.id`, row.SourceObjectID, row.ID).Scan(&rules).Error
	if err != nil {
		return nil, err
	}
	masterPK := strings.TrimSpace(parsePKFromSpec(row.TransformSpec))
	sourcePK := strings.TrimSpace(row.SourcePK)
	valid := rules[:0]
	for _, r := range rules {
		if masterPK != "" && strings.EqualFold(strings.TrimSpace(r.TargetColumn), masterPK) {
			if strings.TrimSpace(r.SourceField) == "" {
				if sourcePK != "" {
					r.SourceField = sourcePK
				} else {
					r.SourceField = strings.TrimSpace(r.TargetColumn)
				}
			}
			if strings.TrimSpace(r.DataType) == "" {
				r.DataType = "TEXT"
			}
		}
		if r.TransformFn != nil && !IsTransformWhitelisted(*r.TransformFn) {
```
