# 07_status_issues_2026-06-04.md — Trạng thái 10 issue (verify code HIỆN TẠI)

> **Agent**: Muscle:Claude-Opus-4.8 | **Ngày**: 2026-06-04
> ⚠️ **CẢNH BÁO GỐC RỄ**: Brain(Gemini) ĐANG sửa code song song với Muscle(tôi) trên CÙNG các file (`master_mapping_rule_handler.go`, `create_mapping_rule.go`, `master_ddl_generator.go`, `master_rule.go`, `server.go`...). Vi phạm §1/§12 (Brain KHÔNG được chạm code) + gây xung đột/bug chồng. **Đây là nguồn hỗn loạn**. Đề xuất: CHỈ 1 agent lái code (Muscle execute), Brain chỉ plan. Tôi KHÔNG sửa turn này để tránh đạp lên edit đang bay của Gemini.

## Trạng thái từng issue (verify code hiện tại — không sửa)

| # | Issue | Trạng thái hiện tại | Còn phải làm |
|---|---|---|---|
| 1 | log `column master_name does not exist` | ✅ FIXED (đã đổi `master_name`→`master_table`; grep 0 ref) | — |
| 2 | approve → master table có đủ 11 system col (`_gpay_id..._raw_data..._updated_at`) | ⚠️ CÒN — `master_ddl_generator.go:93-105` hardcode 11 system col vào CREATE TABLE | **CẦN QUYẾT ĐỊNH**: cột nào giữ. Upsert cần `_gpay_id(PK), _source_id, _hash, _synced_at, _deleted, _version, _source_ts`. `_raw_data`(JSON thô) KHÔNG nên ở master (master=business). Đề xuất: bỏ `_raw_data` (+ `_source` nếu không cần), GIỮ các cột phục vụ upsert/dedup. Xác nhận giúp em. |
| 3 | thêm cột `shadow_status` + `in_shadow` TRƯỚC `status`+`in_master`; chưa đủ thì ko cho approve master | 🟡 PARTIAL — domain đã có `ShadowStatus`,`InMaster` (Gemini); THIẾU `in_shadow` (cột field đã tạo vật lý ở shadow table chưa) + gate "chỉ approve master khi shadow approved+in_shadow" + 2 cột FE | gate logic + in_shadow (đối chiếu information_schema shadow table) + FE 2 cột + disable checkbox |
| 4 | Source Data Type chưa lấy từ mapping_v2 | 🟡 domain có `SourceDataType`; cần verify repo JOIN select `v2.source_data_type` + FE render | verify repo + FE cột |
| 5 | Data Type cho phép edit | ❌ hiện read-only (lấy từ v2). User muốn edit | **CẦN QUYẾT ĐỊNH**: edit data_type ở master = ghi đè đâu? (a) sửa thẳng `mapping_rule_v2.data_type` (ảnh hưởng shadow), hay (b) master có cột `data_type` override riêng (thêm lại cột vào mapping_rule_master). Đề xuất (b) — master override, không đụng v2. |
| 6 | `master-columns?master_binding_id=9` → "master binding not found" | ⚠️ binding 9 (`b2`) CÓ tồn tại; `master_name` đã fix nên hết "not found", NHƯNG handler query `information_schema` trên **control plane (h.db=5433)** — master table vật lý ở **dest (5434)**, CMS KHÔNG có conn dest → "In Master" sẽ **LUÔN RỖNG** (false-negative) | lấy cột master từ **worker** (chỉ worker chạm dest), không query CMS-control-plane |
| 7 | approve field master → table master chưa tạo field đó | ⚠️ `triggerMasterDDL` (handler) publish `cdc.cmd.master-create` khi batch approve; worker `MasterDDLGenerator.Apply` ALTER ADD COLUMN. CƠ CHẾ CÓ | verify E2E: vì sao field chưa được ALTER (rule status='approved' lúc DDL chạy? Apply có nhận? master is_active?). Cần test live |
| 8 | trang `/masters/:name/mappings` thiếu nút tạo mapping thủ công | ❌ THIẾU (turn trước tôi gỡ Add khi redesign) | thêm nút "Tạo mapping thủ công" (chọn mapping_v2_id + target_column, hoặc tạo v2 rule mới rồi link) |
| 9 | scan-array `relation shadow_aaaaa.export_jobs_4 does not exist` | ⚠️ Table EXISTS ở shadow 5436 (453 rows); worker `HandleScanArrayFields` dùng `h.shadowDB` + `resolveTargetSchema(targetTable)` — lỗi do **resolveTargetSchema/h.shadowDB route sai** (không ra schema `shadow_aaaaa` đúng, hoặc h.shadowDB trỏ control). + UX mong muốn: scan OK → **modal preview** field quét được (CHƯA insert) → user confirm → insert `mapping_rule_master` status='pending' | fix DB routing worker + UX modal-confirm |
| 10 | POST /api/mapping-rules thiếu `shadow_binding_id` → ko show | 🟡 Gemini đã thêm field `shadow_binding_id` vào create_mapping_rule | verify INSERT lưu ĐÚNG binding (payload có shadow_schema/shadow_table=export_jobs_4 nhưng response trả shadow_table=export_jobs → resolveScope chọn SAI binding; phải khớp binding theo shadow_schema+shadow_table, không lấy binding đầu của source_object) |

## Nhóm CẦN QUYẾT ĐỊNH trước khi code (tránh làm sai lại)
- **[2]** Master table giữ system col nào? (đề xuất: bỏ `_raw_data`, giữ phần phục vụ upsert).
- **[5]** Data type edit ghi vào đâu? (đề xuất: master override column riêng).

## Nhóm RÕ RÀNG, em execute được ngay (nếu được lái 1 mình)
- [6] In Master lấy từ worker (đúng DB dest).
- [9] fix worker scan-array routing + modal-confirm UX.
- [10] create_mapping_rule resolve đúng shadow_binding theo schema+table.
- [3] in_shadow + gate approve.
- [4] verify source_data_type JOIN + FE.
- [7] verify/fix approve→DDL E2E.
- [8] nút tạo mapping thủ công.

## Đề xuất
1. **Chốt 1 driver**: dừng Gemini sửa code; để Muscle(em) execute trọn 10 issue trong 1 mạch sạch (build+verify từng bước). 2. Chốt 2 quyết định [2],[5]. → em làm 1 lượt, report đầy đủ.
