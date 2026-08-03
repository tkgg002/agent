# Xoá Shadow & Xoá Master

## Mô tả

Implement 2 chức năng xoá:
1. **Xoá Shadow Binding**: Xoá 1 `shadow_binding` row + cascade xoá toàn bộ `master_binding` và `mapping_rule_master` gắn với nó.
2. **Xoá Master Binding**: Chỉ xoá 1 `master_binding` row + các `mapping_rule_master` tương ứng (KHÔNG xoá shadow).

## Open Questions

> [!IMPORTANT]
> **Có cần guard thêm không?** Hiện tại plan chặn các case sau:
> - Xoá Shadow: chặn nếu shadow đang `is_active = true` (phải tắt trước)
> - Xoá Master: chặn nếu master `schema_status = 'approved'` (cần reject trước)
>
> Có cần thêm guard nào khác không? (VD: kiểm tra xem schedule transmute còn active không trước khi xoá master?)

> [!WARNING]
> **Destructive action**: Xoá shadow/master là không thể hoàn tác. Plan sẽ dùng:
> - `ConfirmDestructiveModal` (đã có sẵn) để confirm trước khi xoá
> - Cần yêu cầu `reason` ≥ 10 ký tự từ user
> - Backend đi qua `destructiveChain` (OpsAdmin + Idempotency + Audit middleware)

## Proposed Changes

---

### Backend — cdc-cms-service

#### [NEW] [delete_shadow_binding.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/shadow/delete_shadow_binding.go)

Command xoá shadow binding:
```go
// Cascade: xoá master_binding + mapping_rule_master → xoá shadow_binding
// Guard: chặn nếu shadow is_active = true
type DeleteShadowBindingCommand struct {
    ID        int64  `json:"id"`
    UpdatedBy string `json:"updated_by,omitempty"`
}
```

Logic:
1. Fetch shadow binding row → kiểm tra `is_active`, nếu true → trả lỗi `shadow_binding_active`
2. Tìm tất cả `master_binding` có `shadow_binding_id = ID`
3. Với mỗi master_binding: gọi `DeleteClonedRules(masterBindingID)` → gọi `DeleteMasterBinding(masterBindingID)`
4. Xoá `shadow_binding` bằng method mới `DeleteShadowBinding(id)` trên `ShadowBindingRepo`

#### [MODIFY] [repository.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/ports/repository.go)

Extend `ShadowBindingRepo` và `MasterRepo`:
```go
// ShadowBindingRepo — thêm 2 method
type ShadowBindingRepo interface {
    UpdateActiveStatus(ctx, id int64, isActive bool) (int64, error)
    // NEW:
    GetByID(ctx, id int64) (*ShadowBindingInfo, error)
    DeleteShadowBinding(ctx, id int64) error
    ListMasterBindingsByShadowID(ctx, shadowID int64) ([]int64, error) // trả masterBindingIDs
}
```

> [!NOTE]
> `ShadowBindingInfo` là struct nhỏ chứa `{ID, IsActive}` — đủ để guard check.

#### [NEW] [shadow_binding_repo_gorm_delete.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/source/)

Thêm implementation cho 3 method mới trong `shadowBindingRepoGorm` (hoặc file mới trong package shadow):
```go
func (r) GetByID(ctx, id) (*ports.ShadowBindingInfo, error)
func (r) ListMasterBindingsByShadowID(ctx, shadowID int64) ([]int64, error)
func (r) DeleteShadowBinding(ctx, id int64) error
```

#### [NEW] [delete_master_binding.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/master/delete_master_binding.go)

Command xoá master binding (only, không xoá shadow):
```go
// Guard: chặn nếu schema_status = 'approved'
type DeleteMasterBindingCommand struct {
    ID        int64  `json:"id"`
    UpdatedBy string `json:"updated_by,omitempty"`
}
```

Logic:
1. Fetch `master_binding` by ID → kiểm tra `schema_status`, nếu `approved` → lỗi `cannot_delete_approved_master`
2. `DeleteClonedRules(ID)` — xoá mapping rules
3. `DeleteMasterBinding(ID)` — xoá binding

#### [MODIFY] [master_registry_handler.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/master/master_registry_handler.go)

Thêm method `Delete(c *fiber.Ctx) error` vào `MasterRegistryHandler`:
- Parse `:id` (int64)
- Gọi `DeleteMasterBindingHandler.Handle(ctx, cmd)`
- Error mapping: 404, 409, 500

#### [MODIFY] [shadow_binding_actions_handler.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/shadow/shadow_binding_actions_handler.go)

Thêm method `Delete(c *fiber.Ctx) error` vào `ShadowBindingActionsHandler`:
- Parse `:id` (int64)
- Gọi command `DeleteShadowBindingCommand` qua bus
- Error mapping

#### [MODIFY] [router.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/router/router.go)

Thêm 2 destructive route:
```go
// Xoá shadow binding (cascade: xoá master đi kèm)
api.Delete("/v1/shadow-bindings/:id", append(destructiveChain, h.Shadow.BindingActions.Delete)...)

// Xoá master binding only
api.Delete("/v1/masters/:id", append(destructiveChain, h.Master.Registry.Delete)...)
```

> [!NOTE]
> Route `/v1/masters/:id` dùng `id` (int64) thay vì `:name` string để tránh ambiguity và nhất quán với pattern DELETE thông thường.

#### [MODIFY] [server.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/server/server.go)

Khởi tạo và inject `DeleteShadowBindingHandler` và `DeleteMasterBindingHandler` vào command bus.

---

### Frontend — cdc-cms-web

#### [MODIFY] [TableRegistry.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/TableRegistry.tsx)

Thêm nút **"Xoá Shadow"** ở column `Shadow Actions` trong tab **Shadow Bindings** (`bindingColumns`):

```tsx
{
  title: 'Actions',
  render: (_, row: ShadowBindingRow) => (
    <Button
      size="small"
      danger
      icon={<DeleteOutlined />}
      onClick={() => handleDeleteShadow(row)}
    >
      Xoá Shadow
    </Button>
  )
}
```

Dùng `ConfirmDestructiveModal` để confirm trước khi xoá:
- Hiển thị cảnh báo "Sẽ xoá toàn bộ master bindings đi kèm"
- Yêu cầu nhập reason
- Gọi `DELETE /api/v1/shadow-bindings/:id`

#### [MODIFY] [MasterRegistry.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/MasterRegistry.tsx)

Thêm nút **"Xoá"** vào column `Actions` của master table:

```tsx
<Button
  size="small"
  danger
  icon={<DeleteOutlined />}
  disabled={r.schema_status === 'approved'} // phải reject trước
  onClick={() => setDeleteRow(r)}
>
  Xoá
</Button>
```

Dùng `ConfirmDestructiveModal`:
- Warning: "Xoá master binding và toàn bộ mapping rules. Shadow KHÔNG bị ảnh hưởng."
- Gọi `DELETE /api/v1/masters/:id`

---

## Verification Plan

### Automated Tests
- Build check: `cd cdc-cms-service && go build ./...`
- Frontend build: `cd cdc-cms-web && npm run build`

### Manual Verification
1. Tạo shadow binding → tạo 2 master binding gắn vào → Xoá shadow → verify cả 2 master và shadow đã biến mất trong DB
2. Tạo master binding → Xoá master → verify chỉ master bị xoá, shadow vẫn còn
3. Thử xoá shadow đang `is_active = true` → verify bị chặn với lỗi rõ ràng
4. Thử xoá master `schema_status = 'approved'` → verify bị chặn
