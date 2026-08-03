# Solution: Fix Cascade Delete Logic in DeleteShadowBinding, Audit DeleteMasterBinding, and Fix FE Error Formatting

## 1. Sửa lỗi Xoá Shadow Binding (DeleteShadowBinding)
### Vấn đề hiện tại
Khi thực hiện xoá một `shadow_binding`, handler `DeleteShadowBindingHandler` đang truy vấn và xoá toàn bộ `master_binding` và `source_object_registry` bằng `source_object_id`. 
Do các shadow bindings khác nhau của cùng một nguồn (ví dụ `hyperverge-face-match` và `hyperverge-face-match_1`) chia sẻ chung một `source_object_id`, logic này dẫn đến việc xoá nhầm toàn bộ các shadow bindings và master bindings khác đang sử dụng chung `source_object_registry` đó.

### Giải pháp kỹ thuật
1. **Kiểm tra Active Master**: Chỉ kiểm tra xem có `master_binding` nào thuộc về **chính shadow binding đang xoá** (qua `shadow_binding_id`) đang ở trạng thái active không, thay vì kiểm tra toàn bộ master bindings của `source_object_id`.
2. **Cascade Master Bindings**: Truy vấn các `master_binding` để cascade drop/delete dựa trên `shadow_binding_id = info.ID`, chứ không dùng `source_object_id`.
3. **Giữ nguyên Source Object Registry nếu còn shadow khác**: Trước khi xoá `source_object_registry`, kiểm tra xem còn `shadow_binding` nào khác đang trỏ tới `source_object_id` này không (`SELECT COUNT(1) FROM cdc_system.shadow_binding WHERE source_object_id = ? AND id != ?`). Nếu còn, tuyệt đối không xoá `source_object_registry`.

---

## 2. Audit logic Xoá Master Binding (DeleteMasterBinding)
Tôi đã kiểm tra kỹ toàn bộ logic của `DeleteMasterBindingHandler` trong [delete_master_binding.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/master/delete_master_binding.go) để đảm bảo không xảy ra vấn đề tương tự:

1. **Phạm vi định danh**: Xoá master binding được thực hiện hoàn toàn dựa trên `cmd.ID` (ID cụ thể của `master_binding` được xoá).
2. **Không xoá parent**: Logic của master binding **không** có bất kỳ câu lệnh xoá nào trỏ vào `shadow_binding` hay `source_object_registry`.
3. **Mối quan hệ khoá ngoại**: 
   - Trường `shadow_binding_id` ở bảng `master_binding` có thiết lập `ON DELETE SET NULL`. Do đó, khi xoá master binding, shadow binding hay source object registry hoàn toàn không bị ảnh hưởng.
4. **Cô lập DDL**: Bảng vật lý master là độc nhất cho mỗi master binding nhờ ràng buộc `UNIQUE (master_connection_id, master_schema, master_table)`. Việc thực hiện drop bảng master vật lý này là an toàn và cô lập tuyệt đối.
👉 **Kết luận**: Logic xoá master binding hiện tại đã chuẩn xác, cô lập và an toàn, không cần thay đổi gì thêm.

---

## 3. Hiển thị thông báo lỗi chi tiết trên Frontend (apiError.ts)
### Vấn đề hiện tại
Khi API trả về mã lỗi 409/500 có cấu trúc JSON:
```json
{
  "error": "active_master_bindings_exist",
  "detail": "Vẫn còn Master đang active, vui lòng tắt đi trước khi xoá"
}
```
Hàm `humanizeApiError` của frontend đang ưu tiên lấy trường `error` trước (`e?.response?.data?.error`), dẫn đến việc hiển thị chuỗi kỹ thuật tiếng Anh `active_master_bindings_exist` lên giao diện thay vì thông điệp thân thiện với người dùng trong `detail`.

### Giải pháp kỹ thuật
Chỉnh sửa thứ tự ưu tiên gán biến `raw` trong `humanizeApiError` ở [apiError.ts](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/utils/apiError.ts), đưa `detail` và `message` lên trước `error`.

---

## 4. Tự động dọn dẹp bản ghi cdc_table_registry mồ côi khi đăng ký mới (NEW)
### Vấn đề hiện tại
Nếu một shadow binding trước đây bị xoá không qua API handler (ví dụ: bị cascade delete từ DB khi xoá source registry trước khi cập nhật fix), bản ghi `shadow_binding` bị mất nên giao diện UI không hiển thị, nhưng bản ghi `cdc_table_registry` tương ứng vẫn còn kẹt lại trên DB.
Khi người dùng thử đăng ký lại bảng đó, saga `registry.register` thực hiện bước `register-db` chèn bản ghi vào `cdc_table_registry` sẽ ném lỗi `duplicate key value violates unique constraint` (SQLSTATE 23505) và sập luồng.

### Giải pháp kỹ thuật
Khi thực hiện đăng ký mới (`Register` trong `source_repo_gorm.go`), ta tự động kiểm tra và xoá bản ghi `cdc_table_registry` mồ côi (có cùng `target_table` nhưng không có bất kỳ `shadow_binding` nào tham chiếu) ngay trước khi chèn bản ghi mới:
```go
		// Tự động dọn dẹp bản ghi V1 mồ côi (nếu có cdc_table_registry nhưng không có shadow_binding tương ứng)
		// để tránh lỗi trùng Unique Constraint khi đăng ký lại.
		err := tx.Exec(`
			DELETE FROM cdc_system.cdc_table_registry 
			WHERE target_table = ? 
			  AND NOT EXISTS (
				  SELECT 1 FROM cdc_system.shadow_binding WHERE shadow_table = ?
			  )
		`, entry.TargetTable, entry.TargetTable).Error
		if err != nil {
			return err
		}
```

---

## Chi tiết Code cần thay đổi

### BE: [delete_shadow_binding.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/shadow/delete_shadow_binding.go)

Thay thế logic trong hàm `Handle` từ line 62 thành:

```go
func (h *DeleteShadowBindingHandler) Handle(ctx context.Context, c ports.Command) (json.RawMessage, error) {
    cmd, ok := c.(DeleteShadowBindingCommand)
    if !ok {
        return nil, errors.New("shadow-binding.delete: command type mismatch")
    }

    var info *ports.ShadowBindingInfo
    var err error

    if cmd.ID > 0 {
        info, err = h.shadowRepo.GetByID(ctx, cmd.ID)
        if err != nil {
            if !errors.Is(err, ports.ErrRecordNotFound) {
                return nil, err
            }
        }
    }

    if info == nil {
        return nil, ErrShadowBindingNotFound
    }

    if info.IsActive {
        return nil, ErrShadowBindingIsActive
    }

    sourceObjectID := info.SourceObjectID

    // Khởi tạo Transaction trên metadata DB
    tx := h.db.WithContext(ctx).Begin()
    if tx.Error != nil {
        return nil, tx.Error
    }

    // Kiểm tra xem có master binding nào thuộc shadow này đang active không
    var activeMasterCount int64
    err = tx.Raw(`
        SELECT COUNT(1) 
        FROM cdc_system.master_binding 
        WHERE shadow_binding_id = ? AND is_active = TRUE
    `, info.ID).Scan(&activeMasterCount).Error
    if err != nil {
        tx.Rollback()
        return nil, err
    }
    if activeMasterCount > 0 {
        tx.Rollback()
        return nil, ErrActiveMasterBindingsExist
    }

    // Tải thông tin source object và legacy registry ID
    obj, err := h.sourceRepo.GetByID(ctx, sourceObjectID)
    if err != nil && !errors.Is(err, ports.ErrRecordNotFound) {
        tx.Rollback()
        h.logger.Error("failed to load source object", zap.Int64("source_object_id", sourceObjectID), zap.Error(err))
        return nil, err
    }

    // 1. Cascade: xoá master bindings đi kèm thuộc shadow này (bao gồm cả drop physical master tables)
    var masterIDs []int64
    err = tx.Raw(`SELECT id FROM cdc_system.master_binding WHERE shadow_binding_id = ?`, info.ID).Scan(&masterIDs).Error
    if err != nil {
        tx.Rollback()
        h.logger.Error("cascade delete: failed to query master bindings by shadow_binding_id", zap.Int64("shadow_binding_id", info.ID), zap.Error(err))
        return nil, err
    }

    for _, masterID := range masterIDs {
        binding, err := h.masterRepo.GetByID(ctx, masterID)
        if err == nil {
            masterTableFQN := binding.MasterTable
            if binding.ShadowSchema != "" {
                masterTableFQN = binding.ShadowSchema + "." + binding.MasterTable
            }
            if err := h.publisher.PublishMasterDropTable(ctx, masterTableFQN); err != nil {
                tx.Rollback() // Rollback nếu drop bảng vật lý thất bại
                h.logger.Error("cascade delete: drop physical table failed", zap.String("master", masterTableFQN), zap.Error(err))
                return nil, fmt.Errorf("cascade_drop_master_table_failed: %w", err)
            }
        }

        if err := tx.Exec(`DELETE FROM cdc_system.mapping_rule_master WHERE master_binding_id = ?`, masterID).Error; err != nil {
            tx.Rollback()
            h.logger.Error("delete cloned rules failed", zap.Int64("master_binding_id", masterID), zap.Error(err))
            return nil, err
        }
        if err := tx.Exec(`DELETE FROM cdc_system.master_binding WHERE id = ?`, masterID).Error; err != nil {
            tx.Rollback()
            h.logger.Error("delete master binding failed", zap.Int64("master_binding_id", masterID), zap.Error(err))
            return nil, err
        }
    }

    // 2. Drop bảng vật lý shadow ở database shadow thông qua NATS gửi sang worker
    if info.ShadowSchema != "" && info.ShadowTable != "" {
        if err := h.publisher.PublishShadowDropTable(ctx, info.ShadowSchema, info.ShadowTable); err != nil {
            tx.Rollback() // Rollback nếu drop bảng vật lý shadow thất bại
            h.logger.Error("delete shadow: drop physical shadow table via NATS failed", zap.String("schema", info.ShadowSchema), zap.String("table", info.ShadowTable), zap.Error(err))
            return nil, fmt.Errorf("drop_shadow_table_failed: %w", err)
        }
    }

    // 3. Xoá legacy registry (cdc_table_registry) record
    registryID := cmd.RegistryID
    if registryID > 0 {
        if err := tx.Exec(`DELETE FROM cdc_system.cdc_table_registry WHERE id = ?`, registryID).Error; err != nil {
            tx.Rollback()
            h.logger.Error("delete legacy registry failed", zap.Int64("registry_id", registryID), zap.Error(err))
            return nil, err
        }
    }

    // 4. Xoá shadow binding
    if err := tx.Exec(`DELETE FROM cdc_system.shadow_binding WHERE id = ?`, info.ID).Error; err != nil {
        tx.Rollback()
        return nil, err
    }

    // 5. Kiểm tra xem còn shadow binding nào khác trỏ tới source_object_registry này không
    var remainingShadowCount int64
    err = tx.Raw(`
        SELECT COUNT(1) 
        FROM cdc_system.shadow_binding 
        WHERE source_object_id = ? AND id != ?
    `, sourceObjectID, info.ID).Scan(&remainingShadowCount).Error
    if err != nil {
        tx.Rollback()
        return nil, err
    }

    // Chỉ xoá source object registry nếu không còn shadow binding nào trỏ tới nó
    if remainingShadowCount == 0 {
        if err := tx.Exec(`DELETE FROM cdc_system.source_object_registry WHERE id = ?`, sourceObjectID).Error; err != nil {
            tx.Rollback()
            h.logger.Error("delete source object failed", zap.Int64("id", sourceObjectID), zap.Error(err))
            return nil, err
        }
    }

    // Commit transaction
    if err := tx.Commit().Error; err != nil {
        return nil, err
    }

    body, _ := json.Marshal(map[string]interface{}{
        "message":            "shadow binding deleted",
        "shadow_binding_id":  cmd.ID,
        "deleted_master_count": len(masterIDs),
    })
    return body, nil
}
```

### FE: [apiError.ts](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/utils/apiError.ts)

Thay đổi dòng 119-124 thành:

```typescript
  const raw =
    e?.response?.data?.detail ||
    e?.response?.data?.message ||
    e?.response?.data?.error ||
    e?.message ||
    '';
```

### BE: [source_repo_gorm.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/source/source_repo_gorm.go)

Sửa hàm `Register` (dòng 257-270) thành:

```go
func (r *sourceRepoGorm) Register(ctx context.Context, entry *sourcemodel.TableRegistry) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Tự động dọn dẹp bản ghi V1 mồ côi (nếu có cdc_table_registry nhưng không có shadow_binding tương ứng)
		// để tránh lỗi trùng Unique Constraint khi đăng ký lại.
		err := tx.Exec(`
			DELETE FROM cdc_system.cdc_table_registry 
			WHERE target_table = ? 
			  AND NOT EXISTS (
				  SELECT 1 FROM cdc_system.shadow_binding WHERE shadow_table = ?
			  )
		`, entry.TargetTable, entry.TargetTable).Error
		if err != nil {
			return err
		}

		if err := tx.Create(entry).Error; err != nil {
			return err
		}

		if r.syncer != nil {
			if err := r.syncer.SyncFromLegacyTx(ctx, tx, entry); err != nil {
				return err
			}
		}
		return nil
	})
}
```
