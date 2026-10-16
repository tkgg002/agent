# 09 Technical Solution: Fix Unique Constraint ux_<master_table>__id on Transmute Full

## 1. Vấn đề thực tế (Problem)
Khi thực hiện **Chạy ngay (toàn bộ)** (Transmute Full) cho master binding (ví dụ `export_jobs_1`):
PostgreSQL trả về lỗi:
```
Failed: ERROR: duplicate key value violates unique constraint "ux_export_jobs_1__id" (SQLSTATE 23505)
```

## 2. Phân tích nguyên nhân gốc rễ (Root Cause)
1. **Kiến trúc Schema bảng Master (`export_jobs_1`)**:
   - `PRIMARY KEY (_gpay_id)`: Khóa chính synthetic toàn cục do CDC sinh ra (Sonyflake int64).
   - `UNIQUE INDEX "ux_export_jobs_1__id" ON (_id)`: Ràng buộc duy nhất nghiệp vụ được sinh tự động bởi `master_ddl_generator.go:286-303` dựa trên cấu hình `transform_spec: {"pk": "_id"}`.
2. **Xung đột khi Upsert**:
   - Tại `bulkUpsertMaster` (`centralized-data-service/internal/service/master/transmuter.go:1154`), biến `conflictTarget` bị **hardcode** là `"_gpay_id"`.
   - Câu lệnh SQL sinh ra:
     ```sql
     INSERT INTO master_centrallized_export_service.export_jobs_1 (_gpay_id, _id, ...)
     VALUES ($1, $2, ...)
     ON CONFLICT ("_gpay_id") DO UPDATE SET ...
     ```
   - Khi chạy Transmute Full:
     - Trên Master table đã tồn tại 515 dòng với `_id` cũ và mang `_gpay_id` cũ.
     - Shadow table có 516 dòng với `_gpay_id` mới.
     - Do câu lệnh chỉ bắt `ON CONFLICT ("_gpay_id")`, Postgres không thấy trùng `_gpay_id` nên chuyển sang **INSERT dòng mới**.
     - Khi INSERT, Postgres kiểm tra UNIQUE INDEX `ux_export_jobs_1__id` và phát hiện `_id` đã tồn tại trong Master → ném ngay lỗi `SQLSTATE 23505`.
3. **Cơ chế Deduplication nội bộ mẻ**:
   - Trong `processBatch` (`transmuter.go:926`), dedup chỉ gom nhóm theo `_gpay_id`. Nếu có 2 bản ghi cùng `_id` nhưng khác `_gpay_id`, Postgres sẽ quăng lỗi `SQLSTATE 21000 (ON CONFLICT DO UPDATE command cannot affect row a second time)`.

## 3. Giải pháp kỹ thuật chuẩn kiến trúc (Simplicity First, Minimal Impact)
1. **Xác định động `conflictTarget`**:
   - Dùng hàm `parsePKFromSpec(binding.TransformSpec)` để lấy business PK (ví dụ `_id`).
   - Nếu tìm thấy và tồn tại trong `colTypes`, gán `conflictTarget = pkCol`.
   - Fallback về `conflictTarget = "_gpay_id"`.
2. **Cập nhật danh sách `sets` trong `DO UPDATE SET`**:
   - Khi `conflictTarget != "_gpay_id"`, ta loại trừ cả `conflictTarget` và `_gpay_id` khỏi `sets`.
   - Giữ nguyên `_gpay_id` cũ của dòng Master (không ghi đè PK), cập nhật toàn bộ các trường nghiệp vụ mới (`testsubsettype`, `_source_ts`, `_updated_at`, v.v.).
3. **Đồng bộ cơ chế Dedup trong `processBatch`**:
   - Khóa dedup nội bộ mẻ được lấy theo `rec[conflictTarget]` (hoặc fallback `rec["_gpay_id"]`), giữ lại bản ghi có `_source_ts` mới nhất.
