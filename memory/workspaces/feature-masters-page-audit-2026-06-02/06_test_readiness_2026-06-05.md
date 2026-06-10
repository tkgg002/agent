# 06_test_readiness_2026-06-05.md — Audit READINESS trước manual test (KHÔNG commit)

> **Agent**: Muscle:Claude-Opus-4.8 | 2026-06-05 | Verify cuối trước khi User test tay.

## ✅ Sẵn sàng (verified)
| Hạng mục | Kết quả |
|---|---|
| Build worker | go build = 0 |
| Build CMS | go build = 0 |
| Build FE | tsc -b = 0 |
| Services | worker :8082 UP · CMS :8083 health=200 · FE :5173 UP |
| Routes CMS | /v1/masters (Create/Approve/Reject/toggle-active/Swap/List), /v1/schedules (Create/run-now/PATCH/DELETE/List), /v1/master-mapping-rules (List/Save/Delete/batch/sync-from-shadow), /mapping-rules (shadow CRUD) — ĐỦ |
| Worker subscribers | cdc.cmd.transmute, transmute-shadow, master-create, scan-array (+18 listeners) — ĐỦ |
| source→shadow | NGUYÊN VẸN (không bị transmute đụng) |

## 📌 State DB hiện tại (điểm xuất phát test)
**Masters** (đều `is_active=FALSE`):
| id | table | transform | schema_status | active | shadow | master rules (total/approved/in_master) | dest table |
|----|-------|-----------|---------------|--------|--------|------------------------------------------|-----------|
| 10 | **b3** | copy_1_to_1 | approved | ❌ false | shadow_aaaaa.export_jobs_4 (454 rows) | **14/14/14** | dest b3 (25 cột, RLS=off*) |
| 9 | b2 | copy_1_to_1 | approved | ❌ false | export_jobs_4 (454) | 14/1/1 | dest b2 (11 cột, RLS=on) |
| 7 | sss1 | copy_1_to_1 | approved | ❌ false | export_jobs_4 (454) | 0/0/0 | (chưa tạo) |
| 8 | b1 | copy_1_to_1 | pending_review | ❌ false | export_jobs_4 | 0/0/0 | (chưa tạo) |

(*b3 RLS=off vì tạo trước GAP-01; b2 RLS=on. Re-Approve b3 sẽ bật RLS.)
**Schedules**: rỗng (Sync Modal sẽ tạo). **Shadow data**: export_jobs_4 = 454 rows (đủ để test sync).

## ⚠️ LƯU Ý QUAN TRỌNG để test ra data
**Tất cả master đang `is_active=false` → transmute SẼ SKIP** (master gate, đúng thiết kế). Muốn sync ra data PHẢI **bật Active** trên /masters trước.

## 🧪 Kịch bản test đề xuất (happy-path nhanh nhất)
**Test sync→master (dùng b3 — đã đủ 14 rule approved + table sẵn + shadow 454 rows):**
1. Vào `/masters` → bật **Active** cho `b3` (toggle Active).
2. Bấm **Sync** → chọn **Chạy ngay (run_now)** → nhập reason ≥10 → OK.
3. Kỳ vọng: message "đã bắn worker"; vào **Activity Log** thấy row `transmute / b3 / success` với rows_affected > 0.
4. Verify data: `dw_centrallized_export_service.b3` có dữ liệu nghiệp vụ (không có `_raw_data`).

**Test master mapping page (`/masters/b2/mappings?binding_id=9`):**
- Thấy 12 cột gồm: Source Field, Target Column, Source Data Type, Data Type, Rule Type, **Transform** (mới), Status of Shadow, In Shadow, Status, In Master, Active, Actions.
- Checkbox approve bị **disable** nếu field chưa "in shadow" hoặc shadow chưa approved.
- Batch **Duyệt** vài field → kỳ vọng triggerMasterDDL → cột được thêm vào dest table (verify `\d dw_centrallized_export_service.b2`).
- **Create Manual Mapping**: chọn 1 shadow rule (v2) + target column → tạo rule status **pending**.
- **Scan Array (Flatten)**: nhập explode_path → modal review field → Promote.
- **Sync from shadow**: kéo rule v2 mới về master (pending).

**Test create master MỚI**: tạo master mới → mapping_rule_master clone **status='pending'** (B1 fix — không còn auto-approved).

**Test /schedules**: menu hiện → tạo schedule 3 mode → Run now → **Xoá** (Delete, có confirm).

**Test db→shadow (mapping-rules)**: tạo shadow mapping rule → 201 + hiện trong list (shadow_binding_id resolve đúng — #3a/#10 fix).

## Còn lại (polish, KHÔNG chặn test)
G1 dead endpoint master-columns · G2 path-1 shadow-scope · G3 Save NULL created_by · G4 b3 RLS re-apply · G5 kiến trúc (cache-invalidate/mini-batch/watermark).

## Bug đã đóng phiên này: B1 (clone pending ✅), B2 (Transform column ✅). KHÔNG commit/push.
