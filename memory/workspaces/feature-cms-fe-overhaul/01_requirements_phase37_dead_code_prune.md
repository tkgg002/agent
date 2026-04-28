# Phase 37 — Dead-code Prune & Deprecate (Hồi tố)

> **Note**: Bộ docs này được tạo hồi tố vì session GPT-5.4 thực hiện Phase 37 bị hit usage limit lúc 8:51 AM trước khi kịp tạo file vật lý. Nội dung dựng lại từ log `cdc-system/Untitled-2.ini` dòng 2026-2054.

## 1. Mục tiêu (Goal)

Sau Phase 36, các helper compatibility đã được hardening schema-aware nhưng nhiều path vẫn ở trạng thái "prepared without caller". Phase 37 phải **ra quyết định dứt điểm** cho từng path: giữ làm compatibility reserve, hay prune hẳn.

Không thêm lớp đệm nữa — nguyên tắc "Simplicity First" của CLAUDE.md Rule 6.

## 2. Phạm vi (Scope)

Audit caller thật cho 3 nhóm:
- `centralized-data-service/internal/service/transform_service.go`
- `centralized-data-service/internal/handler/event_bridge.go`
- `cdc-cms-service/internal/repository/registry_repo.go` (helpers raw SQL legacy)

## 3. Phân loại quyết định

| Component | Tình trạng caller | Quyết định | Lý do |
|-----------|------------------|-----------|-------|
| `TransformService` | Không có caller runtime | **Prune** (xóa file) | Dead code, không có giá trị compatibility |
| Helper raw SQL `registry_repo` (`ScanRawKeys`, `PerformBackfill`, `GetDBColumns`) | Không có caller | **Prune** | Không có caller, thay thế bằng `*InSchema()` đã có từ Phase 36 |
| `EventBridge` | Còn test + có giá trị nếu poller quay lại | **Compatibility Reserve** | Đóng dấu rõ không thuộc runtime chính, không xóa |

## 4. Definition of Done

- [ ] `transform_service.go` đã xóa khỏi codebase
- [ ] Helper raw SQL public-centric trong `registry_repo.go` đã xóa
- [ ] `event_bridge.go` có comment/note đánh dấu "compatibility reserve, không thuộc runtime chính"
- [ ] `go test ./...` (cms + worker) pass
- [ ] Không có import broken sau khi prune
- [ ] Workspace docs Phase 37 đầy đủ
