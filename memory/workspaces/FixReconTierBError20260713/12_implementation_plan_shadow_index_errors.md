# Kế hoạch triển khai - Khắc phục lỗi SQL Index trên Shadow Table (SQLSTATE 42703)

## 1. Vấn đề hiện tại
* Khi thực hiện đề xuất index hoặc tự động tạo default index trên Shadow Table, hệ thống lấy trường `timestamp_field` từ cấu hình registry và thực thi câu lệnh `CREATE INDEX` mà không xác minh xem cột đó có tồn tại vật lý trong bảng đích hay chưa.
* Nếu bảng chưa được di trú (provision) đầy đủ các cột nghiệp vụ, PostgreSQL sẽ trả về lỗi `ERROR: column "lastUpdatedAt" does not exist (SQLSTATE 42703)`, làm sập luồng đồng bộ hoặc báo lỗi nghiêm trọng trên giao diện CMS UI.

## 2. Giải pháp Đề xuất

### A. Kiểm tra tồn tại của cột trong `GetRecommendations`
* **Tệp thay đổi**: `internal/service/governance/index_manager.go`
* **Mô tả**:
  1. Trong hàm `GetRecommendations`, trước khi tạo khuyến nghị index cho trường timestamp, chúng ta thực hiện truy vấn `information_schema.columns` để lấy danh sách cột thực tế của bảng đích.
  2. Thực hiện so khớp cột linh hoạt giữa metadata registry và physical schema (chấp nhận khớp chính xác, khớp snake_case hoặc khớp không phân biệt chữ hoa chữ thường).
  3. Chỉ đưa ra khuyến nghị index nếu cột thực sự tồn tại trong DB.

### B. Kiểm tra tồn tại của cột trong tự động hóa provisioning
* **Tệp thay đổi**: `internal/handler/shadow/schema_ddl_handler.go`
* **Mô tả**:
  1. Trong `HandleCreateDefaultColumns`, trước khi gọi `CreateIndex` cho trường timestamp, thực hiện truy vấn danh sách cột thực tế của bảng thông qua `schemaAdapter` (hoặc trực tiếp qua `information_schema.columns`).
  2. Nếu cột không tồn tại, in cảnh báo (`WARN`) và bỏ qua an toàn thay vì thực thi DDL lỗi.

### C. Cải tiến an toàn cho `CreateIndexConcurrently`
* **Tệp thay đổi**: `internal/service/governance/index_manager.go`
* **Mô tả**:
  1. Trong `CreateIndexConcurrently`, bổ sung bước rà soát danh sách cột đầu vào bằng cách truy vấn danh sách cột thực tế của bảng từ `information_schema.columns`.
  2. Nếu có cột bất kỳ không tồn tại vật lý, dừng lại và trả về lỗi rõ ràng: `column "<col>" does not exist in table <schema>.<table>` thay vì chạy câu lệnh DDL trực tiếp để tránh lỗi SQL thô.

### D. Bổ sung Unit Test
* **Tệp thay đổi**: `internal/service/governance/index_manager_test.go`
* **Mô tả**:
  1. Viết unit test `TestIndexManager_NonExistentColumn` tạo bảng kiểm thử và cố gắng tạo index trên cột không tồn tại, xác minh lỗi trả về khớp với mô tả mong muốn.

---

## 3. Kế hoạch xác minh (Verification Plan)

### Kiểm thử Tự động (Automated Tests)
* Chạy toàn bộ test suite của package governance:
  ```bash
  go test -v ./internal/service/governance/...
  ```
* Chạy biên dịch dự án:
  ```bash
  go build ./internal/... ./cmd/... ./pkgs/...
  ```
* Chạy linter kiểm toán quy trình:
  ```bash
  python3 agent/tooling/verify_governance.py
  ```
