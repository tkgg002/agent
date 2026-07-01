# Kế hoạch thực thi: Đối soát và Heal bảng export-jobs bị lệch và noop

## Phase 1: Nghiên cứu & Chẩn đoán (Research & Diagnose)
1. **Kiểm tra Cấu hình Registry**: Kiểm tra cấu hình trong bảng `cdc_system.cdc_table_registry` đối với `export-jobs` (ví dụ: `timestamp_field`, `source_db`, `source_table`, `source_type`, `target_table`).
2. **Khảo sát DB MongoDB & shadow PostgreSQL**:
   - Kiểm tra xem database MongoDB tên chính xác là gì (vì count doc ra 0).
   - Đếm số lượng record thực tế trên MongoDB collection `export-jobs`.
   - Đếm số lượng record thực tế trên PostgreSQL shadow `shadow_testexp.export_jobs`.
   - Tìm ra 1 bản ghi bị lệch.
3. **Phân tích Code Reconciliation**:
   - Xem logic filter và query source ở `internal/service/recon/recon_tier_b.go` và `internal/service/recon/recon_dest_query.go`.
   - Kiểm tra xem có so sánh khớp chuỗi (`export-jobs` vs `export_jobs`) hay lỗi format tên bảng nào gây ra `noop` không.

## Phase 2: Thiết kế kỹ thuật & Phê duyệt giải pháp (Technical Design & Approval)
1. Đề xuất giải pháp sửa đổi trong file `09_tasks_solution_recon_export_jobs.md`.
2. Trình User phê duyệt giải pháp trước khi Muscle tiến hành sửa code.

## Phase 3: Thực thi & Kiểm thử (Implementation & Verification)
1. Thực hiện sửa đổi (code hoặc config).
2. Chạy lại test suite để đảm bảo không bị regression.
3. Trigger đối soát và heal thủ công qua API của bảng `export-jobs`.
4. Xác minh bản ghi đã được heal thành công và kết quả đối soát trả về `ok`.
