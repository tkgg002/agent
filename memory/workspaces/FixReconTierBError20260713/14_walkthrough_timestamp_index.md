# Kết quả triển khai (Walkthrough) - Đề xuất index trên Timestamp Field

## 1. Các thay đổi đã thực hiện

### A. Khuyến nghị index tự động qua Governance API
* **File sửa đổi**: `internal/service/governance/index_manager.go`
* **Nội dung**:
  * Tích hợp cơ chế đề xuất index cho timestamp field vào hàm `GetRecommendations`.
  * Quét từ `cdc_system.cdc_table_registry` và `cdc_system.mapping_rule_v2` để kiểm tra sự tồn tại của index cho cột timestamp.
  * Nếu thiếu, trả về đề xuất khuyến nghị khởi tạo index để tối ưu hóa truy vấn `MaxWindowTs`.
  * Người dùng có thể xem và chủ động tạo index một cách an toàn thông qua tính năng **Quản lý Indexes (Shadow Table)** trên UI.

### B. Bổ sung Unit Test xác thực
* **File sửa đổi**: `internal/service/governance/index_manager_test.go`
* **Nội dung**:
  * Viết thêm test case `TestIndexManager_GetRecommendations_TimestampField` để kiểm thử toàn diện cả 2 trường hợp:
    1. Trả về đề xuất index khi chưa được đánh chỉ mục.
    2. Không đề xuất index nếu cột timestamp đó đã có index hợp lệ.
  * Thiết lập mock data đầy đủ quan hệ khóa ngoại (foreign key chain) gồm `connection_registry`, `source_object_registry`, `cdc_table_registry`, và `mapping_rule_v2`.

### C. Kiểm tra tồn tại của cột trong Database thực tế trước khi đề xuất hoặc chạy DDL Index
* **File sửa đổi**: `internal/service/governance/index_manager.go` & `internal/handler/shadow/schema_ddl_handler.go`
* **Nội dung**:
  * Kiểm tra tồn tại vật lý của cột mục tiêu bằng cách truy vấn từ `information_schema.columns` trước khi đưa ra đề xuất index hoặc thực thi tạo index thông qua DDL.
  * Trong `CreateIndexConcurrently`, nếu cột không tồn tại, trả về lỗi chi tiết chỉ rõ cột nào bị thiếu thay vì để ném lỗi SQL thô từ Postgres.
  * Trong `HandleCreateDefaultColumns` (`schema_ddl_handler.go`), nếu cột timestamp không tồn tại, ghi nhận log `WARN` và bỏ qua an toàn thay vì làm sập tiến trình.

### D. Bổ sung Unit Test cho Cột Không Tồn Tại
* **File sửa đổi**: `internal/service/governance/index_manager_test.go`
* **Nội dung**:
  * Thêm unit test `TestIndexManager_NonExistentColumn` xác minh hàm `CreateIndexConcurrently` trả về lỗi chính xác khi cố gắng tạo index trên một cột không tồn tại thực tế.

## 2. Kết quả kiểm thử (Verification)
* Chạy test suite `go test -v ./internal/service/governance/...` hoàn toàn thành công:
  ```
  === RUN   TestIndexManager_Lifecycle
  --- PASS: TestIndexManager_Lifecycle (0.15s)
  === RUN   TestIndexManager_UnsafeWhere
  --- PASS: TestIndexManager_UnsafeWhere (0.05s)
  === RUN   TestIndexManager_GetRecommendations
  --- PASS: TestIndexManager_GetRecommendations (0.00s)
  === RUN   TestIndexManager_GetRecommendations_TimestampField
  --- PASS: TestIndexManager_GetRecommendations_TimestampField (0.06s)
  === RUN   TestIndexManager_NonExistentColumn
  --- PASS: TestIndexManager_NonExistentColumn (0.08s)
  PASS
  ok  	centralized-data-service/internal/service/governance	1.197s
  ```
* Chạy `python3 agent/tooling/verify_governance.py` vượt qua Audit Governance thành công:
  ```
   ⛳ GOVERNANCE AUDIT PASSED 🟢 (Workspace: FixReconTierBError20260713)
  ```

