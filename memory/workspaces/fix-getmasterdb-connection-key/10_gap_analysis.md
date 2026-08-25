# 10_gap_analysis.md — Gap & Architecture Analysis

## 1. Phân tích lỗ hổng kiến trúc (Architectural Gap Analysis)

### Gap 1: Thiếu tính toàn vẹn định danh bảng Master (Master Table Identity Gap)
- **Mô tả:** Hệ thống trong giai đoạn đầu thiết kế giả định tên bảng master là duy nhất trên toàn hệ thống (`master_table`). Khi mở rộng sang multi-tenant / multi-service (nhiều schema khác nhau như `master_bidv_connector_service`, `master_goopay`, `master_core`), các câu lệnh `LIMIT 1` dựa trên chỉ `master_table` trở thành điểm lỗi chết người (Race Condition & Ambiguity).
- **Trạng thái:** Đang được xử lý toàn diện qua chiến dịch Schema-Qualified FQN.

### Gap 2: Connection Manager Hardcoded Destination (Multi-Tenant Master Connection Gap)
- **Mô tả:** `ConnectionManager.GetMasterDB(ctx, key)` và `GetShadowDB(ctx, key)` discard tham số `key`, hardcode vào `RoleDestination` và `RoleShadow`.
- **Trạng thái:** Cần lên kế hoạch mở rộng `Registry` để hỗ trợ lookup DSN theo `connection_code` từ bảng `cdc_system.connection_registry`.

### Gap 3: Bẫy kiểu dữ liệu & chuỗi NULL trong PostgreSQL (Postgres Type & Concat Semantics Gap)
- **Mô tả:** Thói quen dùng toán tử `||` trong Postgres mà không xét trường hợp một trong các cột là `NULL` dẫn đến toàn bộ chuỗi bị `NULL`.
- **Trạng thái:** Đã thiết lập chuẩn phòng thủ `COALESCE(NULLIF(..., ''), 'public')` làm quy tắc mẫu.
