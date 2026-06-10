# 07_audit_full_feature_2026-06-05.md — AUDIT TOÀN BỘ feature (read-only, không commit)

> **Agent**: Muscle:Claude-Opus-4.8 | 2026-06-05 | 3 sub-agent audit song song (FE/CMS/Worker) + verify LIVE (services+DB). Bằng chứng file:line/exit-code. KHÔNG sửa code, KHÔNG commit.

## Build & Services (THẬT)
| Lane | Build | Services live |
|------|-------|---------------|
| FE cdc-cms-web | tsc=0, vite build=0 | :5173 up |
| CMS cdc-cms-service | go build=0, vet=0 | :8083 health=200 |
| Worker centralized-data-service | go build=0, test PASS | :8082 up |

## ✅ Đã đúng & đủ (verify file:line)
- **FE**: /masters Sync Modal 3-mode + flatten + Mappings nav ✅; /masters/:id/mappings 11 cột (gồm Status of Shadow, In Shadow, In Master) + **gate disable approve khi chưa in_shadow/shadow chưa approved** ✅ + In Master dùng flag `record.in_master` (bỏ endpoint hỏng) ✅ + Scan Array modal review→promote ✅ + Sync from shadow ✅ + KHÔNG còn Sensitive/Mask ✅; /schedules menu hiện + Run now + **Delete** + toggle ✅.
- **CMS**: create_mapping_rule INSERT 18 cột/16?+2NOW (hết 42601) ✅ + resolveScope theo shadow_schema/shadow_table ✅; master-mapping-rules List JOIN v2 ✅, Save default pending ✅, Delete ✅, BatchUpdate→triggerMasterDDL ✅, SyncFromShadow pending ✅; schedules routes đủ (Create/Toggle/RunNow/**Delete**) + RunNow set ScheduleID+last_status='running' ✅; migration 075 link mapping_v2_id ✅; isSystemColumn chỉ block `_`-infra ✅; domain MasterRule (mapping_v2_id/in_master/shadow_status, no sensitive/mask) ✅.
- **Worker**: loadRules JOIN v2 + `?::bigint` ✅; ShadowPK hardcode `_gpay_id` ✅; upsertMaster OCC guard `_source_ts` (GAP-02) ✅; degraded guard ✅; activity log transmute ✅; publishCompleted schedule_id ✅; master DDL **không `_raw_data`** ✅; **RLS ENABLE mọi schema** (GAP-01 A) ✅; tx Apply **lock_timeout/statement_timeout** ✅; scan-array `h.shadowDB` ✅; post_ingest gate nguyên ✅.
- **🟢 source→shadow NGUYÊN VẸN** (agent worker xác nhận chi tiết): shadow upsert (`upsert.go` OCC `>` strict, `_raw_data` vẫn ở shadow), kafka_consumer, shadow DDL/alter, mapping_rule_v2 ingestion — KHÔNG bị transmute work đụng. KHÔNG flag đỏ.

## 🔴 BUG còn lại (audit phát hiện — CHƯA fix)
| # | Mức | File:line | Vấn đề |
|---|-----|-----------|--------|
| B1 | 🔴 | `create_master.go:187` | Clone lúc tạo master hardcode `status='approved'` — **issue #2 chưa đóng**: master rule auto-approved, bỏ qua bước operator duyệt master. Phải `'pending'` (SyncFromShadow đã pending; approve_master clone 'approved' là đúng vì chạy lúc approve). |
| B2 | 🔴 | `MasterMappingFieldsPage.tsx` Create Manual Mapping modal (~580-623) | Modal chỉ có Shadow Rule + Target Column, **THIẾU field `transform_fn`** (GAP-06 chưa trọn) — interface có nhưng UI/payload không gửi. |

## 🟡 Gap nhỏ / hardening (documented)
| # | File:line | Ghi chú |
|---|-----------|---------|
| G1 | `master_mapping_rule_handler.go MasterColumns` (h.db) | Endpoint query control-plane (5433) trong khi master ở dest (5434) → trả rỗng. NHƯNG FE đã bỏ dùng (xài flag in_master) → **dead code vô hại**; nên xoá endpoint cho sạch. |
| G2 | `create_mapping_rule.go:208` path-1 | Khi có cả source_object_id + shadow_schema/shadow_table → path-1 bỏ qua shadow_schema/table (pick newest binding) → có thể chọn nhầm binding (silent). |
| G3 | `master_mapping_rule_handler.go:94` Save | Không set created_by/updated_by từ JWT → NULL audit trail. |
| G4 | dest b3 RLS=false | b3 tạo TRƯỚC GAP-01 → chưa có RLS; re-Approve/re-Apply b3 sẽ bật. b2 RLS=t ✅. |
| G5 | Architecture chặng-2 (xem 03_architecture_shadow_master) | GAP-SAFE-2 (invalidate rule cache sau DDL), GAP-PERF-1 (mini-batch write), GAP-COMP-1 (watermark/Kafka) — chưa exec, đã có giải pháp. |

## Trạng thái DATA live (đối chiếu)
- master_binding: sss1(7)/b2(9)/b3(10) approved nhưng **is_active=false** → transmute SẼ skip (master gate). Để sync data, operator phải **bật Active** trên /masters. (b2 transmute gần nhất: success scanned=0 do is_active=false — đúng gate, không phải bug.)
- mapping_rule_master: b2(9)=14 rule/1 approved/1 in_master; b3(10)=14/14/14 (đã DDL 14 cột → dest b3 25 cột).
- dest: b2 RLS=on(11 cột), b3 RLS=off(25 cột).

## Kết luận
Feature **~95% hoàn chỉnh, build sạch 3 lane, source→shadow an toàn**. Còn **2 bug thật** (B1 clone status, B2 transform_fn modal) + vài hardening nhỏ. KHÔNG có lỗi compile / không có flag đỏ kiến trúc. Để chạy sync ra data: bật **is_active** cho master (toggle Active).

> Em đề xuất pass kế đóng B1 + B2 (2 bug rõ) + G1 (xoá dead endpoint). G2/G3/G4/G5 gom polish. CHỜ anh OK (turn này audit-only, không commit/push như anh dặn).
