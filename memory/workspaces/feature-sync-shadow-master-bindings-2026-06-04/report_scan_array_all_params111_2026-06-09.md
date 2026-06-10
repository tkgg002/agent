# report_scan_array_all_params111_2026-06-09.md — Diagnosis (Muscle, read-only)

> **Triệu chứng**: `POST /api/introspection/scan-array-all {"master_binding_id":15}` quét `params` nhưng **bỏ sót `params111`**.
> **Kết luận**: KHÔNG phải bug code. `params111` là **cột orphan** (không có nguồn trong `_raw_data`, không có `mapping_rule_v2`). scan-array-all hoạt động ĐÚNG khi bỏ qua nó.

## Cách xác minh (data thật, không cheat)
Query trực tiếp PG cdc-metadata (5433, `cdc_dw`) + shadow DB (5436, `cdc_shadow`).

| Kiểm | Kết quả |
|------|---------|
| binding 15 → shadow_binding 82 → shadow table | `shadow_dev000.export_jobs` (ở 5436) |
| `mapping_rule_v2` của sb 82, field JSON approved+active | chỉ **`params`** (JSONB). `createdAt`,`lastUpdatedAt` JSONB nhưng `pending` → loại đúng |
| `mapping_rule_v2` có `params111`? | **0 dòng** (không tồn tại) |
| top-level key trong `_raw_data` | 14 key: __v,_id,createdAt,error,exportType,fileUrl,jobId,lastUpdatedAt,merchantId,**params**,progress,status,totalRecords,userId — **KHÔNG có `params111`** |
| `_raw_data ? 'params111'` (169 row) | **0/169** (raw không hề có key params111) |
| cột vật lý `params111` ở shadow table | CÓ, kiểu `jsonb`, **168/169 có data** (nội dung giống tập con của `params`) |
| `mapping_rule_master` (binding 15) | có `params111` (status pending) + các `params_*` flatten |

## Root cause (cơ chế)
1. **CMS `ScanArrayAll`** (`introspection_handler.go:465-474`) lấy danh sách field cần quét **CHỈ từ `mapping_rule_v2`** (approved + is_active + `data_type LIKE 'JSON%'`). `params111` không có v2 rule → **không bao giờ được đưa vào vòng quét**.
2. **Worker `HandleScanArrayFields`** (`command_handler.go:~2070`) đọc dữ liệu từ **`_raw_data #> path`**, KHÔNG đọc cột vật lý. `_raw_data` không có key `params111` → kể cả nếu CMS có gửi quét thì worker cũng trả rỗng.

→ `params111` là **cột phái sinh/mồ côi**: có cột vật lý + master rule + data, nhưng **không có lineage từ `_raw_data`** và **không có shadow v2 rule**. Nguồn Mongo phát ra `params`, không phát `params111`. Vì vậy scan-array (vốn dựa trên `_raw_data`) đúng khi không thấy nó.

## Vì sao `params` được quét còn `params111` thì không
`params` = key thật trong `_raw_data` + có v2 rule approved+active → hợp lệ để flatten. `params111` thiếu cả hai điều kiện → ngoài phạm vi thiết kế của scan-array.

## Khuyến nghị (best path)
- **Nếu `params111` là rác/thử nghiệm** (khả năng cao — tên "111", data trùng params): cleanup — drop cột `shadow_dev000.export_jobs.params111` + xoá `mapping_rule_master` params111 (binding 15). Không đụng code.
- **Nếu thật sự muốn có field params111**: nguồn (Mongo `export_jobs`) phải emit key `params111` vào document → CDC đưa vào `_raw_data` → discovery sinh v2 rule → khi đó scan-array-all tự thấy. Không nên patch scan để đọc cột vật lý (sẽ tạo mapping từ cột không có lineage → bẩn dữ liệu).
- **Cải tiến phụ (tuỳ chọn, không liên quan params111)**: "Scan Array **All**" hiện chỉ quét v2 JSON `approved`. Nếu muốn "all" gồm cả field `pending` (createdAt/lastUpdatedAt) thì nới điều kiện `status` trong query — nhưng đây là quyết định UX riêng.

## Files
- 0 file source thay đổi (đây là diagnosis read-only).
