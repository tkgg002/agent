# Detailed File Audit Report: Infrastructure Drainage Refactor

Báo cáo chi tiết này đánh giá toàn bộ các tệp tin đã được chỉnh sửa hoặc di chuyển trong quá trình refactor di chuyển các phụ thuộc cơ sở dữ liệu (GORM) và NATS ra khỏi tầng API/Application và di chuyển chúng hoàn toàn vào tầng Infrastructure.

---

## I. Danh Sách Tệp Tin Rà Soát Chi Tiết

### 1. Tầng Ports (Interfaces - `internal/app/ports/`)

#### `reload_publisher.go` [NEW]
- **Nó làm gì**: Định nghĩa cổng giao tiếp `ReloadPublisher` với phương thức `PublishReload(ctx context.Context, shadowTable string) error` nhằm phát tín hiệu reload sang hệ thống messaging.
- **Vì sao ở vị trí đó**: Theo chuẩn thiết kế Hexagonal Architecture, tất cả các interfaces mô tả cổng giao tiếp hướng ngoại (outbound ports) mà tầng ứng dụng cần gọi phải được định nghĩa tại `internal/app/ports/`. Việc này ngăn chặn sự rò rỉ (leak) của các thư viện messaging cụ thể như NATS vào lõi nghiệp vụ.

#### `publisher.go` & `repository.go` [MODIFY]
- **Nó làm gì**: Cung cấp các định nghĩa port dùng chung cho việc xuất bản event và thao tác cơ sở dữ liệu.
- **Vì sao ở vị trí đó**: Nằm ở thư mục gốc của Ports để đóng vai trò làm hợp đồng giao tiếp chuẩn hóa cho toàn bộ ứng dụng.

---

### 2. Tầng Infrastructure - Persistence (DB Adapters - `internal/infra/persistence/...`)

#### `shadow/bridge_status_repo_gorm.go` [MOVE & MODIFY]
- **Nó làm gì**: Triển khai interface truy vấn thông tin trạng thái bridge `shadow.BridgeStatusReader`. Thực hiện đếm số dòng dữ liệu thực tế, kiểm tra sự tồn tại của bảng shadow và cột `_raw_data`.
- **Vì sao ở vị trí đó**: Đã được di chuyển từ gói `recon/` sang gói `shadow/` vì chức năng chính của nó là truy xuất metadata cho shadow tables (thuộc domain `shadow`), giúp cô lập và phân tách sạch sẽ trách nhiệm giữa các gói persistence.

#### `scheduler/job_repo_gorm.go` [MODIFY]
- **Nó làm gì**: Quản lý lưu trữ trạng thái các công việc lập lịch (job runs), thực hiện mapping lỗi `gorm.ErrRecordNotFound` sang `ports.ErrRecordNotFound`.
- **Vì sao ở vị trí đó**: Thuộc gói persistence domain `scheduler`, chịu trách nhiệm làm việc trực tiếp với bảng `cdc_system.job_runs`.

#### `recon/recon_read_repo_gorm.go` [MODIFY]
- **Nó làm gì**: Chứa các câu lệnh SQL thô phức tạp dùng để kết xuất dữ liệu báo cáo đối chiếu (reconciliation report), đếm số dòng bảng, truy vấn lịch sử chạy backfill và danh sách logs đồng bộ bị lỗi.
- **Vì sao ở vị trí đó**: Thuộc gói persistence domain `recon`, đóng vai trò là read adapter phục vụ các query handlers của domain Reconciliation.

#### `governance/approval_service.go` & `provisioning_orchestrator.go` [MODIFY]
- **Nó làm gì**: Quản lý luồng phê duyệt và điều phối khởi tạo tài nguyên DB vật lý (bảng, cột, trigger).
- **Vì sao ở vị trí đó**: Thuộc domain `governance`, quản trị dữ liệu master và các thay đổi cấu trúc bảng.

#### `master/mapping_rule_repo_gorm.go`, `master_mapping_rule_repo_gorm.go`, `master_repo_gorm.go` [MODIFY]
- **Nó làm gì**: Đọc/ghi cấu hình các mapping rules và liên kết bảng master.
- **Vì sao ở vị trí đó**: Thuộc domain `master` dùng để lưu trữ cấu hình ánh xạ dữ liệu đích (master plane).

#### `shadow/shadow_automator.go` [MODIFY]
- **Nó làm gì**: Tự động tạo cấu trúc bảng shadow DDL và đính kèm trigger sonyflake tạo khóa tự động.
- **Vì sao ở vị trí đó**: Nằm trong persistence domain `shadow`, đóng vai trò tự động hóa việc đồng bộ cấu trúc cho các bảng shadow.

#### `system/activity_logger.go` [MODIFY]
- **Nó làm gì**: Ghi log hoạt động của admin vào bảng `cdc_system.cdc_activity_log`.
- **Vì sao ở vị trí đó**: Thuộc domain `system` phục vụ giám sát toàn hệ thống.

---

### 3. Tầng Infrastructure - Messaging (NATS Adapters - `internal/infra/messaging/`)

#### `nats_publisher.go` [MODIFY]
- **Nó làm gì**: Triển khai cụ thể các ports `Publisher` và `ReloadPublisher` bằng thư viện NATS. Thực hiện gửi tin nhắn reload dạng `reload.<shadowTable>` lên NATS JetStream/Core.
- **Vì sao ở vị trí đó**: Là thành phần adapter gửi tin nhắn (Outbound Adapter) sử dụng công nghệ NATS, bắt buộc nằm trong tầng Infrastructure.

---

### 4. Tầng Application (Commands & Handlers - `internal/app/...`)

#### `commands/source/register_registry.go`, `bulk_register_registry.go`, `update_registry.go` [MODIFY]
- **Nó làm gì**: Xử lý nghiệp vụ đăng ký/cập nhật bảng nguồn. Chuyển từ việc trực tiếp gọi client NATS sang gọi port `ports.ReloadPublisher`.
- **Vì sao ở vị trí đó**: Nằm ở tầng Application để thực thi ca sử dụng (Use Case) của gói `source`.

#### `commands/master/update_mapping_rule.go` [MODIFY]
- **Nó làm gì**: Cập nhật mapping rules và gửi tín hiệu reload qua `ports.ReloadPublisher`.
- **Vì sao ở vị trí đó**: Thực thi ca sử dụng nghiệp vụ thay đổi ánh xạ dữ liệu của gói `master`.

---

### 5. Tầng API Presentation (HTTP Controllers - `internal/api/...`)

- **Nó làm gì**: Nhận HTTP request, bind DTOs, gọi command/query ports và trả về HTTP status code phù hợp. Loại bỏ hoàn toàn sự phụ thuộc trực tiếp vào `gorm.ErrRecordNotFound` bằng cách map thông qua `ports.ErrRecordNotFound`.
- **Vì sao ở vị trí đó**: Là cổng vào của ứng dụng (Inbound Adapter), thực hiện đón nhận các yêu cầu REST API.

---

## II. Đánh Giá Về "God Functions" (Hàm Siêu Trách Nhiệm)

Trong đợt rà soát này, tôi đã phân tích độ dài và mức độ phức tạp của từng hàm trong các tệp tin di chuyển/chỉnh sửa để đảm bảo không có hàm nào rơi vào mô hình phản khuôn mẫu "God function" (Hàm ôm đồm quá nhiều việc, khó bảo trì):

1. **Các hàm truy vấn dữ liệu (Persistence Layer)**:
   - Các hàm như `ProbeBridgeStatus`, `ResolveDispatchScopeBySourceObjectID`, `ListLatest`... đều có tính tập trung cao. Chúng chỉ nhận tham số, thực thi câu lệnh SQL thô hoặc ORM xác định, map kết quả và trả về.
   - Hàm dài nhất trong lớp này là `ApproveSchemaTx` (khoảng 86 dòng) trong `master_repo_gorm.go`. Tuy nhiên, đây là logic transactional phức tạp đòi hỏi cập nhật nhiều bảng liên quan trong cùng một Transaction (mapping_rule, shadow_binding, master_binding) để đảm bảo tính nhất quán (Atomicity). Việc viết gộp trong một transaction block là đúng thiết kế để kiểm soát rollback.

2. **Các Command Handlers (Application Layer)**:
   - Các hàm `Handle` trong Command Handlers hiện tại rất tinh gọn (trung bình dưới 40 dòng). Chúng chỉ làm nhiệm vụ lấy thực thể từ repository, thực hiện thay đổi trạng thái, lưu lại thông qua repository và gửi tín hiệu reload thông qua `ports.ReloadPublisher`.

3. **Các HTTP Handlers (API Layer)**:
   - Toàn bộ HTTP Handlers chỉ thực hiện chuyển đổi DTO và điều phối sang App Layer. Các logic check lỗi cơ sở dữ liệu đã được tổng quát hóa thông qua `ports.ErrRecordNotFound`, giúp code controller cực kỳ ngắn gọn và sáng rõ.

**Kết luận**: **Không còn tệp tin hay hàm nào chứa "God function"** trong phạm vi các tệp refactor của workspace này.
