# report_phaseB_fixes_2026-06-04.md — Muscle execute (gaps + issues)

> **Agent**: Muscle:Claude-Opus-4.8 (sole driver) | **Ngày**: 2026-06-04
> Quy tắc: shadow→master KHÔNG đụng source→shadow; build-verify từng bước; báo cáo thật.

## ✅ ĐÃ EXECUTE + BUILD turn này
| Mã | Việc | File | LoC | Verify |
|----|------|------|-----|--------|
| I2 | Master table KHÔNG còn `_raw_data` (master=business; transmute upsert vốn không ghi _raw_data → cột rỗng vô nghĩa). Giữ cột cơ chế (_gpay_id/_source_id/_hash/_synced_at/_deleted/_version/_source_ts) vì upsert/dedup cần. | `centralized-data-service/.../master_ddl_generator.go` | -1/+6 | worker go build=0 |
| GAP-04 | Mở lại menu `/schedules` (đang bị comment → operator vào được) | `cdc-cms-web/src/App.tsx` | +5/-5 | FE tsc=0 |
| GAP-03 | Close-loop run-now: `TransmuteRunCommand.ScheduleID` + RunNow set `last_status='running'` trước dispatch → JobMonitor cập nhật last_status cho run-now (trước chỉ cron) | `transmute_run.go`, `transmute_schedule_handler.go` | +12/-1 | CMS go build=0 |

> ⚠️ Cần **restart worker + CMS** để 3 fix này LIVE (binary đang chạy là bản cũ).

## ✅ ĐÃ XONG bởi Brain/Gemini (verify code hiện tại — không cần làm lại)
- **I1** log `master_name does not exist` → đã đổi `master_table` (grep 0 ref).
- **I10** create mapping thiếu shadow_binding_id → `create_mapping_rule.go` nay resolve binding theo `shadow_schema`+`shadow_table` (path-2 lines 266-273) + lưu `shadow_binding_id`. (Cần test live xác nhận.)
- **I1-blacklist** (master thiếu field) → `isSystemColumn` chỉ còn block cột `_`-infra, bỏ status/params/error/progress.

## 🔧 CÒN LẠI — giải pháp cụ thể (executable, theo priority)

### 🔴 GAP-01 RLS (prod-blocker) — cần ADR + migration (master-side, an toàn)
- Hiện: `master_ddl_generator` chỉ ENABLE RLS khi `MasterSchema=="public"`; policy `038_*.sql:171-196` `USING(true)` (full-access, không cô lập).
- **Giải pháp**: (a) ENABLE + FORCE ROW LEVEL SECURITY cho MỌI master schema (bỏ điều kiện ==public); (b) policy mặc định **deny-by-default** + 1 policy cho service role (vd `current_user = 'gpay_admin'` hoặc qua `app.role` GUC) thay `USING(true)`. Vì chưa có cột tenant → policy tối thiểu: chỉ owner/service role đọc-ghi, KHÔNG để USING(true) trống.
- File: migration mới `cdc_dw/0xx_master_rls_all_schemas.sql` + `master_ddl_generator.go` (emit `ALTER TABLE ... ENABLE/FORCE ROW LEVEL SECURITY` + CREATE POLICY cho mọi schema). **Worker chạm dest → đặt ở worker DDL.**

### 🟠 GAP-02 OCC (out-of-order) — TEST TRƯỚC, không sửa mù
- Master upsert `WHERE _hash IS DISTINCT FROM EXCLUDED._hash` → event cũ (source_ts nhỏ hơn) có thể đè bản mới nếu hash khác.
- **Giải pháp**: viết test reproduce (2 event cùng _source_id, source_ts giảm dần) → assert master giữ bản source_ts lớn nhất. Sau khi reproduce: thêm guard `AND EXCLUDED._source_ts >= <table>._source_ts` vào ON CONFLICT. **TUYỆT ĐỐI không đụng `upsert.go` shadow-side (source→shadow).**

### 🟠 I7 / approve→DDL chưa tạo field master — verify E2E
- `triggerMasterDDL` (batch approve) publish `cdc.cmd.master-create`; worker `MasterDDLGenerator.Apply` ALTER ADD COLUMN. Cần test: approve rule → field xuất hiện ở dest table. Nghi: rule status chưa 'approved' lúc DDL chạy, hoặc master is_active=false gate. → verify live, fix điểm gãy.

### 🟠 GAP-05 schedule Edit/Delete — API + FE
- BE: `PUT /api/v1/schedules/:id` (update cron_expr/mode/is_enabled) + `DELETE /api/v1/schedules/:id`. FE: nút Edit/Delete trong TransmuteSchedules.

### 🟡 GAP-06 / Create Mapping thiếu transform_fn (FE form) + I8 nút tạo mapping thủ công ở master page
- FE MasterMappingFieldsPage: thêm nút "Tạo mapping thủ công" → modal chọn mapping_v2 (hoặc tạo v2 rule mới) + target_column + transform_fn.

### 🟡 I3 in_shadow + gate approve master
- Thêm cờ `in_shadow` (field đã có cột vật lý ở shadow table chưa — worker introspect shadow) + cột FE `shadow_status`/`in_shadow` TRƯỚC `status`/`in_master`; chỉ cho approve master khi shadow approved + in_shadow.

### 🟡 I4 source_data_type — verify repo JOIN `v2.source_data_type` + FE render (domain đã có field).

### 🟡 I5 data_type edit ở master — thêm cột override `data_type` vào `mapping_rule_master` (migration) + repo/handler/FE; worker COALESCE(master.data_type, v2.data_type).

### 🟡 I6 In Master đúng DB — lấy cột master từ **worker** (chỉ worker chạm dest 5434); CMS master-columns query control-plane hiện sai → cột luôn rỗng.

### 🟡 I9 scan-array — worker route sai DB/schema (table có ở shadow 5436); fix `resolveTargetSchema`/`h.shadowDB`; + UX: scan → modal preview field quét được → user confirm → insert pending.

## Coverage thật turn này
- Execute+build: **3** (I2, GAP-04, GAP-03). Đã-xong-bởi-Gemini: 2-3 (I1, I10, blacklist).
- Còn lại: **~10** (GAP-01/02/05/06 + I3/I4/I5/I6/I7/I9) — giải pháp concrete ở trên, em execute tiếp từng cụm có verify.
- KHÔNG báo láo "done hết": đây là multi-pass; turn này chốt cụm an toàn + doc concrete phần còn lại.
