# Báo cáo Audit Kiến trúc Hệ thống (centralized-data-service)

Tài liệu này tổng hợp kết quả rà soát toàn diện (architectural audit) của hệ thống `centralized-data-service`, đối chiếu với bản đồ phân tầng tiêu chuẩn Clean Architecture và Domain Zones.

---

## 1. Tóm tắt kết quả (Executive Summary)

* **Điểm sáng (Clean Areas)**:
  * **Độ sạch Dependencies**: Hoàn toàn tuân thủ quy tắc chiều phụ thuộc. Tầng `model`, `repository`, `service`, `naming`, và `activity` không hề import ngược các tầng cao hơn. Các package trong `pkgs/` hoàn toàn decoupled (không import `internal/`).
  * **Biên dịch & Kiểm thử**: 100% code biên dịch thành công (`go build ./...`) và vượt qua toàn bộ unit tests (`go test ./...`).
* **Điểm cần khắc phục (Architectural Gaps)**:
  * **Rò rỉ logic DB tại tầng Handler**: Rất nhiều handler tự ý giữ `gorm.DB` và viết truy vấn GORM / raw SQL trực tiếp thay vì thông qua Service/Repository.
  * **Bypass Service có sẵn**: Điển hình là `ScanHandler` tự viết lại logic quét field thô và ghi DB thay vì gọi `ScanService`.
  * **Bất đồng bộ tên miền (Domain Misalignment)**: Có sự không nhất quán giữa thư mục tên miền của Model, Repository và Service (ví dụ: `model/system` vs `repository/recon`).

---

## 2. Danh sách lỗi Kiến trúc chi tiết (Detailed Findings)

### A. Rò rỉ Nghiệp vụ & Thao tác DB trực tiếp tại tầng Handler (Business Logic Leakage)

> [!WARNING]
> Tầng Handler chỉ nên chịu trách nhiệm nhận sự kiện (NATS payload, HTTP request), validate payload cơ bản, gọi Service thích hợp và phản hồi kết quả. Việc nhúng logic SQL/GORM tại Handler làm giảm khả năng test độc lập và vi phạm Clean Architecture.

#### 1. Lạm dụng truy vấn thô và ghi DB trong `ScanHandler`
* **File**: `internal/handler/recon/scan_handler_discover.go`
* **Chi tiết lỗi**:
  * Dùng `h.DB.Table(...)` và `h.DB.Create(&rule)` tại dòng 73, 79, 133 để tự kiểm tra trạng thái map và tạo Mapping Rule.
  * Dùng `h.DB.Transaction(func(tx *gorm.DB) error { ... })` tại dòng 407 để chèn dòng vào bảng `mapping_rule_v2` và chạy câu lệnh `INSERT INTO cdc_system.mapping_rule_master ... RETURNING id` thô.
* **Tác động**: Hệ thống đã có sẵn `ScanService` (`internal/service/source/scan_service.go`) thực hiện logic này nhưng bị handler bypass hoàn toàn.

#### 2. Thao tác database thô tại `DiscoverHandler`
* **File**: `internal/handler/source/discover_handler.go` và `discover_handler_mongo.go`
* **Chi tiết lỗi**:
  * Tự truy vấn registry bằng GORM raw SQL tại dòng 188: `h.DB.Raw("SELECT id FROM cdc_system.source_object_registry ...")` thay vì inject `SourceObjectRegistryRepo`.
  * Tự kiểm tra registry ID bằng `h.DB.Where("id = ?", registryID).First(&registry)` tại dòng 13 của `discover_handler_mongo.go`.

#### 3. Tạo Activity Log thủ công tại `ReconHandler`
* **File**: `internal/handler/recon/recon_handler.go`
* **Chi tiết lỗi**:
  * Dòng 139 tự gọi `h.db.Create(&system.ActivityLog{...})` để ghi log tiến trình.
  * Có sẵn `ActivityLogger` Service (`internal/service/governance/activity_logger.go`) nhưng không được sử dụng.
  * Tự truy vấn registry ID bằng `h.db.Where("id = ?", id).First(&entry)` tại dòng 126.

#### 4. Ghi trực tiếp logs đồng bộ thất bại tại `BatchBuffer`
* **File**: `internal/handler/shadow/batch_buffer.go`
* **Chi tiết lỗi**:
  * Gọi `bb.db.Create(bb.buildFailedSyncLog(...))` tại dòng 340 để lưu log đồng bộ lỗi của shadow, bypass qua repo.

#### 5. Hàm Helper ghi log hoạt động nằm sai tầng
* **File**: `internal/handler/orchestration/activity_logger.go`
* **Chi tiết lỗi**:
  * File này định nghĩa hàm global `WriteActivity(ctx, db, op, table...)` tự gọi `db.Create(&system.ActivityLog{})`. Việc đặt helper ghi DB trực tiếp vào package `handler/orchestration` là hoàn toàn sai phân tầng.

---

### B. Bất đồng bộ Miền nghiệp vụ (Domain Misalignment Gaps)

> [!IMPORTANT]
> Để cấu trúc Features/Domain rõ ràng, thư mục con của Model, Repository, Service và Handler cần phải tương thích 1-1 về mặt ngữ nghĩa miền nghiệp vụ.

| Đối tượng | Tầng Model | Tầng Repository | Tầng Service | Tầng Handler | Trạng thái |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **Reconciliation / Reports** | `model/system/` | `repository/recon/` | `service/recon/` | `handler/recon/` | **Lệch Model** (`system` vs `recon`) |
| **Activity Log** | `model/system/` | *Không có* | `service/governance/` | `handler/orchestration/` | **Lệch toàn bộ** (`system`, `governance`, `orchestration`) |
| **Provisioning Orchestrator** | *Không có* | *Không có* | `service/recon/` | `handler/orchestration/` | **Lệch Service** (`recon` vs `orchestration`) |
| **Table Registry** | `model/source/` | `repository/source/` (file `registry_repo.go`) | `service/source/` | `handler/source/` | **Lệch tên file Repo** (`registry_repo` vs `TableRegistry`) |

#### Chi tiết bất đồng bộ nghiêm trọng:
1. **Provisioning Orchestrator**: `ProvisioningOrchestrator` quản lý máy trạng thái (state machine) luồng khởi tạo môi trường (Provisioning Mode), hoàn toàn không liên quan đến đối soát dữ liệu (Reconciliation). Tuy nhiên nó lại được đặt tại `internal/service/recon/provisioning_orchestrator.go`.
2. **Reconciliation Models**: `reconciliation_report.go` và `snapshot_dlq.go` thuộc về miền `recon` nhưng lại đặt trong `model/system/`.

---

## 3. Kế hoạch hành động Đề xuất (Fix Action Plan)

Để giải quyết triệt để các khoảng cách kiến trúc trên mà không gây ảnh hưởng đến tính ổn định của hệ thống, chúng ta đề xuất lộ trình refactor 3 bước:

### Bước 1: Chuẩn hóa Domain Folder & Tên gọi (Domain Alignment)
1. **Di chuyển Model**: Chuyển `reconciliation_report.go` và `snapshot_dlq.go` từ `internal/model/system/` sang `internal/model/recon/`.
2. **Di chuyển Service**: Chuyển `provisioning_orchestrator.go` từ `internal/service/recon/` sang `internal/service/orchestration/` (hoặc tạo mới package `service/orchestration`).
3. **Đổi tên Repository**: Đổi tên `registry_repo.go` thành `table_registry_repo.go` và đổi tên struct thành `TableRegistryRepo` để khớp 1-1 với model `TableRegistry`.

### Bước 2: Trục xuất logic DB khỏi Handlers (Decouple Database from Handlers)
1. **ScanHandler**: Thay vì viết SQL thủ công, inject `ScanService` vào `ScanHandler` và ủy thác toàn bộ logic scan thô qua service.
2. **DiscoverHandler**: Inject `SourceObjectRegistryRepo` vào `DiscoverHandler` để đọc Registry ID thay vì dùng `h.DB.Raw`.
3. **ReconHandler & BatchBuffer**:
   * Sử dụng `ActivityLogger` Service để ghi nhận log thay vì gọi `h.db.Create(&system.ActivityLog{})`.
   * Sử dụng `FailedSyncLogRepo` để ghi nhận lỗi đồng bộ thay vì gọi `bb.db.Create(...)` trực tiếp.

### Bước 3: Loại bỏ Helper sai tầng
* Loại bỏ file `internal/handler/orchestration/activity_logger.go`. Gom tất cả các nơi đang gọi `WriteActivity` chuyển sang sử dụng `ActivityLogger` Service được inject chính thống.
