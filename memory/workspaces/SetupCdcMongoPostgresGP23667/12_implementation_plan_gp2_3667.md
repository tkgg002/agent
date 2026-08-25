# 12_implementation_plan_gp2_3667.md - Kế hoạch Triển khai Chi tiết của AI

## Kế hoạch Triển khai Chi tiết cho GP2-3667

### Bước 1: Khai báo Workspace & Bộ tài liệu Chuẩn
- Khởi tạo thư mục `agent/memory/workspaces/SetupCdcMongoPostgresGP23667`
- Viết đầy đủ 13 file tài liệu chuẩn hoá theo Hiến pháp `GEMINI.md`.

### Bước 2: Khảo sát & Đề xuất DDL & Mapping Rules
- Xây dựng DDL PostgreSQL Master table `transaction_history` tối ưu cho các truy vấn của Core Transaction History Service.
- Thiết lập mapping rules bóc tách ExtJSON BSON fields.

### Bước 3: Đăng ký & Kích hoạt qua CMS API
- Gọi API đăng ký Connector, Source Object `transaction_history`, Shadow Schema, Master Table Registry và Sync Mapping Rules.
- Kiểm tra tính sẵn sàng của Debezium MongoDB Connector và CDC Worker/Transmuter Engine.

### Bước 4: Kiểm thử Verification & Báo cáo
- Thực hiện kiểm thử 8 Quality Gates (G1 - G8).
- Ghi log vào `05_progress_gp2_3667.md`, chạy Process Linter `python3 agent/tooling/verify_governance.py`.
