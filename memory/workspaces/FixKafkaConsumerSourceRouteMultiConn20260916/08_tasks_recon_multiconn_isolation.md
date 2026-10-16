# 08_tasks_recon_multiconn_isolation.md
## Danh sách Nhiệm vụ Triển khai (Checklist)

- [x] **Task 1: Sửa lỗi Boolean Logic & Placeholders trong `GetTableHistory` (CMS Service)**
  - File: `cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go`
  - Tách bạch nhánh điều kiện WHERE cho Segment A (`source_shadow`) và Segment B (`shadow_master`).
  - Khắc phục lỗi `AND master_table = ?` làm biến mất 100% record Segment A.
  - Sửa mảng `args` khớp chính xác với 4 placeholders.

- [x] **Task 2: Gỡ bỏ điều kiện Master Binding trong `validatePipelineConnections` của Segment A Smoke Check (CDS Worker)**
  - File: `centralized-data-service/internal/service/recon/recon_engine_run.go`
  - Cho phép chặng Ingest Source $\rightarrow$ Shadow chạy Smoke Check độc lập mà không bắt buộc phải có approved master binding.

- [x] **Task 3: Chuẩn hóa Qualified Target Table cho `recon_jobs` (CDS Worker Handler)**
  - File: `centralized-data-service/internal/handler/recon/recon_check_handler.go`
  - Ghi nhận `TargetTable` vào `cdc_system.recon_jobs` dạng `<shadow_schema>.<shadow_table>` nếu có `ShadowSchema`.

- [x] **Task 4: Chống duplicate dòng trong `ListLatest` và tối ưu phân lập Multi-Connector (CMS Service)**
  - File: `cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go`
  - Dùng `LEFT JOIN LATERAL` cho `cdc_table_registry` trong `listLatestPrimary`.
  - Chuẩn hóa `DISTINCT ON` trong `listLatestLegacy` bao gồm `shadow_schema` và `master_schema`.

- [x] **Task 5: Phân lập Multi-Connector trên UI (CMS Web)**
  - File: `cdc-cms-web/src/components/ReconPipelineGrid.tsx`
  - Sửa `uniqueMap` trong `buildPipelines` có chứa `source_connection_code` và `shadow_schema`.
  - File: `cdc-cms-web/src/pages/DataIntegrity.tsx`
  - Cập nhật helper `reportByTarget` không dùng bare name đè nhau.

- [x] **Task 6: Nghiệm thu Biên dịch & Báo cáo**
  - Chạy `go build ./cmd/server` trong `cdc-cms-service`.
  - Chạy `go build ./cmd/worker` trong `centralized-data-service`.
  - Chạy `npm run build` trong `cdc-cms-web`.
  - Cập nhật `05_progress.md` append-only.
