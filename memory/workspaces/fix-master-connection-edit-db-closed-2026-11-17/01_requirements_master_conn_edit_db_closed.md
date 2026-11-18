---
tier: T2
mode: standard
suffix: master_conn_edit_db_closed
workspace: fix-master-connection-edit-db-closed-2026-11-17
created_at: 2026-11-17T11:05:00+07:00
---
# 01 Requirements — Kiểm Tra Lỗi `sql: database is closed` & Dữ Liệu Hardcode Trong DB Không Update Khi Chỉnh Sửa Kết Nối Đích (`Master Connection`)

## 1. Yêu cầu của User
- **Thao tác thực tế:** Chỉnh sửa Kết nối đích (`Master Connection`) có `connection_code = postgrestestingpaymentv2` (`Engine Type: PostgreSQL`, `Port: 5000`, `Schema: public`), đổi trường **Tên Database (`database_name`)** từ `payment-service-v2` sang một tên database khác.
- **Hiện tượng lỗi thực tế (tại `11:00:19 17/11/2026`):**
  - Hàng loạt bản ghi trong Activity Log bị lỗi ở các operation `transmute`, `transmute_sub`, `transmute_poison_pill` (triggered bởi `kafka-consumer-hook`) trên các scope:
    - `shadow_payment_service.payments` -> `public.cdc_payment` & `public.cdc_payment_payload`
    - `shadow_payment_bill_service.payment_bills` -> `public.cdc_payment_bill` & `public.cdc_payment_bill_payload`
  - Thông báo lỗi chi tiết:
    `failed to verify master table existence: sql: database is closed` (dẫn tới cô lập nhầm poison pill `poison pill isolated (source_ids=[...]): failed to verify master table existence: sql: database is closed`).
- **Câu hỏi trọng tâm của User:**
  > "kiểm tra cho anh, chỗ nào trong db đang hard, và ko update khi master edit"
  - Cần kiểm tra toàn diện cả 2 vế:
    1. **Trong Database (`cdc_system.*`):** Những bảng/cột nào đang lưu cứng (`hardcode` / snapshot copy) thông tin của kết nối đích (`database_name`, `connection_string` / `dsn`, `host`, `port`, `schema_name`, `target_db`...) tại thời điểm tạo `master_connections` hoặc `master_binding` / `connection_registry` mà khi sửa `Master Connection` trên UI lại không được cập nhật đồng bộ?
    2. **Trong Runtime Code (`cdc-cms-service` & `centralized-data-service`):** Vì sao sau khi edit `database_name` của `postgrestestingpaymentv2`, luồng `transmute` / `transmute_sub` lại nhận lỗi `failed to verify master table existence: sql: database is closed` (thay vì mở kết nối tới DB mới) và tại sao lỗi hạ tầng kết nối (`sql: database is closed`) lại bị phân loại nhầm thành `transmute_poison_pill`?

## 2. Ràng buộc Thực thi (Technical Constraints từ `GEMINI.md` & `lessons/INDEX.md`)
- **L065 (No Mutex Across Network I/O & Clean Resource Lifecycle):** Kiểm tra kỹ cơ chế lock/cache của `ConnectionManager`, `TransmuterModule`, `ReconDestAgent`, `MasterDDLGenerator` hoặc struct giữ `*gorm.DB` / `*sql.DB` trực tiếp khiến khi `InvalidatePool` đóng `sqlDB.Close()` thì struct khác vẫn giữ pointer `*gorm.DB` đã đóng!
- **L068 (Clean DB Handle Lifecycle):** Không đóng đột ngột hoặc giữ stale reference `*gorm.DB` / `*sql.DB` qua các module con sau khi connection pool đã bị invalidate.
- **L070 & Rule #20 (End-to-End Secret & PII Masking):** Tuyệt đối che mật khẩu/URI/IP nhạy cảm (`postgres://***:***@<host>:5000/...`) khi truy vấn kiểm tra DB và khi ghi tài liệu vào `agent/memory/`.
- **Rule #12 (Zero DB Cheat / Read-Only Investigation):** Trong bước điều tra S2, chỉ đọc (`SELECT` có mask password) trên DB `cdc_system` để chứng minh dữ liệu thực tế; tuyệt đối không chạy `UPDATE`/`ALTER` trực tiếp vào DB.
- **Rule #1, #9, #13 (Brain Propose-Only):** Xác minh 100% bằng chứng `file:line` + dữ liệu DB thực tế, phản biện ngược tại `DG0`, thiết kế **ĐÚNG 1 PHƯƠNG ÁN TỐI ƯU NHẤT** kèm **Code Demo** trong `09_tasks_solution_master_conn_edit_db_closed.md` + trên khung chat, qua cổng `DG1/DG2` và **DỪNG CHỜ `APPROVE`**.
