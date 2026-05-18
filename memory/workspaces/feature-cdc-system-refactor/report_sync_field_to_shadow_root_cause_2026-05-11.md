# Root Cause Report: "Sync Field to Shadow" chỉ thấy test_field

**Date**: 2026-05-11  
**Author**: Brain:Antigravity  
**Conversation**: 918181cd-b973-45a6-928c-d55b707bd9c2

---

## 1. Symptom (Triệu chứng)

User click "Sync Fields to Shadow" trong CMS (MappingFieldsPage) → shadow table chỉ thêm được `test_field`, 
KHÔNG có các business fields của source table như `customer`, `total`, `status`, `order_id`, v.v.

---

## 2. Evidence từ DB thực tế

```sql
-- mapping_rule_v2 cho export-jobs (id=49): CHỈ 1 row test
SELECT id, source_field, target_column, status FROM cdc_system.mapping_rule_v2 WHERE source_object_id = 49;
-- id=70, test_field_from_v2 → test_field, approved

-- mapping_rule_v2 cho market_orders (id=48): ZERO rows
SELECT id FROM cdc_system.mapping_rule_v2 WHERE source_object_id = 48;
-- (0 rows)

-- Nhưng shadow table market_orders ĐÃ CÓ business columns
SELECT column_name FROM information_schema.columns WHERE table_schema = 'shadow_market_source_v1' AND table_name = 'market_orders';
-- _id, customer, total, status, created_at, order_id + CDC system cols
```

---

## 3. Root Cause Chain

```
[FE Click "Sync Fields to Shadow"]
   ↓ POST /api/v1/source-objects/registry/:registry_id/create-default-columns
   ↓
[CMS Handler: CreateDefaultColumnsV2]
   ↓ ResolveDispatchScopeBySourceObjectID → DispatchScope{SourceTable: "market_orders", ShadowSchema: "shadow_market_source_v1", TargetTable: "market_orders"}
   ↓ Dispatch cmd CreateDefaultColumnsCommand
   ↓
[Worker: HandleCreateDefaultColumns]
   ↓ 1. ensureCDCColumnsInSchema → thêm system cols OK
   ↓ 2. GetActiveRulesBySourceTable("market_orders") ← QUERY V2 MAPPING RULES
   ↓    → JOIN source_object_registry WHERE source_object_name = 'market_orders'
   ↓    → Result: 0 rules (market_orders chưa có trong mapping_rule_v2!)
   ↓ 3. Loop 0 rules → columnsAdded = 0
   ↓
[Shadow table] chỉ có system CDC cols, không có business fields
```

**Root cause**: `mapping_rule_v2` chưa được seed với business fields của `market_orders` và `export-jobs`.

---

## 4. Tại sao mapping_rule_v2 trống?

`mapping_rule_v2` được seed bởi:
1. **`bridgeMappingRulesToV2`** trong `HandleDiscover` (khi provisioning=true + sourceID > 0)
2. **Manual add** qua CMS AddMappingModal
3. **Test fixtures** (test_field_from_v2 được add thủ công)

`market_orders` và `export-jobs` được register KHÔNG qua provisioning flow đầy đủ nên bước `discover` + `bridgeMappingRulesToV2` CHƯA chạy.

---

## 5. Giải pháp

### Option A: Discover + Bridge (Recommended for existing shadow tables)

Shadow table `market_orders` đã có business columns → Worker `HandleDiscover` scan shadow columns → tạo V1 mapping rules → `bridgeMappingRulesToV2` seed V2.

**Trigger via CMS**: Click "Scan Unmapped Fields" button → gọi `/api/introspection/scan-raw/{target_table}`

**Hoặc trigger via API**:
```bash
curl -X POST http://localhost:8083/api/v1/source-objects/{id}/scan-fields
```

### Option B: Direct SQL seed (nhanh nhất cho testing)

```sql
-- Seed mapping_rule_v2 từ shadow table columns cho market_orders (id=48)
INSERT INTO cdc_system.mapping_rule_v2 
  (source_object_id, master_binding_id, source_field, target_column, data_type, source_format, is_active, status, created_by)
SELECT 
  48, NULL, column_name, column_name,
  UPPER(REPLACE(data_type, ' ', '_')),
  'raw', true, 'approved', 'brain_seed'
FROM information_schema.columns
WHERE table_schema = 'shadow_market_source_v1' AND table_name = 'market_orders'
  AND column_name NOT LIKE '_%'  -- skip CDC meta cols
ON CONFLICT DO NOTHING;
```

### Option C: Flow đúng (long-term fix)

Đảm bảo wizard luôn chạy `discover` command trước khi "Sync Fields to Shadow":
- Step 6 (Review Proposals) → SchemaManager detect columns → create proposals
- Step 7 (Approve Proposals) → approved proposals → `mapping_rule_v2` được seed
- THEN: Step "Sync Fields to Shadow" mới có data để ADD COLUMN

---

## 6. Additional Bug Found: Worker log lỗi NUMERIC

Worker logs hiện có lỗi:
```
ERROR: invalid input syntax for type numeric: "305/2" (SQLSTATE 22P02)
```

`amount` field đang chứa fraction string "305/2" thay vì số. Cần xem xét thêm transform_fn hoặc cast logic.

---

## 7. Next Actions (Boss-gated)

1. **IMMEDIATE**: Dùng Option A (scan-fields) hoặc Option B (SQL seed) để unblock Sync Fields to Shadow
2. **SHORT-TERM**: Đảm bảo discover flow được trigger trong Registration flow  
3. **LONG-TERM**: Fix wizard để mandatory chạy discover trước khi enable "Sync Fields to Shadow"
