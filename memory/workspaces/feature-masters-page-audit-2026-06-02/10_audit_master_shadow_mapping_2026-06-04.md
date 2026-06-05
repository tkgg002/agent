# 10_audit_master_shadow_mapping_2026-06-04.md — AUDIT (KHÔNG sửa code)

> **Agent**: Muscle:Claude-Opus-4.8 | **Ngày**: 2026-06-04
> **Lệnh User**: "audit lại toàn bộ chức năng, KHÔNG được làm". Đây là audit-only, mọi kết luận có evidence DB/`file:line`. Tôi nhận sai ở vài điểm (đánh dấu 🔴 MEA CULPA).
> **Lưu ý**: một số file đã được sửa song song (SyncFromShadow, flatten re-add ở master FE) — audit phản ánh state HIỆN TẠI.

---

## Tóm tắt 5 vấn đề + verdict

| # | Vấn đề User nêu | Verdict | Mức |
|---|---|---|---|
| 1 | master ít field hơn shadow (13 vs 8) | 🔴 BUG THẬT — blacklist loại nhầm cột nghiệp vụ | CAO |
| 2 | status master = approved (nên là chuyện riêng) | 🔴 BUG (của tôi) — clone hardcode 'approved' | CAO |
| 3 | thiếu trạng thái "đã DDL field vào master chưa" | ❌ THIẾU feature | TRUNG |
| 4 | flatten đặt nhầm (master cần, shadow không) | ✅ ĐÃ ĐÚNG ở code hiện tại (đã re-add master) | — |
| 5 | add mapping shadow báo ok nhưng ko hiện | 🔴 BUG THẬT — rule lưu shadow_binding_id=NULL | CAO |

---

## [1] 🔴 master thiếu field nghiệp vụ — blacklist over-match (CAO)
**Evidence (DB)**: shadow_binding 74 (`shadow_aaaaa.export_jobs_4`) có **14** field mapping_rule_v2:
`__v, _id, createdAt, error, exportType, fileUrl, jobId, lastUpdatedAt, merchantId, params, progress, status, totalRecords, userId`.
master_binding 7 (`sss1`) chỉ có **9**: thiếu `_id, error, params, progress, status`.
- `_id` → đúng là loại (Mongo system).
- **`error, params, progress, status` → SAI: đây là CỘT NGHIỆP VỤ của export_jobs** (trạng thái/tiến độ/tham số/lỗi của job export), KHÔNG phải cột hệ thống CDC.

**Root cause**: blacklist (ở `create_master.go:191-194` clone, `master_mapping_rule_handler.go isSystemColumn`, `SyncFromShadow:154-157`) trộn cột hệ thống CDC (`_`-prefixed: `_gpay_id,_raw_data,...`) VỚI các tên generic (`status, params, error, progress`). Tên generic trùng cột nghiệp vụ thật → bị loại oan khỏi master.
**Fix direction (chưa làm)**: blacklist CHỈ nên chứa cột hệ thống CDC thật (`_`-prefixed + `_id`). Bỏ `status, params, error, progress` khỏi blacklist (chúng là business). Hệ thống đã có cờ phân biệt cột hệ thống ở tầng shadow (column metadata `is_system`?) — nên dùng cờ đó thay vì so tên.

## [2] 🔴 clone master hardcode status='approved' (CAO — của tôi)
**Evidence**: `create_master.go:187` `SELECT ?, v2.id, v2.target_column, true, 'approved', ...` → mọi master rule clone ra = `approved` (master_binding 7: 9/9 rule `approved`).
**Đối lập**: `SyncFromShadow:150` (vừa thêm) dùng `'pending'` — đúng ý User.
**Root cause**: tôi hardcode 'approved' lúc redesign. Sai vì: **duyệt ở shadow (mapping_rule_v2.status) và duyệt ở master là 2 việc khác nhau** — operator phải review/duyệt rule trên master riêng. Auto-approve bỏ qua bước kiểm soát master.
**Fix direction**: clone (create_master) đổi `'approved'` → `'pending'` (đồng bộ với SyncFromShadow). Worker chỉ transmute rule master `status='approved'` → buộc operator duyệt master trước khi field vào DW.

## [3] ❌ Thiếu trạng thái "field đã DDL vào master table chưa" (TRUNG)
**Hiện trạng**: master mapping page hiển thị status duyệt (pending/approved/rejected) + active, NHƯNG không cho biết cột đó **đã được tạo vật lý** (ALTER/CREATE) trong master table ở dest chưa.
**Cơ chế hiện có**: worker `MasterDDLGenerator.Apply` tạo cột khi approve master (CREATE TABLE/ALTER ADD COLUMN). Nhưng không có cờ/đối chiếu hiển thị per-field "ddl_applied".
**Fix direction (chưa làm)**: thêm cột hiển thị bằng cách đối chiếu `mapping_rule_master.target_column` với `information_schema.columns` của physical master table (dest DB) — hoặc track `ddl_applied_at` khi worker Apply. Cảnh báo: dest DB (5434) chỉ worker chạm được → cần API worker trả trạng thái, hoặc CMS query qua connection master (hiện CMS không có connection dest). Cần thiết kế (đề xuất: worker emit ddl_applied, CMS lưu cờ).

> **UPDATE (Gemini vừa thêm 13:20)**: endpoint `GET /v1/master-mapping-rules/master-columns` + cột "In Master" FE. 🔴 **NHƯNG SAI DB**: handler `MasterColumns` (`master_mapping_rule_handler.go:243,251`) dùng `h.db` = **control plane (5433/cdc_dw)** query `information_schema.columns`, trong khi master table vật lý nằm ở **dest (5434/goopay_dest)** — CMS KHÔNG có connection dest (xem `09_solution_master_provisioning`). ⇒ `information_schema.columns` ở control plane KHÔNG thấy master table → cột "In Master" **luôn rỗng (false-negative)**. Đây đúng là rào kiến trúc tôi nêu: chỉ worker chạm dest được. Fix đúng phải qua worker (emit ddl_applied) hoặc CMS mở connection master (role dest) chỉ để introspect.

## [4] ✅ Flatten — ĐÃ ĐÚNG ở code hiện tại (master có, shadow không)
**Evidence**:
- Master FE `MasterMappingFieldsPage.tsx:62-66,172-203` CÓ "Scan Array (Flatten)" + explode_path modal (đã re-add).
- Shadow FE `MappingFieldsPage.tsx`: grep flatten = **rỗng** (không có) ✅.
- Endpoint `introspection_handler.go:385 ScanArray` (POST /introspection/scan-array/:table) publish `cdc.cmd.scan-array` cho worker; route `router.go:343`.
**Đánh giá**: đúng tư duy của User (shadow=raw không cần flatten; master cần). 🔴 MEA CULPA: turn trước tôi gỡ flatten khỏi master = SAI; đã được re-add. **Cần verify chạy thật**: scan-array → worker bóc array tại explode_path → tạo v2 rule pending → sync master. (Chưa verify E2E trong audit này — đề xuất test sau.)
**Lưu ý thiết kế**: flatten ở master gọi scan-array tạo rule trong `mapping_rule_v2` (pending) rồi sync sang master. Hợp lý (rule trích xuất sống ở v2; master pick qua mapping_v2_id). User xác nhận hướng này OK chứ?

## [5] 🔴 Add mapping shadow "ok" nhưng không hiện (CAO)
**Evidence (DB)**: rule user vừa add = `mapping_rule_v2 id=223`: `source_object_id=66, shadow_binding_id=NULL, source_field='params', target_column='aaaaa', status=pending`. → INSERT thành công (201) nhưng **shadow_binding_id=NULL**.
**FE shadow list** (`MappingFieldsPage.tsx:185`): `params.shadow_binding_id = scopedBindingID` → list LỌC theo `shadow_binding_id`. Rule 223 (NULL) → **không match → invisible**.
**Root cause**:
1. `create_mapping_rule.go resolveScope` không gán được `shadow_binding_id` cho rule 223 (lưu NULL).
2. **source_object 66 có 5 shadow_binding active** (`export_jobs, _2, _3, _4, _5` — id 66/69/72/74/79) → scope resolution mơ hồ; khi tạo rule không xác định được binding nào → NULL.
3. List filter strict theo shadow_binding_id → rule NULL biến mất.
**Fix direction (chưa làm)**: create_mapping_rule PHẢI lưu `shadow_binding_id` khớp scope FE đang xem (FE gửi binding_id → BE dùng trực tiếp, không re-resolve mơ hồ); HOẶC list fallback theo source_object_id khi shadow_binding_id NULL. Gốc sâu: source_object 66 có 5 binding (sprawl export_jobs) — cần rà tại sao 1 source có 5 shadow_binding.

---

## Quan sát thêm (broad functional audit)
- **master_binding 7 `is_active=FALSE`** dù `schema_status=approved`: master approved nhưng chưa bật active → worker transmute sẽ skip (is_active gate). Khác với per-field status. User phân biệt đúng: "approve bên kia kệ nó" = duyệt shadow ≠ duyệt master ≠ active master.
- **Parallel edits đã có trong source** (chưa rõ đã build/restart): `SyncFromShadow` (master_mapping_rule_handler.go — pull v2 mới → master pending), flatten re-add (MasterMappingFieldsPage). → 1 phần item 2/4 đang được xử lý. **Cần xác nhận route `sync-from-shadow` đã đăng ký trong router chưa** (nếu chưa → handler dead-code).
- **Blacklist trùng lặp 3 nơi** (create_master, isSystemColumn, SyncFromShadow) → sửa phải đồng bộ cả 3 (DRY risk).

## Ưu tiên fix (đề xuất — CHỜ User chốt, chưa làm)
| P | Việc |
|---|---|
| P0 | [1] Sửa blacklist: chỉ loại cột `_`-prefixed CDC, bỏ `status/params/error/progress` (đồng bộ 3 nơi) → master đủ field nghiệp vụ |
| P0 | [5] create_mapping_rule lưu đúng shadow_binding_id (dùng binding_id FE gửi); + điều tra source_object 66 sao có 5 binding |
| P1 | [2] clone đổi status 'approved'→'pending' (đồng bộ SyncFromShadow) |
| P1 | [4] verify flatten master chạy E2E (scan-array → worker → v2 pending → sync master) |
| P2 | [3] thêm trạng thái "ddl_applied" per field ở master (worker emit / CMS đối chiếu information_schema) |
| P2 | xác nhận route sync-from-shadow đã đăng ký; DRY blacklist |

## Files đã đọc/đối chiếu (không sửa)
- DB: mapping_rule_v2 (shadow 74, rule 223), mapping_rule_master (master 7), shadow_binding (source_object 66 → 5 binding), master_binding 7.
- Code: create_master.go (clone), create_mapping_rule.go (resolveScope), master_mapping_rule_handler.go (List/Save/SyncFromShadow), mapping_rule_handler_list.go (shadow list filter), introspection_handler.go (ScanArray), router.go, MasterMappingFieldsPage.tsx, MappingFieldsPage.tsx.
- **Không thay đổi dòng code nào.**
