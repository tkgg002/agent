# 12_implementation_plan.md — Kế hoạch triển khai chi tiết của AI

## 1. Mục tiêu kỹ thuật
Bổ sung toàn diện hỗ trợ trường `snapshot_max_rps` vào CMS Backend và CMS Frontend, cho phép operator cấu hình giới hạn RPS trên giao diện TableRegistry.

## 2. Các bước thực thi (Execution Steps)

### Bước 1: Backend Go (`cdc-cms-service`)
- [ ] Mở `internal/app/queries/source/source_objects_read_models.go`, thêm field `SnapshotMaxRPS *int` json tag `snapshot_max_rps,omitempty`.
- [ ] Mở `internal/infra/persistence/source/source_object_read_repo_gorm.go`, thêm `so.snapshot_max_rps,` vào danh sách cột SELECT của `ListSourceObjects`.
- [ ] Mở `internal/app/commands/source/update_source_object_v2.go`, thêm `SnapshotMaxRPS *int` vào struct `UpdateSourceObjectV2Command`, bổ sung validation và gán vào `updates["snapshot_max_rps"]`.
- [ ] Mở `internal/api/source/source_object_actions_handler.go`, thêm `SnapshotMaxRPS *int` vào struct body parser và gán sang Command.
- [ ] Chạy `go build ./...` hoặc `go test ./...` để verify compile.

### Bước 2: Frontend React (`cdc-cms-web`)
- [ ] Mở `src/types/index.ts`, thêm `snapshot_max_rps?: number | null;` vào `SourceObjectRow`.
- [ ] Mở `src/pages/TableRegistry.tsx`:
  - Thêm `'snapshot_max_rps'` vào `V2_EXCLUSIVE_FIELDS`.
  - Cập nhật `openEdit` set value cho `snapshot_max_rps`.
  - Cập nhật `handleEdit` xử lý trường hợp clear value (gửi 0 hoặc delete).
  - Thêm `<Form.Item name="snapshot_max_rps" label="Snapshot Max RPS (snapshot.v2)">` vào Modal "Chỉnh sửa Source Object".
- [ ] Chạy `npm run build` trong `cdc-cms-web` để verify TypeScript và Bundler.
