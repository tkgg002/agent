# Plan / Kế Hoạch: Phân Tích Luồng Đối Soát Tier 2 XOR-Hash (Tier 2 XOR-Hash Reconciliation Analysis)

## English Version
### Goal
Investigate and analyze the Tier 2 window-based XOR-hash reconciliation flow in the `centralized-data-service` to verify its implementation and confirm that it is strictly read-only.

### Steps
1. **Source Code Mapping**: Locate the active files under `internal/service/recon/`.
2. **ReconCore Control Flow Analysis**: Analyze `RunTier2` in `recon_tier_a.go` to understand how time windows are calculated, XOR-hashes are queried, and drifts are processed.
3. **Source Agent Verification**: Inspect `HashWindow` and `ListIDTsInWindow` in `recon_hash.go` and `recon_stream.go` to verify MongoDB and PostgreSQL source queries.
4. **Destination Agent Verification**: Inspect `HashWindow` and `ListIDTsInWindow` in `recon_dest_hash.go` and `recon_dest_query.go` to verify replica DB queries.
5. **Read-only Connection Audit**: Audit the helper `readOnlyDB(ctx)` in `recon_dest_agent.go` to verify transaction security (e.g., read-only isolation level, defer rollback).
6. **Documentation**: Write a comprehensive, Vietnamese technical analysis report mapping all findings.
7: **ID & Timestamp Mapping Audit**: Verify that the primary key ID and timestamp fields conform to the new shadow registration schema (e.g., `_id` as PK, `lastUpdatedAt` as Timestamp Field).
8: **Output & Healing Flow Audit**: Investigate what results are returned after Tier 2, where they are stored, and how the subsequent heal process runs to fix the drifts.
9: **CMS-Web Flow Audit**: Analyze how CMS-Web reads reports and triggers the heal action.
10: **Heal Execution & State Transition Audit**: Investigate the case reported by the user (why report 7 had mismatched but did not heal them instantly, why report 8 transitioned to "healed" with null stale_ids/mismatched).
11: **Heal Path Audit (FetchAndWriteByIDs vs NATS Debezium Signal)**: Verify whether the active heal path goes through direct MongoDB write (FetchAndWriteByIDs) or sends a Debezium signal to "cdc.cmd.debezium-signal".
12: **False Mismatched Detection Audit**: Inspect why the Tier 2 reconciliation logic detected 3 mismatched IDs when there were actually none (false mismatched/drift detection).
13: **DDL Timezone Core Mapping Audit**: Investigate the core DDL generation and type mapping logic to find why timestamp fields are mapped to TIMESTAMP instead of TIMESTAMPTZ in the shadow database.
14: **Heal Routing Inconsistency Audit**: Audit heal routing logic to find why window drift (Branch 1) always sends Debezium signals and doesn't fallback to the direct write path (Branch 2) like Safety Net when Debezium is disabled.
15: **FE/BE Co-Design & Routing Redesign specs**: Design the FE modal controls (radio buttons for Window vs Full-diff, dynamic from/to inputs, max 30-day validation) and modify the BE routing logic to execute based on the mode param, replacing the Debezium NATS signal with direct FetchAndWriteByIDs in Window mode.
16: **Technical Solution Mapping**: Document the detailed code changes in `09_tasks_solution_tier2_check.md` for user review and approval.
17: **Frontend (FE) Execution**: Implement UI modifications in `useReconStatus.ts`, `ConfirmDestructiveModal.tsx`, and `DataIntegrity.tsx` to complete the CMS-Web modal redesign.

---

## Tiếng Việt
### Mục tiêu
Nghiên cứu và phân tích luồng đối soát Tier 2 (window-based XOR-hash) trong dịch vụ `centralized-data-service` để xác minh thiết kế kỹ thuật và đảm bảo luồng hoạt động chỉ đọc (read-only), không sửa đổi dữ liệu.

### Các bước thực hiện
1. **Xác định file mã nguồn**: Định vị các file đang hoạt động trong thư mục `internal/service/recon/`.
2. **Phân tích luồng kiểm soát ReconCore**: Phân tích hàm `RunTier2` trong `recon_tier_a.go` để hiểu cách tính toán cửa sổ thời gian, cách so sánh XOR-hash và cách drill-down xử lý sai lệch.
3. **Xác minh Source Agent**: Kiểm tra các hàm `HashWindow` và `ListIDTsInWindow` trong `recon_hash.go` và `recon_stream.go` đối với nguồn MongoDB và PostgreSQL.
4. **Xác minh Destination Agent**: Kiểm tra các hàm `HashWindow` và `ListIDTsInWindow` trong `recon_dest_hash.go` và `recon_dest_query.go` đối với đích Shadow DB.
5. **Kiểm tra kết nối chỉ đọc**: Kiểm tra kỹ helper `readOnlyDB(ctx)` trong `recon_dest_agent.go` để xác thực mức độ cô lập transaction chỉ đọc và việc rollback kết nối.
6. **Xác thực việc ánh xạ ID và Timestamp**: Kiểm tra xem PK Field (`_id` / PK thật) và Timestamp Field (`lastUpdatedAt` / trường timestamp cấu hình) đã được phân giải và truyền chính xác trong luồng Tier 2 chưa.
7. **Phân tích Đầu ra & Luồng Heal**: Điều tra kết quả đối soát Tier 2 nhận được là gì, lưu trữ ở đâu, và cơ chế heal sửa lỗi dữ liệu sai lệch diễn ra như thế nào.
8. **Phân tích Luồng CMS-Web**: Khảo sát cách giao diện quản trị đọc báo cáo đối soát và trigger lệnh heal.
9. **Điều tra Hành vi Chạy Heal thực tế**: Giải thích chi tiết trường hợp của User (tại sao report 7 có mismatched, tại sao report 8 chuyển thành "healed" và các mảng ID trở thành `null`).
10. **Audit đường chạy Heal thực tế**: Kiểm tra cụ thể xem khi trigger heal nó chạy vào `FetchAndWriteByIDs` (ghi trực tiếp) hay bắn Debezium incremental snapshot signal qua NATS topic `cdc.cmd.debezium-signal`.
11. **Audit việc phát hiện mismatched giả (false mismatched)**: Tìm nguyên nhân tại sao hệ thống phát hiện 3 bản ghi mismatched giả.
12. **Audit kiểu dữ liệu DDL ở core**: Tìm nguyên nhân tại sao timestamp trong Shadow DB lại được map thành kiểu TIMESTAMP (without time zone) thay vì TIMESTAMPTZ (with time zone).
13. **Audit sự không đồng nhất phân nhánh heal**: Audit logic điều phối của luồng heal để tìm hiểu tại sao khi phát hiện drift trong window (Nhánh 1) hệ thống lại luôn bắn Debezium signal mà không tự động fallback sang direct write path (Nhánh 2) giống như Safety Net khi Debezium bị ngắt.
14. **Thiết kế chi tiết FE/BE và Redesign Routing**: Thiết kế chi tiết các nút FE (Radio button chọn Window/Full-diff, input from/to, logic disable/validate 30 ngày) và sửa BE routing theo mode param mới, dùng FetchAndWriteByIDs thay thế cho Debezium NATS signal ở Window mode.
15. **Lập hồ sơ giải pháp kỹ thuật chi tiết**: Tài liệu hóa các đoạn mã diff cần thiết vào tệp `09_tasks_solution_tier2_check.md` và chờ phê duyệt.
16. **Thực thi Frontend (FE)**: Cập nhật `useReconStatus.ts`, `ConfirmDestructiveModal.tsx` và `DataIntegrity.tsx` để hoàn thành modal điều khiển Heal trên CMS-Web.
17. **Tích Hợp Tracing (OTEL)**: Bổ sung OpenTelemetry traces (parent & child spans) cho từng bước trong `StreamIDsInTimeRange`, `TimeBoundedDiffMissingFromShadow` và `healSegmentA` để tăng khả năng quan sát (observability).
18. **Tài liệu hóa**: Viết báo cáo phân tích kỹ thuật chi tiết bằng tiếng Việt ghi nhận toàn bộ kết quả phân tích.
