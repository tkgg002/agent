# Kế hoạch Di chuyển resolveMasterBindingRef lên ReconBase (Symmetric Design)

Kế hoạch này giải quyết phản hồi của User về tính đối xứng cấu trúc bằng cách di chuyển phương thức hỗ trợ Segment B (`resolveMasterBindingRef`) từ `CheckHandler` lên lớp cha dùng chung `ReconBase` (tương tự như cách định vị của helper Segment A `resolveTargetTableConfig`).

## User Review Required

> [!IMPORTANT]
> - Chúng ta sẽ di chuyển `resolveMasterBindingRef` từ `recon_check_handler.go` sang `recon_base_handler.go`.
> - Việc này đảm bảo tính nhất quán (Symmetric Design) cho tất cả các handler thừa kế từ `ReconBase`.

## Proposed Changes

### 1. Base Handler Component

#### [MODIFY] [recon_base_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_base_handler.go)
- Thêm phương thức `resolveMasterBindingRef` vào `ReconBase`:
  ```go
  func (h *ReconBase) resolveMasterBindingRef(ctx context.Context, masterTable string) *servicerecon.MasterBindingRef {
      for _, ref := range h.reconCore.ListActiveMasterBindings(ctx) {
          if ref.MasterTable == masterTable || ref.MasterRel() == masterTable {
              return &ref
          }
      }
      return nil
  }
  ```

### 2. Check Handler Component

#### [MODIFY] [recon_check_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_check_handler.go)
- Loại bỏ hoàn toàn phương thức `resolveMasterBindingRef` khỏi `CheckHandler`.
- Do `CheckHandler` kế thừa từ `ReconBase` qua cấu trúc nhúng (`*ReconBase`), tất cả các cuộc gọi tới `h.resolveMasterBindingRef` sẽ tự động chuyển hướng lên `ReconBase` và hoạt động bình thường.

---

## Verification Plan

### Automated Tests
- Chạy kiểm tra biên dịch gói:
  ```bash
  go build ./internal/handler/recon/...
  ```
- Chạy toàn bộ unit tests của package `recon`:
  ```bash
  go test -v ./internal/handler/recon/...
  ```
