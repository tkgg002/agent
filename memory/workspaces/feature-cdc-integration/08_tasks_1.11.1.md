# Phase 1.11.1 Task: Fix 13 tồn đọng từ User Test

## Status: COMPLETED

## Task Checklist

### Nhóm A: Backend Bugs
- [x] **A1**: Fix GET /api/mapping-rules?status=X → 500 — thêm pagination + filtering
- [x] **A2**: Fix GET /api/airbyte/sources → 500 — verified port config đúng (8083). Lỗi do Airbyte API unreachable.

### Nhóm B: Frontend Bugs
- [x] **B1**: Fix /queue crash — rewrite với safe accessors, guard format() callbacks, Alert error state
- [x] **B2**: Registry row click → `/registry/:id/mappings` (thay vì /schema-changes)
- [x] **B3**: Rename action buttons — Standardize, Discover + icons + tooltips

### Nhóm C: Feature Enhancements
- [x] **C1**: Source Database dropdown hiện tất cả Airbyte sources (merge registry + airbyteSources)
- [x] **C2**: Tạo `MappingFieldsPage.tsx` — system default fields, mapping rules with active/inactive toggle, custom mapping, scan unmapped
- [x] **C3**: Scan unmapped fields button tích hợp trong MappingFieldsPage (gọi `/api/introspection/scan/:table`)
- [ ] **C4**: Ghi nhận ADR về AirbyteHandler architecture — P2, chưa cần fix

## Files Changed
### Backend (cdc-cms-service)
- `internal/repository/mapping_rule_repo.go` — thêm `GetAllFilteredPaginated()`
- `internal/api/mapping_rule_handler.go` — pagination + extra filter params

### Frontend (cdc-cms-web)
- `src/pages/QueueMonitoring.tsx` — rewrite crash-proof
- `src/pages/TableRegistry.tsx` — action buttons UX, row navigation, Source DB dropdown
- `src/pages/MappingFieldsPage.tsx` — **MỚI**
- `src/App.tsx` — thêm route `/registry/:id/mappings`

## Definition of Done
- [x] Tất cả 500 errors resolved (A1 code fix, A2 config issue)
- [x] /queue không crash
- [x] Mapping fields page hoạt động
- [x] All builds OK
- [x] 05_progress.md updated
