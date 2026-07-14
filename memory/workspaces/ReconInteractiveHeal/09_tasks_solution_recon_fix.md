# Hồ sơ Giải pháp Kỹ thuật - Di chuyển resolveMasterBindingRef sang ReconBase

Tài liệu này chứa các thay đổi chi tiết đối với mã nguồn Go để di chuyển helper `resolveMasterBindingRef` sang `ReconBase` nhằm đảm bảo tính nhất quán của thiết kế đối xứng (Symmetric Design).

## 1. recon_base_handler.go

Thêm helper `resolveMasterBindingRef` ngay sau `resolveTargetTableConfig`.

```diff
 func (h *ReconBase) resolveTargetTableConfig(targetTable string) *source.TableRegistry {
 	if h.metadata != nil {
 		if item := h.metadata.GetTableConfig(targetTable); item != nil {
 			return item
 		}
 		if !strings.HasPrefix(targetTable, ShadowPrefix) {
 			if item := h.metadata.GetTableConfig(ShadowPrefix + targetTable); item != nil {
 				return item
 			}
 		}
 		if item := h.metadata.GetTableConfigBySource(targetTable); item != nil {
 			return item
 		}
 	}
 	if h.registryRepo == nil {
 		return nil
 	}
 	v1, err := h.registryRepo.GetByTargetTable(context.Background(), targetTable)
 	if err == nil {
 		return v1
 	}
 	return nil
 }
 
+func (h *ReconBase) resolveMasterBindingRef(ctx context.Context, masterTable string) *servicerecon.MasterBindingRef {
+	for _, ref := range h.reconCore.ListActiveMasterBindings(ctx) {
+		if ref.MasterTable == masterTable || ref.MasterRel() == masterTable {
+			return &ref
+		}
+	}
+	return nil
+}
```

## 2. recon_check_handler.go

Xóa phương thức `resolveMasterBindingRef` khỏi `CheckHandler`.

```diff
-func (h *CheckHandler) resolveMasterBindingRef(ctx context.Context, masterTable string) *servicerecon.MasterBindingRef {
-	for _, ref := range h.reconCore.ListActiveMasterBindings(ctx) {
-		if ref.MasterTable == masterTable || ref.MasterRel() == masterTable {
-			return &ref
-		}
-	}
-	return nil
-}
```
