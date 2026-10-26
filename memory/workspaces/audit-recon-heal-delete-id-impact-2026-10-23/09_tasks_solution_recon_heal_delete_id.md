# 09. Thiết Kế Giải Pháp Duy Nhất (Single Best Solution) — Khắc Phục Đứt Gãy Recon & Heal (Segment A, Segment B, 2h / 7d / 30d) Sau Bản Vá Delete / `_source_id` / `_gpay_id`

- **Workspace:** `audit-recon-heal-delete-id-impact-2026-10-23`
- **Suffix:** `recon_heal_delete_id`
- **Tier:** `T2`
- **Mode:** `standard`

---

## I. Tóm Tắt Giải Pháp Duy Nhất (Single Best Zero-Cheat Solution — Chuẩn Hóa Từ Config/Metadata Xuyên Suốt 3 Repo)

> **Nguyên tắc tối thượng (`Zero-Cheat, Strict Config/Metadata Source of Truth`):**
> 1. **TUYỆT ĐỐI CẤM đoán mò khóa chính (`PK`) bằng danh sách fallback (`resolveBusinessPKForMaster`, `_id -> id -> _source_id`):**
>    - Khóa chính của bảng **Master** bắt buộc được đọc **duy nhất** từ `Spec (JSON) *` (`cdc_system.master_binding.transform_spec->>'pk'`, ví dụ `{"pk": "_id"}`) thông qua `resolveConflictTarget(binding, colTypes)`.
>    - Để đảm bảo `transform_spec` luôn có `"pk"`, khóa chặt ngay từ tầng Config/Validation trên `cdc-cms-web` và `cdc-cms-service` (tự động lấy `primary_key_field` từ `shadow_binding` $\rightarrow$ `source_object_registry` điền mặc định vào `Spec (JSON) *` và chặn lưu nếu thiếu `"pk"`).
> 2. **TUYỆT ĐỐI CẤM hardcode `IN ('', 'default', 'default_shadow')` hoặc bắn cứng `"shadow_connection_key": "default"`:**
>    - Mọi nơi phát lệnh `cdc.cmd.transmute-shadow` (`BatchBuffer`, `SinkWorker`, `executeHealSegA`, `ReconCore.RunOrphanPrune`, `BatchTransformHandler`) đều phải truy xuất đúng bộ định danh `(binding_code, shadow_schema, shadow_connection_key)` từ `cdc_system.shadow_binding` JOIN `cdc_system.connection_registry`.
>    - Tại `MasterBindingRepo.ListMasterTargetsByShadowIdentity` (`master_binding_repo.go:110`), khớp chính xác theo `sb.binding_code = @binding_code` hoặc `scr.connection_code = @shadow_connection_key` (xóa bỏ hoàn toàn `IN ('', 'default', 'default_shadow')`).
> 3. **TUYỆT ĐỐI KHÔNG ĐỤNG CHẠM** vào thuật toán đối soát của `recon_stream_bucket_engine.go`, `recon_tier_b.go`, `recon_heal_fetch.go`, `recon_heal.go`, `schema_adapter.go`.

---

### 1. Repo 1: `centralized-data-service` (Xóa bỏ 100% Fallback/Hardcode Cheat, Truy xuất chuẩn từ Metadata DB)

- **1.1. Xóa bỏ hoàn toàn `resolveBusinessPKForMaster` (`internal/service/master/transmuter_utils.go:751-768`) và chuẩn hóa `transmuter.go`:**
  - Xóa hàm `resolveBusinessPKForMaster` trong `transmuter_utils.go`.
  - Trong `inferLastGpayIDFromMaster` (`transmuter.go:1114-1146`) và `deleteFromMasterBySourceAndGpayIDs` (`transmuter.go:1686-1728`):
    - Chỉ dùng `pkCol := resolveConflictTarget(binding, colTypes)` (đọc trực tiếp `parsePKFromSpec(binding.TransformSpec)` từ `Spec (JSON) *`).
    - Trong `deleteFromMasterBySourceAndGpayIDs(ctx, db, binding, rules, colTypes, sourceIDs, gpayIDs)`: chỉ thực hiện `DELETE FROM master WHERE "<pkCol>" IN (cleanSourceIDs)` khi `pkCol != "" && pkCol != "_gpay_id"` và `pkCol` được xác nhận map từ `binding.SourcePK` / `_source_id` trong `rules` (`mapping_rule_master`) hoặc trùng `binding.SourcePK` khi `copy_1_to_1`.
  - Trong `loadMaster` (`transmuter.go:849, 853, 856`) và `master_ddl_generator.go:479`: đổi `COALESCE(sor.primary_key_field, 'id')` $\rightarrow$ `COALESCE(sor.primary_key_field, '')`, và `COALESCE(mc.connection_code, 'default')` / `COALESCE(sc.connection_code, 'default')` $\rightarrow$ `COALESCE(..., '')`.
  - Giữ nguyên logic đúng đã khắc phục Regression 1:
    - Không bao giờ sinh `_gpay_id` bằng `deterministicSourceGpayID` (SHA-256) trong `RunEvents` khi `copy_1_to_1` (`transmuter.go:774`).
    - Xóa dòng mang `_gpay_id` SHA-256 cũ (`staleLegacyGpayIDs`) trước khi `bulkUpsertMaster` (`transmuter.go:781-794, 1288-1293, 1784`).
    - Bảo vệ dòng `row.Deleted == false` khỏi tombstone cũ cùng `_source_id` trong `Run` và `processBatch` (`transmuter.go:404-418, 1226-1236`).

- **1.2. Tách bạch khóa chính Master (`Spec JSON`) và khóa chính Shadow (`_source_id`) trong `internal/handler/recon/recon_execute_heal_handler.go`:**
  - Thay thế `resolvePKColumnAndType` (`dòng 690-723`) bằng `resolveMasterPKFromSpec(ctx, masterDB, ref) (pkCol string, pkType string)`:
    - Đọc `pkCol` **duy nhất** từ `ref.TransformSpec` (`{"pk": "..."}`). Nếu `ref == nil` hoặc `pkCol == ""` hoặc `pkCol == "_gpay_id"`, trả về `("", "")` — **tuyệt đối không fallback sang `entry.PrimaryKeyField` hay `"_id"`**!
    - Tra `pkType` từ `cdc_system.mapping_rule_master` (`WHERE master_binding_id = ref.ID AND target_column = pkCol`) hoặc `information_schema.columns` trên `masterDB`.
  - Tại `resolveSourceIDsForSegmentB` (`dòng 875`, khi `effectiveIDType != "gpay"` và cần tra bảng Shadow để lấy `_source_id`): truy vấn trực tiếp trên cột `"_source_id"` của bảng Shadow (`SELECT "_source_id" FROM shadow WHERE "_source_id" IN (?) AND NOT COALESCE("_deleted", false)`), không lấy tên cột Master tra vào bảng Shadow!

- **1.3. Xóa bỏ `IN ('', 'default', 'default_shadow')` tại `master_binding_repo.go:110` và chuẩn hóa cả 5 nơi bắn `cdc.cmd.transmute-shadow`:**
  - Trong `MasterBindingRepo.ListMasterTargetsByShadowIdentity` (`internal/repository/master/master_binding_repo.go:92-125`):
    - Xóa bỏ `IN ('', 'default', 'default_shadow')` và `COALESCE(..., 'default')`.
    - Khớp chính xác:
      ```sql
      AND (
        (@binding_code <> '' AND sb.binding_code = @binding_code)
        OR
        (@binding_code = '' AND sb.shadow_table = @shadow_table
          AND COALESCE(sb.shadow_schema, '') = COALESCE(NULLIF(@shadow_schema, ''), COALESCE(sb.shadow_schema, ''))
          AND (@shadow_connection_key = '' OR scr.connection_code = @shadow_connection_key))
      )
      ```
  - Tại cả **5 nơi bắn `cdc.cmd.transmute-shadow`**, truy xuất đúng `(shadow_binding_code, shadow_schema, shadow_connection_key)` từ `cdc_system.shadow_binding sb LEFT JOIN cdc_system.connection_registry cr ON cr.id = sb.shadow_connection_id`:
    1. `BatchBuffer.hasPostIngestSchedule` & `publishTransmuteTrigger` (`internal/handler/shadow/batch_buffer_fanout.go:74-82, 140-169`): mở rộng câu SQL trong `hasPostIngestSchedule` để `LEFT JOIN cdc_system.connection_registry cr ON cr.id = sb.shadow_connection_id`, lưu `postIngestMeta{enabled, bindingCode, shadowSchema, shadowConnKey}` vào `piCache`, và gửi đúng `shadow_binding_code`, `shadow_schema`, `shadow_connection_key` trong `publishTransmuteTrigger`.
    2. `SinkWorker.hasPostIngestSchedule` & `publishTransmuteTrigger` (`internal/sinkworker/worker.go:369-425`): tương tự `BatchBuffer`, lấy và cache `(enabled, bindingCode, shadowSchema, shadowConnKey)` từ `shadow_binding` JOIN `connection_registry`.
    3. `ReconHandler.publishTransmuteShadowForSegA` (`internal/handler/recon/recon_execute_heal_handler.go:339-392`): tra `(binding_code, shadow_schema, connection_code)` từ `cdc_system.shadow_binding` JOIN `connection_registry` theo bảng Shadow trước khi publish.
    4. `ReconCore.RunOrphanPrune` (`internal/service/recon/recon_tier_a.go:580-590`): tra `(binding_code, shadow_schema, connection_code)` từ `cdc_system.shadow_binding` JOIN `connection_registry` trên `c.ConfigDB`.
    5. `BatchTransformHandler.publishTransmuteTrigger` (`internal/handler/shadow/batch_transform_handler.go:588-618`): tra `(binding_code, shadow_schema, connection_code)` từ `cdc_system.shadow_binding` JOIN `connection_registry` trên `h.MasterDB`.

- **1.4. Bổ sung `ShadowConnectionKey` & `ShadowBindingCode` vào `MasterBindingRef` (`recon_engine_segment_b.go:19-33, 77-99`) và chuẩn hóa `ConnectionManager.GetMasterDB` / `GetShadowDB` (`connection_manager.go:53-129`):**
  - Trong `ListActiveMasterBindings` (`recon_engine_segment_b.go:77-99`): select thêm `COALESCE(cr_sh.connection_code, '') AS shadow_connection_key` và `COALESCE(sb.binding_code, '') AS shadow_binding_code` vào `MasterBindingRef`.
  - Trong `ConnectionManager.GetMasterDB` (`connection_manager.go:60-85`) và `GetShadowDB` (`connection_manager.go:90-129`): khi bản ghi active trong `connection_registry` có `conn.Host == ""` (đại diện cho DB mặc định của role đó như `default_master`), phân giải về `m.reg.GetDB(database.RoleDestination)` (hoặc `RoleShadow`) và lưu vào `masterDBPool[key]` / `shadowDBPool[key]` để không query lại DB hay bắn log `WARN` giả.

---

### 2. Repo 2: `cdc-cms-service` (Cưỡng chế `Spec (JSON) *` luôn có `{"pk": "..."}` & Cung cấp đủ Metadata cho UI)

- **2.1. Bắt buộc và tự động điền `"pk"` trong `transform_spec` khi tạo/sửa/auto-provision `master_binding`:**
  - `internal/app/commands/master/create_master.go:64-79, 114-117`: trong `CreateMasterHandler.Handle`, lấy `primary_key_field` từ `shadow_binding` $\rightarrow$ `source_object_registry`; nếu `cmd.Spec` chưa có `"pk"`, tự động bơm `"pk": primaryKeyField` vào `spec`. Nếu sau đó `"pk"` vẫn rỗng thì trả lỗi validation `400`.
  - `internal/app/commands/master/update_master_spec.go:38-56` & `internal/infra/persistence/master/master_repo_gorm.go:409-418`: trong `UpdateMasterSpecCommand.Validate()`, bắt buộc JSON `Spec` phải chứa `"pk"` là chuỗi khác rỗng; trong `masterRepoGorm.UpdateSpec`, thêm `AND schema_status <> 'approved'`.
  - `internal/infra/persistence/governance/provisioning_orchestrator.go:379-389, 447-471` (`seedMasterBindingForAdvance`): `LEFT JOIN cdc_system.source_object_registry sor ON sor.id = sb.source_object_id` và ghi `jsonb_build_object('pk', COALESCE(NULLIF(btrim(sor.primary_key_field), ''), '_id'))` vào `transform_spec` thay vì `'{}'::jsonb`.
- **2.2. Chuẩn hóa `ApproveSchemaTx`, `CloneMappingRules`, `ResolveMasterConnection` (`master_repo_gorm.go:315-334, 476-489, 502-543`) và `EnsureDefaultMasterConnection` (`bootstrap/master_connection.go:11-28`):**
  - Trong `ApproveSchemaTx` và `CloneMappingRules`: ưu tiên đọc PK từ `NULLIF(btrim(mb.transform_spec->>'pk'), '')` rồi mới tới `NULLIF(btrim(so.primary_key_field), '')`.
  - Trong `ResolveMasterConnection`: khi truyền `code` tường minh mà không tìm thấy trong `connection_registry` thì trả lỗi `gorm.ErrRecordNotFound`; ở nhánh mặc định chỉ lọc `role_type IN ('master', 'destination', 'dest', 'mixed')` (xóa bỏ `OR engine_type IN ('postgres', 'postgresql')`).
  - Trong `EnsureDefaultMasterConnection` (`bootstrap/master_connection.go` & `cmd/server/main.go`): truyền `(host, port, dbName)` của Master DB để cập nhật đầy đủ vào `default_master` giống hệt `EnsureDefaultShadowConnection`.
- **2.3. Trả thêm `shadow_connection_code` và `primary_key_field` trong `GET /api/v1/shadow-bindings` (`source_object_read_repo_gorm.go:410-495` & `source_read_port.go:93-110`):**
  - `LEFT JOIN cdc_system.connection_registry shd_cn ON shd_cn.id = sb.shadow_connection_id` để trả về `shadow_connection_code` và `primary_key_field` cho `cdc-cms-web`.

---

### 3. Repo 3: `cdc-cms-web` (Xóa bỏ Hardcode trên UI, Đồng bộ `Spec (JSON) *` và `connection_code` từ API Metadata)

- **3.1. `src/pages/MasterRegistry.tsx`:**
  - Khi người dùng chọn bảng Shadow (`L1303-1315`) hoặc đổi `transform_type` (`L1401-1407`), tự động đọc `selectedShadow.primary_key_field` từ API `/api/v1/shadow-bindings` để cập nhật trường `"pk"` trong `form.spec` (ví dụ `{"pk": "transaction_id"}` cho bảng SFTP, `{"pk": "_id"}` cho bảng MongoDB).
  - Khi bấm Submit ở cả Modal Tạo mới (`L771`) và Modal Sửa Spec (`L1602`), kiểm tra chặt `typeof parsed.pk === 'string' && parsed.pk.trim() !== ''`; nếu thiếu `"pk"` thì hiển thị `message.error('Spec (JSON) bắt buộc phải có trường "pk" (ví dụ: {"pk":"_id"})')` và chặn gửi request.
  - Ở dropdown chọn Master Connection (`L291, 591, 1364-1377`), dùng trực tiếp `connection_code` từ `targetMasterConnections` (`default_master`, `master_2`, ...), xóa bỏ option hardcode `'default'`.
- **3.2. `src/pages/ActivityManager.tsx`:**
  - Tại các dòng `629, 739, 893`, lấy `sb.shadow_connection_code` từ dữ liệu `/api/v1/shadow-bindings` thay vì gán cứng `'default_shadow'`.

---

## II. Code Demo Chi Tiết (Chuẩn Bị Cho DG1 / DG2)

### Demo 1: `centralized-data-service` — Xóa bỏ `resolveBusinessPKForMaster` và chỉ đọc PK của Master từ `Spec (JSON) *` (`resolveConflictTarget`)
```go
// Trong internal/service/master/transmuter.go (deleteFromMasterBySourceAndGpayIDs):
pkCol := resolveConflictTarget(binding, colTypes)
if len(cleanSourceIDs) > 0 && pkCol != "" && pkCol != "_gpay_id" && isMasterPKMappedFromSourcePK(binding, rules, pkCol) {
	pkType := colTypes[pkCol]
	if isIntegerColumnType(pkType) {
		var intIDs []int64
		for _, s := range cleanSourceIDs {
			if v, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64); err == nil {
				intIDs = append(intIDs, v)
			}
		}
		if len(intIDs) > 0 {
			res := db.WithContext(ctx).Exec(fmt.Sprintf(`DELETE FROM %s WHERE %s IN ?`, masterTable, quoteIdent(pkCol)), intIDs)
			if res.Error != nil {
				return 0, res.Error
			}
			totalAffected += res.RowsAffected
		}
	} else {
		res := db.WithContext(ctx).Exec(fmt.Sprintf(`DELETE FROM %s WHERE %s IN ?`, masterTable, quoteIdent(pkCol)), cleanSourceIDs)
		if res.Error != nil {
			return 0, res.Error
		}
		totalAffected += res.RowsAffected
	}
}
```

### Demo 2: `centralized-data-service` — Truy xuất `(binding_code, shadow_schema, shadow_connection_key)` từ `cdc_system.shadow_binding` JOIN `connection_registry` và khớp chính xác trong `MasterBindingRepo`
```go
// Trong internal/repository/master/master_binding_repo.go (ListMasterTargetsByShadowIdentity):
sql := `
	SELECT
		mb.id,
		mb.binding_code,
		COALESCE(mb.master_schema, 'public') AS master_schema,
		mb.master_table,
		COALESCE(mcr.connection_code, '') AS master_connection_key
	FROM cdc_system.master_binding mb
	JOIN cdc_system.shadow_binding sb ON sb.id = mb.shadow_binding_id
	LEFT JOIN cdc_system.connection_registry scr ON scr.id = sb.shadow_connection_id
	LEFT JOIN cdc_system.connection_registry mcr ON mcr.id = mb.master_connection_id
	WHERE mb.is_active = true
	  AND mb.schema_status = 'approved'
	  AND (
	    (@binding_code <> '' AND sb.binding_code = @binding_code)
	    OR
	    (@binding_code = '' AND sb.shadow_table = @shadow_table
	      AND COALESCE(sb.shadow_schema, '') = COALESCE(NULLIF(@shadow_schema, ''), COALESCE(sb.shadow_schema, ''))
	      AND (@shadow_connection_key = '' OR scr.connection_code = @shadow_connection_key))
	  )
`
```

### Demo 3: `cdc-cms-service` & `cdc-cms-web` — Cưỡng chế `Spec (JSON) *` luôn có `{"pk": "..."}` từ `source_object_registry.primary_key_field`
```go
// Trong cdc-cms-service/internal/app/commands/master/update_master_spec.go:
func (c UpdateMasterSpecCommand) Validate() error {
	if c.MasterID <= 0 {
		return fmt.Errorf("%w: master_id must be positive", domain.ErrValidation)
	}
	if len(c.Spec) == 0 || !json.Valid(c.Spec) {
		return fmt.Errorf("%w: spec must be valid JSON", domain.ErrValidation)
	}
	var parsed map[string]any
	if err := json.Unmarshal(c.Spec, &parsed); err != nil {
		return fmt.Errorf("%w: spec must be a JSON object", domain.ErrValidation)
	}
	pk, _ := parsed["pk"].(string)
	if strings.TrimSpace(pk) == "" {
		return fmt.Errorf("%w: spec must contain a non-empty \"pk\" field (e.g. {\"pk\":\"_id\"})", domain.ErrValidation)
	}
	return nil
}
```

---

## 3. Khắc phục từ Vòng Lặp Phản Tỉnh `S5 -> S3 -> S4 -> S5` (Adversarial Self-Healing Remediation)

Qua rà soát phản biện từng dòng `git diff` và chạy truy vấn đối chứng trực tiếp trên container PostgreSQL thật (`gpay-postgres-cdc`), phát hiện và xử lý dứt điểm các điểm sau:

1. **Sửa lỗi cột không tồn tại `mb.shadow_connection_id` (`SQLSTATE 42703` trên PostgreSQL thật):**
   - Bảng `cdc_system.master_binding` (`mb`) trên PostgreSQL thật chỉ có `shadow_binding_id` và `master_connection_id`, **không có cột `shadow_connection_id`** (cột `shadow_connection_id` nằm trên `cdc_system.shadow_binding sb`).
   - Sửa `COALESCE(mb.shadow_connection_id, sb.shadow_connection_id)` thành `sb.shadow_connection_id` tại cả 3 nơi:
     - `centralized-data-service/internal/repository/master/master_binding_repo.go:102` (`ListMasterTargetsByShadowIdentity`)
     - `centralized-data-service/internal/handler/shadow/batch_buffer_fanout.go:180` (`resolvePostIngestSchedule`)
     - `centralized-data-service/internal/sinkworker/worker.go:436` (`resolvePostIngestSchedule`)
   - Đồng thời tại `master_binding_repo.go:83` (`ListMasterTargetsByShadowTable`), đổi `COALESCE(cr.connection_code, 'default')` thành `COALESCE(cr.connection_code, '')`.
2. **Xóa bỏ schema SQLite giả lập trong `batch_buffer_fanout_test.go` và bổ sung Integration Test chạy thẳng trên PostgreSQL thật (`gpay-postgres-cdc`):**
   - Trong `centralized-data-service/internal/handler/shadow/batch_buffer_fanout_test.go`, xóa bỏ cột `shadow_connection_id` bịa đặt trên bảng `master_binding` và bổ sung bài test tích hợp kết nối trực tiếp vào PostgreSQL thật (`gpay-postgres-cdc` `:5433`) kiểm chứng cả `MasterBindingRepo.ListMasterTargetsByShadowIdentity`, `BatchBuffer.resolvePostIngestSchedule`, và `ReconCore.ResolveShadowBindingIdentity` đối với các binding thật (`payment_bills`, `payment_bills_1`, `export_jobs`).
3. **Loại bỏ các fallback còn sót và khôi phục comment gốc:**
   - Trong `cdc-cms-service/internal/infra/persistence/governance/provisioning_orchestrator.go:383, 435-446`: bỏ fallback `'_id'` trong `COALESCE(NULLIF(btrim(sor.primary_key_field), ''), '_id')`, kiểm tra nếu `src.PrimaryKeyField == ""` thì trả lỗi cấu hình rõ ràng; khôi phục đầy đủ 11 dòng comment giải thích `ON CONFLICT (binding_code)`.
   - Trong `cdc-cms-web/src/pages/MasterRegistry.tsx:1336, 1442`: bỏ fallback `|| '_id'`, chỉ lấy `b?.primary_key_field?.trim() || ''` từ cấu hình `source_object_registry` (nếu nguồn chưa cấu hình `primary_key_field` thì để người dùng nhập tường minh và chặn tại bước validate).
   - Trong `cdc-cms-web/src/pages/ActivityManager.tsx:629, 701, 740, 894`: bỏ fallback `'default_shadow'` / `'default_master'`, hiển thị chính xác `shadow_connection_code` / `master_connection_code` trả về từ API.


