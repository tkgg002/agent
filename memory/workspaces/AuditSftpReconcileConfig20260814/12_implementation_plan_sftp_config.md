# 12_implementation_plan: Kế Hoạch Triển Khai Chi Tiết Của AI (AI Execution Plan)

## 1. Mục Tiêu Thực Thi
Đóng vai **Brain (Chairman & Architect)** lập toàn bộ kế hoạch kỹ thuật, sơ đồ dữ liệu, cấu hình JSON Spec và quy chuẩn vận hành SFTP Server để đưa hệ thống Kafka Connect FS SFTP Reconcile lên Production.

## 2. Các Bước Triển Khai Thực Tế

### Bước 1: Rà soát & Đánh giá Rủi ro
- Xác nhận tính chính xác của 6 Tripwires (Password plain-text, `host.docker.internal`, DDoS poll, Partial read, Header shotgun config, Null Key ordering).
- Bổ sung 3 Cổng An toàn Enterprise (DLQ, Decimal Precision, Partitioning).

### Bước 2: Thiết kế Golden JSON Connector Specification
- Thiết lập `FileConfigProvider` biến bảo mật credentials.
- Điều chỉnh `policy.sleepy.sleep = 60000`, `policy.recursive = false`.
- Thêm SMT `ValueToKey` + `ExtractField` theo `transaction_id`.
- Cấu hình DLQ `errors.tolerance = all`.

### Bước 3: Thiết lập Quy chuẩn Vận hành SFTP Server
- Đơn giản hóa đường dẫn URI, tôn trọng chuẩn Production.
- Đưa ra Atomic Upload Protocol (`.tmp` -> `.csv`).
- Soạn thảo Shell Script Cronjob Cleanup & Archive file cũ > 3 ngày.

### Bước 4: Khởi Tạo Bộ Quản Trị Tri Thức Vật Lý
- Tạo đầy đủ 13 file tài liệu chuẩn trong Workspace `AuditSftpReconcileConfig20260814`.
- Chạy kiểm định Linter `verify_governance.py` đảm bảo đạt 100% Governance Audit PASSED.
