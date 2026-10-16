# DANH SÁCH NHIỆM VỤ (TASK LIST): SỬA 3 LỖI SOURCE COUNT & CASCADING METADATA

- [ ] **Task 1: Backend CMS Connection API**
  - Mở rộng repo `masterRepoGorm` hỗ trợ lọc theo role hoặc lấy tất cả (`ListAllConnections`).
  - Cập nhật `MasterRegistryHandler.ListConnections` đọc `role` / `role_type` hoặc route `/connections`.
  - Đăng ký route `GET /v1/connections` và `GET /connections` trong `router.go`.
  - Build kiểm thử `cdc-cms-service`.

- [ ] **Task 2: Worker Server Panic Error Catching**
  - Cập nhật `server_scheduler.go`: Bổ sung ghi `last_error` vào DB khi recover panic.
  - Chạy unit test và build `centralized-data-service`.

- [ ] **Task 3: Frontend Cascading Selector & Error Tooltip**
  - Cập nhật query nạp connection trong `ActivityManager.tsx` lấy toàn bộ connection.
  - Lọc connection chính xác theo `selectedLayer` (`source`, `shadow`, `master`).
  - Đồng bộ lineage namespace ở Tầng 3 theo `selectedConn`: Đọc `default_schema`, `default_database` và các schema binding của connection đó.
  - Nâng cấp cột `last_error` với Tooltip định dạng đẹp, trực quan, hỗ trợ xem trọn vẹn lỗi và sao chép mã lỗi.
  - Chạy `npm run build` trên `cdc-cms-web`.

- [ ] **Task 4: Audit & Bàn Giao**
  - Verify Gate G1–G8 trên môi trường thực tế.
  - Cập nhật `05_progress.md` (Append ONLY).
  - Xuất bản báo cáo hoàn tất `11_report_source_count_cascading_and_error.md`.
