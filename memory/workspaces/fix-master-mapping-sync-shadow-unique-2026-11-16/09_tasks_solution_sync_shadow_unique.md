---
tier: T2
mode: standard
suffix: sync_shadow_unique
workspace: fix-master-mapping-sync-shadow-unique-2026-11-16
created_at: 2026-11-17T09:34:30+07:00
---
# 09 Tasks Solution — Khắc Phục Triệt Để Lỗi Vi Phạm Ràng Buộc Duy Nhất (`SQLSTATE 23505`) Khi "Sync Từ Shadow" Trên Trang Master Mapping

## 1. Tóm tắt Điều tra (`S2` — 100% `VERIFIED`) & Phản biện Tiền Kế Hoạch (`S2.5 / DG0`)

### 1.1. Truy vết Luồng Lỗi End-to-End (`binding_id=61`, `/masters/cdc_payment_bill/mappings?binding_id=61`)
1. **Frontend (`cdc-cms-web`):**
   - Nút **"Sync từ Shadow"** tại `REF: /Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/MasterMappingFieldsPage.tsx:369-390` (`:1024-1030`) gọi `POST /api/v1/master-mapping-rules/sync-from-shadow?master_binding_id=61`.
   - Khi API trả HTTP `500` chứa lỗi PostgreSQL `ERROR: duplicate key value violates unique constraint "ux_mapping_rule_master_target" (SQLSTATE 23505)`, hàm `humanizeApiError` tại `REF: /Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/utils/apiError.ts:53-57` tra `CONSTRAINT_HUMAN["ux_mapping_rule_master_target"]` không thấy nên trả về câu mặc định:
     > `"Bản ghi đã tồn tại (vi phạm ràng buộc duy nhất)."`
2. **Backend (`cdc-cms-service`) & Schema (`cdc_system.mapping_rule_master`):**
   - Route `POST /api/v1/master-mapping-rules/sync-from-shadow` (`REF: /Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/master/master_mapping_rule_handler.go:294-311`) → `SyncFromShadowHandler.Handle` (`REF: /Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/master/sync_from_shadow.go:27-43`) → `SyncRulesFromShadow` (`REF: /Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/master/master_mapping_rule_repo_gorm.go:248-250`) → `syncRulesFromAllShadowsTx` (`REF: /Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/master/master_multi_shadow_repo_gorm.go:209-281`).
   - Ràng buộc duy nhất thực tế trên `cdc_system.mapping_rule_master` theo `REF: /Users/trainguyen/Documents/work/data-hub/cdc-cms-service/migrations/schema/cdc_system_model/078_master_allow_many_per_shadow_field.sql:19-23`:
     - `ux_mapping_rule_master_v2 (master_binding_id, mapping_v2_id)` đã `DROP` để cho phép `1 shadow field -> N master fields`.
     - Chỉ còn duy nhất **`ux_mapping_rule_master_target ON cdc_system.mapping_rule_master (master_binding_id, target_column)`**.

### 1.2. Bốn Phát Hiện Gốc Rễ (`F1`–`F4`) & Kết Quả Cổng `DG0`
> **Kết quả `DG0` (`06_laya_summary_dg0_20261117T093350.json`):** `gate = PASS` (`trusted_rounds = []` / `UNVALIDATED` — toàn bộ kết luận dưới đây được xác minh trực tiếp bằng ground-truth mã nguồn & migration `file:line`).

- **`F1` — Bước `INSERT` trong `syncRulesFromAllShadowsTx` tìm trùng sai khóa định danh & thiếu `ON CONFLICT`:**
  - `REF: /Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/master/master_multi_shadow_repo_gorm.go:215-229`
  - `REF: /Users/trainguyen/Documents/work/data-hub/cdc-cms-service/migrations/schema/cdc_system_model/078_master_allow_many_per_shadow_field.sql:8-15`
  - Đặc tả thiết kế trong Migration `078` và hàm chuẩn `cloneAllShadowRulesTx` (`master_multi_shadow_repo_gorm.go:49-60`) ghi rõ: việc kiểm tra idempotency khi kéo rule từ Shadow phải dựa trên **`m.mapping_v2_id = v2.id`** (để không tạo lại rule mặc định khi Operator đã đổi tên `target_column` trên Master, ví dụ `billId -> bill_id`), kèm `ON CONFLICT (master_binding_id, target_column) DO NOTHING`.
  - Ngược lại, `syncRulesFromAllShadowsTx` dùng `WHERE NOT EXISTS (... m.target_column = v2.target_column)` và thiếu `ON CONFLICT` lẫn điều kiện `COALESCE(v2.is_active, true) = true`. Ngay khi Operator đổi tên bất kỳ cột nào trên `cdc_payment_bill` (`binding_id=61`), bước `INSERT` thấy tên gốc `v2.target_column` chưa tồn tại nên **chèn thêm 1 bản ghi mới trùng `mapping_v2_id` với tên cột gốc `v2.target_column`**.
- **`F2` — Hai khối `UPDATE` rename ngược (`:235-272`) tự đụng độ `ux_mapping_rule_master_target` (`SQLSTATE 23505`) và phá hỏng cấu hình đổi tên cột của Operator:**
  - `REF: /Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/master/master_multi_shadow_repo_gorm.go:235-272`
  - `REF: /Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/master/update_mapping_rule.go:21-29`
  - `REF: /Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/master/update_column_metadata.go:42-56`
  - Ở tầng Shadow (`mapping_rule_v2`), `UpdateMappingRuleCommand` không hề cho phép sửa `target_column`. Sự khác biệt `m.target_column <> v2.target_column` trên thực tế đến từ việc **Operator chủ động đổi tên cột đích ở tầng Master** (`camelCase -> snake_case`) hoặc tách `1 shadow field -> N master fields` cùng `mapping_v2_id`.
  - Ngay sau khi `F1` vừa chèn dòng `(master_binding_id=61, mapping_v2_id=X, target_column=v2.target_column)` trong cùng transaction, `resRenameNotInMaster` (`:235-248`) hoặc `resRenameInMaster` (`:253-268`) chạy `UPDATE ... SET target_column = v2.target_column WHERE m.mapping_v2_id = v2.id AND m.target_column <> v2.target_column` lên dòng cũ (`target_column = 'bill_id'`) -> **đụng trực diện dòng vừa chèn ở `F1` (hoặc đụng giữa nhiều dòng cùng `mapping_v2_id`) -> nổ `SQLSTATE 23505` trên `ux_mapping_rule_master_target`!** Hơn nữa, dù không có `F1`, 2 khối `UPDATE` này cũng sẽ ghi đè mất toàn bộ tên cột `snake_case` mà Operator đã cấu hình.
- **`F3` — Rule mới kéo từ Shadow bị gán cứng `status = 'approved'` thay vì `'pending'`, lệch hợp đồng UI & tiềm ẩn lỗi `SQLSTATE 42703` trên `TransmuterModule`:**
  - `REF: /Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/master/master_multi_shadow_repo_gorm.go:52,219`
  - `REF: /Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/MasterMappingFieldsPage.tsx:380,1029`
  - `REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go:1044`
  - `REF: /Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/master/batch_update_rules.go:42-53`
  - `TransmuterModule.loadRules` nạp mọi rule có `m.is_active = true AND m.status = 'approved'`, trong khi lệnh tạo cột vật lý `cdc.cmd.master-create` chỉ được phát khi Operator duyệt rule từ `pending -> approved` qua `BatchUpdateRulesHandler`. Chèn thẳng `status = 'approved'` (`in_master = false`) trong `SyncFromShadow` vừa trái với UI (`"status pending, cần duyệt"`) vừa khiến Transmuter ghi vào cột chưa được tạo DDL trên bảng Master đang chạy.
- **`F4` — `CONSTRAINT_HUMAN` ở Frontend thiếu khóa `ux_mapping_rule_master_target`:**
  - `REF: /Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/utils/apiError.ts:41-57`
  - Bảng ánh xạ đang chỉ có tên khóa V1 cũ `mapping_rule_target_column_key`, thiếu `ux_mapping_rule_master_target`.

---

## 2. Kiểm Tra Thiết Kế 5 Bước Trước Khi Sửa Code (Blast-Radius = 0)

1. **Nhóm chức năng & Tầng (Functional Group & Layer Routing):**
   - `cdc-cms-service`: Nhóm `master`, tầng Driven Persistence Adapter (`internal/infra/persistence/master/master_multi_shadow_repo_gorm.go` + unit test `master_repo_gorm_test.go`). Không sửa chữ ký interface `MasterRuleRepository.SyncRulesFromShadow` hay command/API layer.
   - `cdc-cms-web`: Tầng tiện ích hiển thị lỗi (`src/utils/apiError.ts`).
2. **File mẫu chuẩn hiện hữu (Canonical Pattern Reference):**
   - Đối chiếu trực tiếp `cloneAllShadowRulesTx` tại `REF: /Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/master/master_multi_shadow_repo_gorm.go:37-64` và đặc tả trong `REF: /Users/trainguyen/Documents/work/data-hub/cdc-cms-service/migrations/schema/cdc_system_model/078_master_allow_many_per_shadow_field.sql:8-23`.
3. **Đối chiếu 5 Shared Chokepoints (`CP-1..CP-5`) & Ma trận Đa hình 3 Chiều:**
   - Không chạm vào bất kỳ Chokepoint Data Plane nào (`CP-1..CP-5`).
   - Tương thích 100% cả 3 chiều đa hình:
     - *Source Engine:* Mongo, Postgres, MySQL, SFTP (đọc chung `mapping_rule_v2`).
     - *Lineage Cardinality:* `1-1`, `1 Source -> N Shadows -> M Masters` (cô lập chặt theo `master_binding_id`), `multi_shadow` (`N Shadows -> 1 Master` lặp qua `collectShadowIDsFromSpec`, khi 2 bảng Shadow có cột trùng tên thì bảng đứng trước giữ cột và bảng sau tự động bỏ qua an toàn nhờ `LOWER(TRIM(...))` + `ON CONFLICT DO NOTHING`), và `1 Shadow Field -> N Master Fields` (không bao giờ chèn lặp hay ghi đè các cột đã tách từ cùng `mapping_v2_id`).
     - *Master Engine:* Postgres, Mongo, MySQL, ClickHouse.
4. **Kiểm tra Hợp đồng Liên Service (Cross-Service Contract Check):**
   - HTTP Response của `POST /api/v1/master-mapping-rules/sync-from-shadow` giữ nguyên 100% `{"status": "synced", "synced_new": res.SyncedNew, "synced_renamed": res.SyncedRenamed}` (`synced_renamed = 0`, tương thích hoàn toàn với `MasterMappingFieldsPage.tsx:373-384` vốn chỉ đọc `res.synced_new`).
5. **Phạm vi Kiểm thử & Wiring:**
   - Không thay đổi constructor hay wiring tại `internal/server/server.go:264`.
   - Bổ sung unit test thực tế trên SQLite in-memory (`GO_ENV=test`) trong `master_repo_gorm_test.go` bao phủ đủ 4 kịch bản: (a) rule đã đổi tên `camelCase -> snake_case` trên Master (`billId -> bill_id`) không bị nhân bản và không nổ `23505`; (b) `1 shadow field -> N master fields` cùng `mapping_v2_id`; (c) `multi_shadow` có 2 bảng Shadow trùng tên cột `updatedAt` và lệch hoa/thường `BillCode` vs `billcode`; (d) rule mới ở Shadow được kéo sang với `status = 'pending'`.

---

## 3. Giải Pháp Duy Nhất Tối Ưu Nhất (Ánh Xạ 1–1 Với `F1`–`F4`) & Code Demo

### 3.1. Ánh xạ `F1..F4` → Giải pháp
| Phát hiện | Giải pháp duy nhất tối ưu | File tác động |
| :--- | :--- | :--- |
| **`F1` + `F2` + `F3`** | Chuẩn hóa `syncRulesFromAllShadowsTx`: (1) Loại bỏ 2 khối `UPDATE` cưỡng bức ghi đè `m.target_column = v2.target_column`; (2) Chỉ `INSERT` các rule `v2` có `status = 'approved'` và `COALESCE(v2.is_active, true) = true`, loại trừ cột hệ thống `ReservedShadowSystemColumns()`, kiểm tra đồng thời `NOT EXISTS (... m.mapping_v2_id = v2.id)` **và** `NOT EXISTS (... LOWER(TRIM(m.target_column)) = LOWER(TRIM(v2.target_column)))`, khử trùng nội bộ `v2.id = MIN(...)` theo `LOWER(TRIM(v2.target_column))`, đặt `status = 'pending'`, `CURRENT_TIMESTAMP`, kèm `ON CONFLICT (master_binding_id, target_column) DO NOTHING`; (3) Hỗ trợ tên bảng khi `GO_ENV == "test"` giống `cloneAllShadowRulesTx`. | `cdc-cms-service/internal/infra/persistence/master/master_multi_shadow_repo_gorm.go` + `master_repo_gorm_test.go` |
| **`F4`** | Bổ sung `ux_mapping_rule_master_target` vào `CONSTRAINT_HUMAN` để hiển thị thông báo tiếng Việt tường minh khi trùng `target_column` trong cùng Master Binding. | `cdc-cms-web/src/utils/apiError.ts` |

### 3.2. Code Demo 1 — Backend `syncRulesFromAllShadowsTx` (`cdc-cms-service/internal/infra/persistence/master/master_multi_shadow_repo_gorm.go`)

```go
func syncRulesFromAllShadowsTx(ctx context.Context, db *gorm.DB, masterBindingID, shadowBindingID int64, updatedBy string) (int64, int64, error) {
	var syncedNew int64
	shadowIDs := collectShadowIDsFromSpec(ctx, db, masterBindingID, shadowBindingID)
	if len(shadowIDs) == 0 {
		return 0, 0, nil
	}

	v2Table := "cdc_system.mapping_rule_v2"
	masterRuleTable := "cdc_system.mapping_rule_master"
	if os.Getenv("GO_ENV") == "test" {
		v2Table = "mapping_rule_v2"
		masterRuleTable = "mapping_rule_master"
	}

	reservedCols := domainShadow.ReservedShadowSystemColumns()
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, sbID := range shadowIDs {
			if sbID <= 0 {
				continue
			}
			resInsert := tx.Exec(`
				INSERT INTO `+masterRuleTable+` (
					master_binding_id, mapping_v2_id, target_column, is_active, status, created_by, updated_by, created_at, updated_at
				)
				SELECT ?, v2.id, TRIM(v2.target_column), true, 'pending', 'shadow-sync', ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
				  FROM `+v2Table+` v2
				 WHERE v2.shadow_binding_id = ?
				   AND v2.status = 'approved'
				   AND COALESCE(v2.is_active, true) = true
				   AND TRIM(COALESCE(v2.target_column, '')) <> ''
				   AND LOWER(TRIM(v2.target_column)) NOT IN (?)
				   AND v2.id = (
				       SELECT MIN(v2d.id)
				         FROM `+v2Table+` v2d
				        WHERE v2d.shadow_binding_id = v2.shadow_binding_id
				          AND v2d.status = 'approved'
				          AND COALESCE(v2d.is_active, true) = true
				          AND LOWER(TRIM(v2d.target_column)) = LOWER(TRIM(v2.target_column))
				   )
				   AND NOT EXISTS (
				       SELECT 1 FROM `+masterRuleTable+` m
				        WHERE m.master_binding_id = ?
				          AND m.mapping_v2_id = v2.id
				   )
				   AND NOT EXISTS (
				       SELECT 1 FROM `+masterRuleTable+` m
				        WHERE m.master_binding_id = ?
				          AND LOWER(TRIM(m.target_column)) = LOWER(TRIM(v2.target_column))
				   )
				ON CONFLICT (master_binding_id, target_column) DO NOTHING`,
				masterBindingID, updatedBy, sbID, reservedCols, masterBindingID, masterBindingID,
			)
			if resInsert.Error != nil {
				return resInsert.Error
			}
			syncedNew += resInsert.RowsAffected
		}
		return nil
	})
	if err != nil {
		return 0, 0, err
	}
	return syncedNew, 0, nil
}
```

### 3.3. Code Demo 2 — Frontend `CONSTRAINT_HUMAN` (`cdc-cms-web/src/utils/apiError.ts`)

```ts
  mapping_rule_target_column_key:
    'Cột target trong rule này đã được mapping rồi.',
  ux_mapping_rule_master_target:
    'Tên cột đích (target_column) này đã tồn tại trong Master Binding hiện tại.',
```
