# Workspace Context: Refactoring Master Mapping Rule Handler

## Objective
Tái cấu trúc file `internal/api/master_mapping_rule_handler.go` (đang đóng vai trò God Object) theo chuẩn **Screaming Architecture & Hexagonal Architecture/CQRS** để:
1. Giải phóng các logic nghiệp vụ và truy vấn DB ra khỏi tầng HTTP Delivery (API).
2. Định nghĩa các port mới (`MasterDDLPublisher` và các method bổ sung trong `MasterRuleRepository`).
3. Tách biệt rõ ràng các Use Cases thành Command & Query cụ thể (Write & Read Models).
4. Chuẩn hóa hạ tầng persistence (GORM SQL trong `master_rule_repo_gorm.go`) và hạ tầng messaging (NATS client trong `nats_master_ddl_publisher.go`).
5. Di chuyển các hàm utility dùng chung ra `pkgs/utils/` và `internal/naming/`.

## Governance Compliance
- Không có lỗi vi phạm quy trình Governance (tạo workspace trước khi thực hiện nghiên cứu chuyên sâu hoặc chỉnh sửa code).
- Gốc rễ lỗi vi phạm quy trình Governance trước đó: Không có (N/A).
