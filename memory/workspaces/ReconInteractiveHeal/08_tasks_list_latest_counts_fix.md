# Danh sách Task - Sửa lỗi hiển thị tổng record trong tab Pipeline

- [x] Cập nhật query `listLatestPrimary` trong `recon_read_repo_gorm.go` của Backend.
- [x] Cập nhật việc tính toán `sourceTotal`, `shadowActive`, `masterActive` trong `ReconPipelineGrid.tsx` của Frontend.
- [x] Biên dịch dự án Backend (`go build ./...` trong `cdc-cms-service`) và Frontend (`npx tsc --noEmit` trong `cdc-cms-web`).
- [x] Chạy script kiểm soát quy trình (`verify_governance.py`).
