# Thiết kế Kỹ thuật Chi tiết: Tái cấu trúc và Di chuyển Handlers (Cập nhật)

Tài liệu này đặc tả chi tiết kế hoạch thay đổi mã nguồn để loại bỏ các vi phạm phân tầng (layering violations) và gom nhóm các handler về đúng package chức năng theo **Hệ quy chiếu chức năng thư mục cốt lõi**.

---

## 1. Tái cấu trúc `base/base_handler.go` (Domain-agnostic Base Utilities)

### 1.1 Mục tiêu
Loại bỏ hoàn toàn các phụ thuộc liên quan đến `metadata` registry, `source` model, `master` targets, hoặc tiến trình `provisioning` khỏi gói `base`. Struct `BaseHandler` sẽ chỉ chứa các tài nguyên dùng chung không phụ thuộc nghiệp vụ: `DB`, `NatsConn`, và `Logger`.

### 1.2 Thay đổi cấu trúc struct
```go
// TRƯỚC
type BaseHandler struct {
	DB           *gorm.DB
	NatsConn     *nats.Conn
	Logger       *zap.Logger
	RegistryRepo RegistryResolver
	Metadata     metadata.MetadataRegistry
}

// SAU
type BaseHandler struct {
	DB           *gorm.DB
	NatsConn     *nats.Conn
	Logger       *zap.Logger
}
```

### 1.3 Các phương thức bị loại bỏ/trục xuất khỏi `base/base_handler.go`
1. Interface `RegistryResolver` -> Xóa bỏ khỏi `base/`.
2. Trục xuất sang gói `source/`:
   - `SetMetadataRegistry`
   - `SetRegistryResolver`
   - `ResolveTableConfigByID`
   - `ListActiveTableConfigs`
3. Trục xuất sang gói `master/`:
   - `ResolveTargetTableConfig`
   - `ResolveTargetRoute`
   - `ResolveTargetSchema` (Chuyển sang gói `master/` do việc phân giải đích thuộc về miền trách nhiệm của `master`).
4. Trục xuất sang gói `orchestration/`:
   - `WriteActivity` (Ghi nhật ký tiến trình hoạt động thuộc về Nhạc trưởng).

### 1.4 Các tiện ích được GIỮ LẠI tại `base/`
- Các phương thức xuất bản tin NATS: `NatsPublish`, `PublishResult`, `PublishResultWithSubject`.
- Các helper kết nối HTTP: `ConnectGET`, `ConnectPOST`, `ConnectPUT`, `ConnectCall`, `LogCommandResult`.
- Các helper an toàn SQL và Sanitization: `IsSafeIdent`, `IsSafeType`, `SanitizeAdminError`, `SanitizeAdminResultMap`, `SanitizeAdminFields`.
- Các hàm chuyển đổi ánh xạ kiểu dữ liệu thô: `NormalizeMappingRuleDataType`, `BuildCastExpr`.
- Các helper kiểm tra sự tồn tại của bảng thô trong DB: `TableExists`, `TableExistsInSchema`, `HasColumn`, `HasColumnInSchema`.

---

## 2. Di chuyển `provisioning_emit.go` khỏi `base/`
Toàn bộ file `internal/handler/base/provisioning_emit.go` sẽ được di chuyển sang gói `orchestration/` (tại `internal/handler/orchestration/provisioning_emit.go` và đổi sang `package orchestration`).
- Các struct được di chuyển: `StepResult`, `stepCompletedPayload`.
- Hàm được di chuyển: `EmitStepCompleted`.

---

## 3. Khai báo explicit Dependency cho các Handlers
Các struct handler thuộc các package khác nhau nếu cần tương tác với `MetadataRegistry` sẽ tự khai báo dependency cục bộ và phương thức inject tương ứng.

### 3.1 `DiscoverHandler` (`internal/handler/source/discover_handler.go`)
```go
type DiscoverHandler struct {
	base.BaseHandler
	metadataRegistry metadata.MetadataRegistry
	...
}

func (h *DiscoverHandler) SetMetadataRegistry(m metadata.MetadataRegistry) {
	h.metadataRegistry = m
}
```

### 3.2 `ScanHandler` (`internal/handler/recon/scan_handler.go`)
```go
type ScanHandler struct {
	base.BaseHandler
	metadataRegistry metadata.MetadataRegistry
	...
}

func (h *ScanHandler) SetMetadataRegistry(m metadata.MetadataRegistry) {
	h.metadataRegistry = m
}
```

### 3.3 `BatchTransformHandler` (`internal/handler/shadow/batch_transform_handler.go`)
```go
type BatchTransformHandler struct {
	base.BaseHandler
	metadataRegistry metadata.MetadataRegistry
	...
}

func (h *BatchTransformHandler) SetMetadataRegistry(m metadata.MetadataRegistry) {
	h.metadataRegistry = m
}
```

### 3.4 `SchemaDDLHandler` (`internal/handler/shadow/schema_ddl_handler.go`)
```go
type SchemaDDLHandler struct {
	base.BaseHandler
	metadataRegistry metadata.MetadataRegistry
	...
}

func (h *SchemaDDLHandler) SetMetadataRegistry(m metadata.MetadataRegistry) {
	h.metadataRegistry = m
}
```

---

## 4. Phân bổ lại tập tin vào các Thư mục Chức năng (Đúng Quy Chiếu)

### 4.1 Thư mục `source/` (Đăng ký Nguồn, Connectors & Discovery)
1. `internal/handler/orchestration/discover_handler.go` -> `internal/handler/source/discover_handler.go` (đổi sang `package source`).
2. `internal/handler/orchestration/mongo_discover_handler.go` -> `internal/handler/source/mongo_discover_handler.go` (đổi sang `package source`).
3. **Logic thao tác DB Nguồn (Bóc tách từ `provisioning_step_handlers.go`)**:
   - Di chuyển các hàm helper sau sang gói `source/` (có thể đặt tại `internal/handler/source/discovery_utils.go` hoặc gộp vào file thích hợp):
     - `isMongoEngine`
     - `fetchSourceEngine`
     - `preflightMongoSource`
     - `inferSourceColumns`, `inferPGCols`, `inferMySQLCols`, `inferMongoCols` (Việc đọc mẫu dữ liệu DB nguồn phải do `source/` đảm nhận).

### 4.2 Thư mục `shadow/` (Kafka Ingestion, Buffer & Shadow Target)
1. `internal/handler/master/schema_ddl_handler.go` -> `internal/handler/shadow/schema_ddl_handler.go` (đổi sang `package shadow`).
   - Phương thức `bridgeMappingRulesToV2` được rút ra và đặt tại `base/base_handler.go` để làm hàm utils public dùng chung.
2. **Logic Shadow Bind**:
   - Tạo file `internal/handler/shadow/provisioning_shadow_bind.go` chứa struct `ShadowBindHandler` và phương thức `HandleShadowBind`.
   - Chứa các helper `resolveShadowTarget` và `upsertShadowBinding`.
   - Để báo cáo trạng thái hoàn thành step lên Control Plane, handler sẽ gọi qua hàm `EmitStepCompleted` của package `orchestration`.

### 4.3 Thư mục `master/` (Master Target Operations & DML Swaps)
1. `internal/handler/master/master_ddl_handler.go` -> Giữ nguyên.
2. `internal/handler/master/transmute_handler.go` -> **BẮT BUỘC GIỮ NGUYÊN** tại `master/`. Không di chuyển sang `orchestration/` do Transmute thực hiện SQL DML trực tiếp vào DB Đích (Data Plane).

### 4.4 Thư mục `recon/` (Data Reconcile, DLQ & Self-Healing)
1. `internal/handler/orchestration/dlq_handler.go` -> `internal/handler/recon/dlq_handler.go` (đổi sang `package recon`).
2. `internal/handler/orchestration/dlq_state_machine.go` -> `internal/handler/recon/dlq_state_machine.go` (đổi sang `package recon`).
3. `internal/handler/orchestration/scan_handler.go` -> `internal/handler/recon/scan_handler.go` (đổi sang `package recon`).

### 4.5 Thư mục `orchestration/` (State Machine & Control Plane)
1. `internal/handler/orchestration/provisioning_handler.go` -> Giữ nguyên.
2. `internal/handler/orchestration/snapshot_runner_handler.go` -> Giữ nguyên.
3. **Schedule Enable Step**:
   - Tạo file `internal/handler/orchestration/provisioning_schedule_enable.go` chứa struct `ScheduleEnableHandler` với phương thức `HandleScheduleEnable` và struct `scheduleEnableRequest`.
   - Sử dụng helper `EmitStepCompleted` từ file `provisioning_emit.go` cùng thư mục.
4. Xóa tệp tin rỗng `internal/handler/orchestration/provisioning_step_handlers.go`.

---

## 5. Cập nhật Đăng ký Dependency (`worker_server_init.go`)
- Sửa đổi toàn bộ các đường dẫn import tương ứng với các package mới.
- Khởi tạo và liên kết các dependencies thông qua các hàm cài đặt explicit cục bộ (như `SetMetadataRegistry`).
