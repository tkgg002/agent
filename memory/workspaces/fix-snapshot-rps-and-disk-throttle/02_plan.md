# 02_plan.md — Kế hoạch triển khai cao tầng

## 1. Mục tiêu
Hoàn thiện tính năng điều tiết tốc độ `snapshot_max_rps` trên giao diện CMS và Backend, giúp operator chủ động khống chế tốc độ snapshot, bảo vệ đĩa PostgreSQL khỏi hiện tượng bão hòa I/O (95-100%).

## 2. Kỹ năng sử dụng (Pre-flight Skill Declaration)
- `golang-patterns`: Triển khai chuẩn Domain/Port/Adapter cho `cdc-cms-service`.
- `react-patterns`: Triển khai Form/Modal/Validation trên Ant Design của `cdc-cms-web`.
- `database-design`: Quản trị schema và tối ưu I/O cho PostgreSQL.

## 3. Lộ trình thực hiện (Roadmap)
- **Phase 1: Backend CMS (`cdc-cms-service`)**
  - Cập nhật Read Model `SourceObjectListItem`.
  - Cập nhật SQL query trong `source_object_read_repo_gorm.go`.
  - Cập nhật Command, Handler, và DTO trong `update_source_object_v2.go` và `source_object_actions_handler.go`.
  - Build & Unit Test Backend.
- **Phase 2: Frontend CMS (`cdc-cms-web`)**
  - Cập nhật Type interface `SourceObjectRow` trong `types/index.ts`.
  - Cập nhật `TableRegistry.tsx` (V2_EXCLUSIVE_FIELDS, openEdit, handleEdit, Form.Item).
  - Build Frontend (`npm run build`).
- **Phase 3: Verification & Resume**
  - Verify toàn bộ chuỗi cập nhật qua API và UI.
  - Hướng dẫn cấu hình RPS và Resume snapshot cho `bank_requests`.
