# Architecture Decisions

## ADR 1: Giữ nguyên cơ chế báo lỗi khi shadow table trống ở phía Backend
- **Quyết định**: Giữ nguyên việc backend (`centralized-data-service`) trả về error `"shadow table %s is empty"` khi scan-fields một bảng không có data trong shadow DB.
- **Lý do**: Đây là hành vi đúng đắn vì không thể quét metadata/fields của một bảng rỗng. Backend cần ghi nhận rõ ràng lý do lỗi vào `cdc_activity_log`.
- **Giải pháp**: Xử lý việc dừng polling trên Frontend/API gateway để tránh loading vô hạn.
