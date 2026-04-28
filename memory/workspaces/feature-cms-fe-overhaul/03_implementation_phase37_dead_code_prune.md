# Phase 37 — Implementation

## Files changed (theo log)

| File | Loại | +/- |
|------|------|-----|
| `cdc-system/cdc-cms-service/internal/repository/registry_repo.go` | Edit (xóa methods legacy) | 0/-49 |
| `cdc-system/centralized-data-service/internal/handler/event_bridge.go` | Edit (compatibility note) | +11/-3 |
| `cdc-system/centralized-data-service/internal/service/transform_service.go` | **Deleted** | 0/-112 |

**Tổng**: 3 files changed, +11/-164.

## Chi tiết thay đổi

### 1. Xóa `transform_service.go`
- Lý do: 0 caller runtime, dead code.
- Phase 36 đã thêm `metadata MetadataRegistry` và schema-aware helper cho service này, nhưng vì không có caller thật → quyết định không kéo wrapper lên mà prune luôn.

### 2. Xóa methods legacy trong `registry_repo.go`
- Xóa: `ScanRawKeys`, `PerformBackfill`, `GetDBColumns` (-49 dòng).
- Giữ lại: `ScanRawKeysInSchema`, `PerformBackfillInSchema`, `GetDBColumnsInSchema` (đã thêm ở Phase 36).

### 3. `event_bridge.go` — Compatibility Reserve
- Thêm comment header đánh dấu file này là compatibility reserve, không thuộc runtime chính hiện tại.
- Schema-aware helpers (Phase 36) được giữ nguyên để nếu poller được restore, code đã sẵn sàng.

## Notes về session bị block

Session GPT-5.4 đã thực hiện việc edit/delete code thành công (3 files changed) nhưng bị hit usage limit khi đang chuyển sang tạo bộ docs phase. Bộ docs này được dựng lại hồi tố từ log để giữ đúng quy tắc "Full Doc Set" (CLAUDE.md Rule 7).
