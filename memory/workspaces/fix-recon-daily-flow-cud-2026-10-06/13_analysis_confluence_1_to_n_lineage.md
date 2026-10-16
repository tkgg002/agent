# Phân Tích Gốc Rễ: Quan Hệ 1 Source -> N Shadows -> M Masters & Lỗi "Bùa Số Liệu / LIMIT 1" Trong Confluence Report

- **Thời gian:** 2026-10-15T10:16:00+07:00
- **Trạng thái:** Read-Only Analysis & Proposal (Tuân thủ Rule #1 & Rule #13 — Không tự ý sửa source code)

---

## 1. Nhận Lỗi Trực Diện Trước Phản Biện Của User

Phản biện của User:
> *"cho mày kiểm tra lại đó, source export_jobs này, có nhiều shadow, 1 shadow có nhiều master. tao k biết mày đang làm gì luôn, mày biết cái nào mà lấy ra để kiểm tra. làm số liêu bùa à, hay ilimit 1 cái đầu tiên"*

**Xác nhận 100% lỗi tư duy và thực thi:**
Trong `confluence_runner.go` (dòng 182–203) và `ActivityManager.tsx` (dòng 383–394, 687–702), code đã áp đặt sai lầm mô hình phẳng `1 Bảng = 1 Dòng` và dùng câu lệnh SQL chắp vá `WHERE object_name IN (?) ORDER BY CASE WHEN layer_type = 'source' THEN 0 ... LIMIT 1` để lấy bừa 1 dòng đầu tiên. Đây đúng nghĩa là **"bùa số liệu bằng `LIMIT 1`"**, hoàn toàn mù tịt trước cấu trúc fan-out `1 Source -> N Shadows -> M Masters` thực tế của hệ thống.

---

## 2. Sự Thật Cấu Trúc Dữ Liệu Thực Tế Trong `cdc_dw` (`cdc_system`)

Qua truy vấn trực tiếp 4 bảng metadata (`connection_registry`, `source_object_registry`, `shadow_binding`, `master_binding`) trên `cdc_dw`:

### 2.1. Trường hợp 1: 1 Source -> N Shadows (`payment-bills`, `source_object_id = 192`)
- **Source (`id = 192`):** `[traitestpbs]` `payment-bill-service.payment-bills` (`mongodb`)
  - **Shadow 1 (`shadow_binding.id = 192`, `active = true`):** `[default_shadow]` `shadow_traitestpbs.payment_bills`
    - **Master (`master_binding.id = 46`, `active = true`):** `[default_master]` `master_payment_bill_service.payment_bills`
  - **Shadow 2 (`shadow_binding.id = 196`, `active = true`):** `[default_shadow]` `shadow_traitestpbs.payment_bills_1`
    - **Master (`master_binding.id = 47`, `active = true`):** `[default_master]` `master_payment_bill_service.payment_bills_1`

### 2.2. Trường hợp 2: 1 Source -> 1 Shadow -> M Masters (`export-jobs`, `source_object_id = 201`)
- **Source (`id = 201`):** `[traitestces]` `centrallized-export-service.export-jobs` (`mongodb`)
  - **Shadow (`shadow_binding.id = 201`, `active = true`):** `[default_shadow]` `shadow_traitestces.export_jobs`
    - **Master 1 (`master_binding.id = 48`, `active = true`, `approved`):** `[default_master]` `master_centrallized_export_service.export_jobs`
    - **Master 2 (`master_binding.id = 52`, `active = true`, `approved`):** `[default_master]` `master_centrallized_export_service.export_jobs_1`
    - **Master 3 (`master_binding.id = 51`, `active = true`, `approved`):** `[master_2]` `master_centrallized_export_service.export_jobs_2`
    - *(Master 4 `master_binding.id = 50`: `[master_2]` `master_centrallized_export_service.export_jobs` — `is_active = false, schema_status = pending_review`)*

---

## 3. Ba Điểm "Bùa Số Liệu" Trong Code Hiện Tại (`confluence_runner.go` & `ActivityManager.tsx`)

1. **Dùng `LIMIT 1` trộn lẫn cả 3 `layer_type` (`confluence_runner.go` dòng 182–187):**
   ```sql
   SELECT creation_count, layer_type
   FROM cdc_system.daily_source_counts
   WHERE object_name IN ('export_jobs', 'export-jobs') AND report_date = '2026-10-14'
   ORDER BY CASE WHEN layer_type = 'source' THEN 0 WHEN layer_type = 'shadow' THEN 1 ELSE 2 END, updated_at DESC
   LIMIT 1
   ```
   - Không lọc theo `connection_code` hay `container_namespace`.
   - Nếu tầng `source` chưa đếm mà chỉ mới có bản ghi đếm của `shadow` hoặc `master`, câu SQL này tự động lấy số đếm của `shadow`/`master` gắn mác vào cột `"Số Bản Ghi Ngày Tạo (Source)"`!
   - Hoàn toàn bỏ sót các bảng Master có tên khác tên gốc như `export_jobs_1` và `export_jobs_2` (dòng `id = 11` trong `daily_source_counts` có `object_name = 'export_jobs_2'` trên `master_2`).
2. **Truy vấn bảng không tồn tại `cdc_system.recon_hourly_pipeline_stats` (`confluence_runner.go` dòng 196–203):**
   - Bảng `recon_hourly_pipeline_stats` không tồn tại trong DB (`SQLSTATE 42P01`). Code chỉ `logger.Warn` rồi để `KafkaIngest = 0, ShadowIngest = 0, MasterUpsert = 0`.
   - Sau đó so sánh `sourceCount (0) == MasterUpsert (0)` và báo `"🟢 MATCH (Khớp 100%)"` hoàn toàn giả tạo.
3. **Frontend `ActivityManager.tsx` (dòng 383–394 & 687–702) chỉ lưu tên bảng trần trụi:**
   - Dropdown chỉ lấy chuỗi `target_table` (`export_jobs`) và `.find()` phần tử đầu tiên, không lưu `source_object_id` và không phản ánh cây `Source -> Shadow -> Master`.

---

## 4. Phương Án Thiết Kế Đúng Chuẩn Theo Cây Huyết Thống (Lineage Tree 1 -> N -> M)

Khi chọn `export_jobs` (`source_object_id = 201`), hệ thống phải:
1. **Truy vết đầy đủ các nhánh `Source -> Shadow -> Master` đang active từ `source_object_registry` JOIN `shadow_binding` JOIN `master_binding` JOIN `connection_registry`.**
2. **Tra cứu `cdc_system.daily_source_counts` theo đúng bộ khóa 5 thành phần `(report_date, layer_type, connection_code, container_namespace, object_name)` cho TỪNG trạm:**
   - Trạm **Source**: `('2026-10-14', 'source', src_conn_code, source_database, source_object_name)`
   - Trạm **Shadow**: `('2026-10-14', 'shadow', shd_conn_code, shadow_schema, shadow_table)`
   - Trạm **Master**: `('2026-10-14', 'master', mst_conn_code, master_schema, master_table)`
3. **Hiển thị từng nhánh `Source -> Shadow -> Master` thành các dòng rõ ràng trên bảng Confluence:**
   Với dữ liệu thật ngày `2026-10-14` đang có trong `cdc_system.daily_source_counts` (ID 11, 12, 13), kết quả đúng của `export-jobs` phải là **3 dòng tương ứng với 3 nhánh Master active**:

| Source (`[Conn] DB.Object`) | Số Đếm Source | Shadow (`[Conn] Schema.Table`) | Số Đếm Shadow | Master (`[Conn] Schema.Table`) | Số Đếm Master | Trạng Thái Đối Chiếu |
| :--- | ---: | :--- | ---: | :--- | ---: | :--- |
| `[traitestces]` `centrallized-export-service.export-jobs` | `0` | `[default_shadow]` `shadow_traitestces.export_jobs` | `0` | `[default_master]` `master_centrallized_export_service.export_jobs` | `Chưa đếm` | ⚪ PENDING (Thiếu số đếm Master) |
| `[traitestces]` `centrallized-export-service.export-jobs` | `0` | `[default_shadow]` `shadow_traitestces.export_jobs` | `0` | `[default_master]` `master_centrallized_export_service.export_jobs_1` | `Chưa đếm` | ⚪ PENDING (Thiếu số đếm Master) |
| `[traitestces]` `centrallized-export-service.export-jobs` | `0` | `[default_shadow]` `shadow_traitestces.export_jobs` | `0` | `[master_2]` `master_centrallized_export_service.export_jobs_2` | `0` | 🟢 MATCH (Source = Shadow = Master = 0) |
