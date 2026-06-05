# 02_plan_mapping_master_redesign_2026-06-04.md — Redesign mapping_rule_master (lean, link mapping_v2) + bỏ Flatten-shadow

> **Agent**: Muscle:Claude-Opus-4.8 | **Ngày**: 2026-06-04
> **Nguồn**: User feedback (4 vấn đề). #3a + #3b đã thực thi; #1 + #2 plan dưới đây (đụng 2 query worker → cần làm cẩn thận, không gấp).

---

## ĐÃ THỰC THI (turn này)
- **#3a — db→shadow vỡ (42601)**: `create_mapping_rule.go:162` INSERT 18 cột nhưng VALUES chỉ 15 `?` + 2 NOW() = 17 → thêm 1 `?` (16 `?`+2 NOW=18). **Bug committed sẵn, KHÔNG do phiên này**. ✅ build CMS=0.
- **#3b — cdc_internal hồi sinh**: `test/.../approve_schema_proposal_integration_test.go` chạy `CREATE SCHEMA cdc_internal` (đã DROP ở migration 038) → đổi sang `shadow_e2e`. ✅ Giữ comment lịch sử migration 037. (smoke_failover.sh còn default `cdc_internal.shadow_test_users` — TODO nhỏ, overridable.)

---

## #1 — Redesign `mapping_rule_master` (KHÔNG copy field, link mapping_v2_id, JOIN lấy field)

### Nguyên tắc
Master rule = "chọn **rule nghiệp vụ** nào của shadow (`mapping_rule_v2`) đưa sang master này" + tên cột master + trạng thái duyệt. **KHÔNG copy** source_field/data_type/sensitive — lấy qua JOIN `mapping_rule_v2`. Filter nguồn theo **`shadow_binding_id`** (đúng nhất), không phải copy toàn bộ.

### A. Migration `075_redesign_mapping_rule_master.sql` (074 chưa release → DROP+CREATE an toàn)
```sql
BEGIN;
DROP TABLE IF EXISTS cdc_system.mapping_rule_master CASCADE;
CREATE TABLE cdc_system.mapping_rule_master (
  id                BIGSERIAL PRIMARY KEY,
  master_binding_id BIGINT NOT NULL REFERENCES cdc_system.master_binding(id) ON DELETE CASCADE,
  mapping_v2_id     BIGINT NOT NULL REFERENCES cdc_system.mapping_rule_v2(id) ON DELETE CASCADE, -- NEW
  target_column     VARCHAR(255) NOT NULL,            -- tên cột master (default = v2.target_column)
  is_active         BOOLEAN NOT NULL DEFAULT TRUE,
  status            VARCHAR(32) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','approved','rejected')),
  notes             TEXT,
  created_by        VARCHAR(100),
  updated_by        VARCHAR(100),
  created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX ux_mrm_v2     ON cdc_system.mapping_rule_master(master_binding_id, mapping_v2_id);
CREATE UNIQUE INDEX ux_mrm_target ON cdc_system.mapping_rule_master(master_binding_id, target_column);
CREATE INDEX idx_mrm_binding      ON cdc_system.mapping_rule_master(master_binding_id);
COMMIT;
```
**BỎ**: source_field, source_path, data_type, source_data_type, **source_format**, transform_fn, is_nullable, default_value, is_sensitive, mask_strategy.
**Giải thích "source_format là gì"**: là cách extract (raw/jsonpath/expression) — thuộc tính của rule nghiệp vụ ở `mapping_rule_v2`. Master KHÔNG redefine → BỎ, lấy qua JOIN.

### B. `create_master.go` clone (filter theo shadow_binding_id, KHÔNG copy)
Thay block clone hiện tại (sao chép từng field) bằng INSERT...SELECT link:
```go
shadowBindingID := shadow.ShadowBindingID // có sẵn từ resolve
h.db.WithContext(ctx).Exec(`
  INSERT INTO cdc_system.mapping_rule_master
    (master_binding_id, mapping_v2_id, target_column, is_active, status, created_by, updated_by)
  SELECT ?, v2.id, v2.target_column, true, 'approved', ?, ?
    FROM cdc_system.mapping_rule_v2 v2
   WHERE v2.shadow_binding_id = ?           -- filter ĐÚNG theo shadow_binding_id
     AND v2.master_binding_id IS NULL
     AND v2.is_active = true
     AND lower(v2.target_column) NOT IN (<17 system cols blacklist>)
  ON CONFLICT (master_binding_id, mapping_v2_id) DO NOTHING`,
  masterBindingID, cmd.UpdatedBy, cmd.UpdatedBy, shadowBindingID)
```

### C. GET List `master-mapping-rules?master_binding_id=X` (JOIN, không copy)
```sql
SELECT m.id, m.master_binding_id, m.mapping_v2_id, m.target_column, m.is_active, m.status, m.notes,
       v2.source_field, v2.source_path, v2.data_type, v2.source_data_type, v2.source_format,
       v2.transform_fn, v2.is_nullable, v2.default_value
  FROM cdc_system.mapping_rule_master m
  JOIN cdc_system.mapping_rule_v2 v2 ON v2.id = m.mapping_v2_id
 WHERE m.master_binding_id = ? [AND m.status=? AND m.is_active=?]
 ORDER BY m.target_column
```
→ FE nhận field nghiệp vụ ĐÚNG từ v2, master chỉ giữ target_column/status/is_active.

### D. WORKER (⚠️ rủi ro — đụng transmute vừa fix): JOIN mapping_rule_v2
- `transmuter.go loadRules`:
```sql
SELECT m.id, ?::bigint AS source_object_id, m.master_binding_id,
       v2.source_field, m.target_column, v2.data_type, v2.source_format, v2.source_path,
       v2.transform_fn, v2.is_nullable, v2.default_value
  FROM cdc_system.mapping_rule_master m
  JOIN cdc_system.mapping_rule_v2 v2 ON v2.id = m.mapping_v2_id
 WHERE m.master_binding_id = ? AND m.is_active = true AND m.status = 'approved'
 ORDER BY m.id
```
- `master_ddl_generator.go` (query đọc mapping_rule_master ~:74): JOIN tương tự để lấy target_column + v2.data_type.

### E. Domain/Repo/Handler/FE
- `domain/mapping/master_rule.go`: MasterRule bỏ các field đã drop, thêm `MappingV2ID int64`; thêm field "read-only từ join" (SourceField/DataType/... chỉ để serialize ra FE).
- `master_mapping_rule_repo_gorm.go`: rewrite masterRuleRow + List/Save/BatchSave (Save giờ INSERT (master_binding_id, mapping_v2_id, target_column, is_active, status) ON CONFLICT (master_binding_id, mapping_v2_id)).
- `master_mapping_rule_handler.go`: CreateOrUpdateMasterRuleRequest đổi (mapping_v2_id thay source_field/data_type/sensitive/mask).
- FE `MasterMappingFieldsPage.tsx`: bỏ cột Sensitive/Mask Strategy; source_field/data_type hiển thị read-only (từ join); Add rule chọn từ danh sách v2 chưa map.

---

## #2 — Bỏ "Scan Array (Flatten)" khỏi Master Mapping (sai quy tắc: master KHÔNG đụng shadow)
- **BE**: xoá route `admin.Post("/v1/master-mapping-rules/flatten")` (router.go:405); xoá method `Flatten` + helper `discoverJsonPaths/extractPaths/normalizeTargetColumn/sanitizeIdentifier` + field `shadowDB` khỏi `MasterMappingRuleHandler` + bỏ inject shadowDB ở server.go (master handler KHÔNG còn chạm shadow DB).
- **FE**: xoá nút "Scan Array (Flatten)" + flatten modal + `handleFlattenJson` + `jsonFieldOptions` + state `flattenOpen/sourceFieldToFlatten/flattening`.
- Lý do: flatten (bóc JSON array) là việc ở tầng **shadow mapping (mapping_rule_v2)**; master chỉ chọn lại rule đã có. Master query shadow sample = vi phạm ranh giới.

---

## Verification (sau khi thực thi #1+#2)
1. Build worker + CMS = 0; FE tsc+build = 0.
2. Restart worker + CMS.
3. db→shadow: POST /api/mapping-rules tạo rule shadow OK (201, hết 42601).
4. Tạo master `copy_1_to_1` mới → GET master-mapping-rules trả field từ JOIN v2 (source_field/data_type đúng, KHÔNG có sensitive/mask) → approve → transmute ghi master (scanned>0, inserted>0).
5. Flatten: route trả 404, FE không còn nút.
6. Worker test service+handler PASS (no regression).

## Rủi ro & lý do làm cẩn thận
- D (worker JOIN) đụng đúng 2 query transmute vừa ổn định qua nhiều vòng debug → SAI 1 chỗ là vỡ lại. Phải build+live-verify transmute sau mỗi thay đổi worker.
- Migration DROP+CREATE: an toàn vì 074 chưa release; master test (sssss) sẽ mất master rules → re-clone khi tạo master mới (hoặc re-approve).
