# Implementation Plan: Refactor Raw SQL out of governance Commands (Hexagonal Refactor) / Kế hoạch thực thi: Bóc tách các câu lệnh SQL thô ra khỏi các Commands thuộc governance

## Goal Description / Mô tả mục tiêu
Currently, the command handlers for schema proposals (`ApproveSchemaProposalHandler` and `RejectSchemaProposalHandler`) directly depend on `gorm.DB` and execute raw SQL statements using `.Exec()` and `.Raw()`. This violates the Hexagonal Architecture boundary by leaking infrastructure details into the Application layer.
This plan refactors these raw SQL calls into `SchemaProposalRepo` (Infrastructure Adapter) and updates the handlers to use the clean repository interface.

Hiện tại, các command handlers xử lý schema proposals (`ApproveSchemaProposalHandler` và `RejectSchemaProposalHandler`) đang phụ thuộc trực tiếp vào `gorm.DB` và thực thi các câu lệnh SQL thô thông qua `.Exec()` và `.Raw()`. Điều này vi phạm ranh giới kiến trúc Hexagonal do để lộ chi tiết hạ tầng (GORM/SQL) vào tầng Application.
Kế hoạch này sẽ tái cấu trúc, di chuyển các câu lệnh SQL thô này vào `SchemaProposalRepo` (Infrastructure Adapter) và cập nhật các handlers để sử dụng interface repository sạch.

---

## Proposed Changes / Thay đổi đề xuất

### 1. Domain Ports Layer / Tầng Domain Ports

#### [MODIFY] [repository.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/ports/repository.go)
Add two new business methods to `SchemaProposalRepo` interface to encapsulate transaction and database operations for approving and rejecting proposals.

Bổ sung hai phương thức nghiệp vụ mới vào interface `SchemaProposalRepo` nhằm đóng gói các thao tác cơ sở dữ liệu và transaction khi phê duyệt và từ chối proposal.

```go
type SchemaProposalRepo interface {
	List(ctx context.Context, tableName *string, status *string) ([]model.SchemaProposal, error)
	GetByID(ctx context.Context, id int64) (*model.SchemaProposal, error)
	Update(ctx context.Context, proposal *model.SchemaProposal) error
	
	// New methods / Các phương thức mới
	Approve(ctx context.Context, proposal *model.SchemaProposal, finalType string, finalPath, finalFn string, reviewedBy string, overrideType, overrideJSONPath, overrideTransformFn *string) error
	Reject(ctx context.Context, proposalID string, reviewedBy string, reason string) error
}
```

---

### 2. Infrastructure Layer (Persistence Adapter) / Tầng Hạ tầng (Persistence Adapter)

#### [MODIFY] [schema_proposal_repo_gorm.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/governance/schema_proposal_repo_gorm.go)
Implement the `Approve` and `Reject` methods using GORM.
- For `Approve`: Wrap all DB changes (checking shadow schema, running `ALTER TABLE`, inserting mapping rules, and updating proposal status) inside a transaction. Handle failure path by marking status as `failed` with error message.
- For `Reject`: Perform updates with CAS guard (`status = 'pending'`). Return a clean error string `not_pending_or_not_found` if no rows affected.

Triển khai phương thức `Approve` và `Reject` sử dụng GORM.
- Đối với `Approve`: Gom nhóm toàn bộ thay đổi DB (kiểm tra shadow schema, chạy `ALTER TABLE`, insert mapping rules, và cập nhật trạng thái proposal) vào trong một transaction. Xử lý kịch bản lỗi bằng cách cập nhật trạng thái thành `failed` kèm thông tin lỗi.
- Đối với `Reject`: Thực hiện cập nhật với cơ chế CAS guard (`status = 'pending'`). Trả về lỗi `not_pending_or_not_found` nếu không có bản ghi nào bị tác động.

---

### 3. Application Layer (Commands) / Tầng Application (Commands)

#### [MODIFY] [approve_schema_proposal.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/governance/approve_schema_proposal.go)
- Change dependency from `db *gorm.DB` to `repo ports.SchemaProposalRepo`.
- Remove GORM and SQL imports.
- Load proposal using `h.repo.GetByID`.
- Keep validation logic (type validation, identifiers regex matching).
- Delegate database transaction execution to `h.repo.Approve`.

- Thay đổi dependency từ `db *gorm.DB` sang `repo ports.SchemaProposalRepo`.
- Loại bỏ các package import liên quan đến GORM và SQL.
- Tải thông tin proposal thông qua `h.repo.GetByID`.
- Giữ nguyên các logic kiểm tra (loại dữ liệu, regex tên bảng/cột).
- Ủy quyền thực thi transaction cơ sở dữ liệu cho phương thức `h.repo.Approve`.

#### [MODIFY] [reject_schema_proposal.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/governance/reject_schema_proposal.go)
- Change dependency from `db *gorm.DB` to `repo ports.SchemaProposalRepo`.
- Delegate database update to `h.repo.Reject`. Map returned error to `ErrSchemaProposalNotPendingOrNotFound`.

- Thay đổi dependency từ `db *gorm.DB` sang `repo ports.SchemaProposalRepo`.
- Ủy quyền cập nhật cơ sở dữ liệu cho phương thức `h.repo.Reject`. Ánh xạ lỗi trả về thành `ErrSchemaProposalNotPendingOrNotFound`.

---

### 4. Composition Root (Dependency Injection) / Điểm Khởi Tạo (Dependency Injection)

#### [MODIFY] [server.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/server/server.go)
- Initialize `schemaProposalRepo` using `persistenceGov.NewSchemaProposalRepo(db)`.
- Pass `schemaProposalRepo` to command handlers instead of `db`.

- Khởi tạo `schemaProposalRepo` sử dụng `persistenceGov.NewSchemaProposalRepo(db)`.
- Truyền `schemaProposalRepo` vào các command handlers thay vì `db`.

---

## Verification Plan / Kế hoạch kiểm thử

### Automated Tests / Kiểm thử tự động
Run compilation and test suite:
Biên dịch dự án và chạy bộ kiểm thử:
```bash
go build ./...
go test ./internal/app/commands/governance/...
go test ./...
```

### Manual Verification / Kiểm thử thủ công
Verify that the project builds correctly and there are no lint/vet errors.
Xác minh dự án biên dịch thành công và không có lỗi lint/vet.
