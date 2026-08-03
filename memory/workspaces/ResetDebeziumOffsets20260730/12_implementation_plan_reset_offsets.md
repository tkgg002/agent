# Kế Hoạch Triển Khai Chi Tiết - Reset Debezium Connector Offsets

## Các Bước Triển Khai (Execution Steps)

1. **Bước 1 (Backend Infra Client)**: Thêm method `DeleteOffsets` vào `internal/infra/http/kafka_connect.go`.
2. **Bước 2 (Backend API Handler)**: Thêm method `ResetOffsets` vào `internal/api/source/system_connectors_handler.go`.
3. **Bước 3 (Backend Router)**: Đăng ký endpoint mới trong `internal/router/router.go`.
4. **Bước 4 (Frontend UI Component)**: Thêm nút "Xóa Offset" + Modal xác nhận trong `SourceConnectors.tsx`.
5. **Bước 5 (Kiểm thử & Báo cáo)**: Biên dịch Go backend và build Vite frontend để kiểm tra syntax/type checking.
