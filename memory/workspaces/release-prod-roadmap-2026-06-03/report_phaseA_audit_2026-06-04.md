# report_phaseA_audit_2026-06-04.md — Audit Blueprint vs Source THỰC TẾ

> **Ngày**: 2026-06-04 | **Agent**: Brain:Claude-Opus-4.8 (điều phối 3 subagent audit)
> **Loại**: AUDIT (read-only). **KHÔNG sửa source code.** Brain tuân thủ §12.
> **Phạm vi**: Luồng Master/Transmute (shadow→master). KHÔNG đụng luồng source→shadow.

## 1. Cách audit (để không báo láo — lesson L99)
- 3 subagent đọc trực tiếp repo `data-hub/{cdc-cms-web, cdc-cms-service, centralized-data-service}`.
- Mỗi claim blueprint phải kèm `file:line` thật. Không suy diễn.
- Kết quả lưu thành file vật lý: `10_gap_analysis_phaseA.md` (chi tiết) + report này.

## 2. Kết quả tổng (coverage)
| Lane | Coverage blueprint | Trạng thái |
|------|--------------------|-----------|
| FE (cdc-cms-web) | ~85% | 3 trang LIVE; gap = CRUD schedule + transform_fn + menu + filter |
| API (cdc-cms-service) | ~80% | G1-G3 đủ lõi; gap = update/delete + endpoint hợp nhất (scan-flatten, sync/status) |
| Worker (centralized-data-service) | ~90% wired | E1-E5 chạy thật; rủi ro chất lượng = RLS + OCC + close-loop |
| **Tổng** | **~85%** | **Không build từ đầu — chỉ đóng gap + siết chất lượng** |

## 3. Phát hiện đáng chú ý (đã verify file:line)
- ✅ **Đã có thật nhiều hơn blueprint mô tả**: Atomic Swap, Toggle Active gate, inline data_type edit, shadow-approval checkbox gate (FE); BatchUpdate + SyncFromShadow + MasterColumns + SyncHealth (API); fencing + FOR UPDATE SKIP LOCKED + hasPostIngestSchedule gate + JobMonitor idempotent (Worker).
- 🔴 **GAP-01 RLS (HIGH/prod-blocker)**: `master_ddl_generator.go:216-219` chỉ enable RLS khi `MasterSchema=="public"`; `038_*.sql:171-196` tạo policy `USING(true)` = permissive full-access. Schema khác bỏ RLS.
- 🟠 **GAP-02 OCC**: master upsert `transmuter.go:546-557` dùng `_hash IS DISTINCT FROM` thay vì `_source_ts` → cần verify rủi ro out-of-order. (Shadow-side `upsert.go` mới dùng `_source_ts` — KHÔNG đụng.)
- 🟠 **GAP-03 close-loop**: `job_monitor.go:75-79` bỏ qua trigger không có `schedule_id` → realtime/run-now không cập nhật `last_status`.
- 🟠 **GAP-04 menu**: `App.tsx:118-122` comment out `/schedules` — operator không vào được.
- 🟠 **GAP-05/06**: thiếu Edit/Delete schedule (FE+API) + thiếu `transform_fn` trong Create Mapping.

## 4. Naming chốt (tránh bug lặp lại)
- Cột resolve binding là **`master_table`** (KHÔNG phải `master_name`) — xác nhận `approve_master.go:69`, `create_schedule.go:59`, `master_registry_handler_resolve.go:19`.

## 5. Files THAY ĐỔI trong turn này
| File | Loại | LOC (mới) |
|------|------|-----------|
| `08_tasks_phaseA_capability_map.md` | doc (turn trước) | — |
| `10_gap_analysis_phaseA.md` | doc audit mới | ~95 dòng |
| `report_phaseA_audit_2026-06-04.md` | report này | ~70 dòng |
| `05_progress.md` | append log | +1 dòng |
> **0 file source code bị sửa** (đây là audit read-only).

## 6. Verify trước khi báo "done audit"
- Mọi verdict đều có `file:line` từ subagent đọc source thật → không phải phỏng đoán.
- Chưa chạy build/test vì KHÔNG sửa code (audit). Khi Muscle execute gap → bắt buộc build+test+security gate rồi mới done.

## 7. Khuyến nghị (best path)
1. Đóng **GAP-01 RLS** đầu tiên (HIGH, cần ADR + migration).
2. **Verify rồi sửa GAP-02 OCC** (test reproduce trước, không sửa mù).
3. Cụm vận hành **GAP-03/04/05/06** để operator dùng trọn vòng trên UI.
4. **GAP-07..12** gom polish hoặc defer kèm ticket — không chặn big-bang release.

Chi tiết severity + evidence: xem `10_gap_analysis_phaseA.md`.
