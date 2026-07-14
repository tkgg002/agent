# Implementation Plan: Clean Up Redundant Columns and Standardize Source Metadata

Kế hoạch dọn dẹp các luồng Smoke Check dư thừa (`RunSmokeCheck`/`RunSmokeCheckB`) và loại bỏ các cột không còn sử dụng (`tier`, `target_table`) khỏi bảng `cdc_reconciliation_report`. Đồng thời, chuẩn hóa bảng này bằng cách thêm các cột định danh nguồn trực quan (`source_type`, `source_host`, `source_table`) để lưu vết đầy đủ thông tin kết nối và tích hợp hiển thị trực quan lên giao diện Frontend.

---

## Proposed Changes

### Component: cdc-cms-service (Database & Read-Side Model)

#### [NEW] [092_recon_cleanup_redundant.sql](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/migrations/schema/recon_dlq/092_recon_cleanup_redundant.sql)
- SQL migration thực hiện:
  - Drop cột `tier` và `target_table` khỏi bảng `cdc_system.cdc_reconciliation_report`.
  - Add thêm 3 cột mới: `source_type VARCHAR`, `source_host VARCHAR`, và `source_table VARCHAR` vào bảng `cdc_system.cdc_reconciliation_report`.

#### [MODIFY] [reconciliation_report.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/model/recon/reconciliation_report.go)
- Xóa bỏ trường `Tier` khỏi struct `ReconciliationReport`.
- Giữ lại trường `TargetTable` nhưng xóa tag mapping database (hoặc đổi thành `gorm:"-"`) để phục vụ tương thích ngược với dữ liệu động tính toán từ các query.
- Bổ sung 3 trường mới map database:
  - `SourceType string gorm:"column:source_type" json:"source_type,omitempty"`
  - `SourceHost string gorm:"column:source_host" json:"source_host,omitempty"`
  - `SourceTable string gorm:"column:source_table" json:"source_table,omitempty"`

#### [MODIFY] [source_object_read_repo_gorm.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/source/source_object_read_repo_gorm.go)
- Cập nhật các câu lệnh SQL JOIN đến bảng `cdc_reconciliation_report` thay vì join bằng `target_table` cũ thì join bằng `shadow_table` (vì chặng A `shadow_table` chính là target table chặng A):
  - Dòng 84: `WHERE rr.shadow_table = COALESCE(sb.shadow_table, tr.target_table)`
  - Dòng 268: `WHERE rr.shadow_table = tr.target_table`
  - Dòng 404: `WHERE rr.shadow_table = sb.shadow_table`
- Đồng thời cập nhật SELECT list trong các subquery LATERAL để trả về `rr.shadow_table AS target_table` thay vì truy vấn trực tiếp cột vật lý `rr.target_table` đã bị drop:
  - Dòng 79: Đổi `rr.target_table,` thành `rr.shadow_table AS target_table,`
  - Dòng 264: Đổi `rr.target_table,` thành `rr.shadow_table AS target_table,`


#### [MODIFY] [recon_read_repo_gorm.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go)
- Cập nhật câu lệnh `UNION ALL` trong `GetTableHistory`:
  - Loại bỏ trường `tier` (ở cả 2 phần SELECT).
  - Phần SELECT từ `cdc_reconciliation_report`: chọn trực tiếp các trường `source_type`, `source_host`, `source_table`, `recon_start_time`, `recon_end_time`.
  - Phần SELECT từ `cdc_recon_smoke_result`: chọn tương tự các trường trên trực tiếp từ bảng smoke (riêng start/end time chọn `NULL::timestamp without time zone`).
- Cập nhật hàm `listLatestPrimary` (dòng 84-102): thay đổi `NULL::text AS source_type/source_host/source_table` thành select trực tiếp các trường `r.source_type`, `r.source_host`, `r.source_table` từ bảng `cdc_reconciliation_report`.

---

### Component: centralized-data-service (Recon Engine & Backend Handlers)

#### [MODIFY] [reconciliation_report.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/model/recon/reconciliation_report.go)
- Xóa bỏ trường `Tier` khỏi struct `ReconciliationReport`.
- Thay đổi tag GORM của `TargetTable` thành `gorm:"-"` để GORM bỏ qua cột này khi ghi dữ liệu (do cột vật lý này đã bị drop khỏi DB).
- Bổ sung 3 trường mới map database:
  - `SourceType string gorm:"column:source_type" json:"source_type"`
  - `SourceHost string gorm:"column:source_host" json:"source_host"`
  - `SourceTable string gorm:"column:source_table" json:"source_table"`

#### [MODIFY] [reconciliation_report_repo.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/repository/recon/reconciliation_report_repo.go)
- Thay thế các truy vấn theo cột vật lý `target_table` cũ bằng `shadow_table` hoặc `master_table` tương ứng theo segment, hoặc kiểm tra cả hai nếu không có segment.
- Ánh xạ tham số `tier` thành tập hợp `check_type` tương ứng (do cột `tier` đã bị drop).
- Cụ thể:
  - `GetLatestByTable`: Check `master_table` nếu segment là `shadow_master`, check `shadow_table` nếu là `source_shadow`, ngược lại check cả hai.
  - `GetLatestMissingReport` & `GetLatestMissingReportWithSegment`: Check `shadow_table`/`master_table` và đổi lọc `tier = ?` thành `check_type IN ?` (trong đó Tier 1 mapping thành `smoke`/`count_windowed`, Tier 2 thành `hash_window`, Tier 3 thành `bucket_hash`/`deep_check`).
  - `GetUnhealedReports`: Sửa sang check `shadow_table = ? OR master_table = ?`.


#### [MODIFY] [recon_engine_segment_b.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_engine_segment_b.go)
- Trong hàm `stampA` (Segment A), tự động bóc tách và gán thông tin nguồn đầy đủ trước khi lưu DB:
  - `report.SourceType = entry.SourceType`
  - `report.SourceHost = extractHost(entry.SourceURL)`
  - `report.SourceTable = entry.SourceTable`
- Trong hàm `stampB` (Segment B), gán thông tin shadow plane nguồn trước khi lưu DB:
  - `report.SourceType = "postgresql"`
  - `report.SourceHost = "shadow_plane"`
  - `report.SourceTable = ref.ShadowTable`

#### [MODIFY] [recon_tier_a.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_tier_a.go)
- Xóa bỏ hoàn toàn hàm dead code `RunSmokeCheck`.

#### [MODIFY] [recon_tier_b.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_tier_b.go)
- Xóa bỏ hoàn toàn hàm dead code `RunSmokeCheckB`.

#### [MODIFY] [recon_check_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_check_handler.go)
- Trong `validateAndEnrichContext`, loại bỏ `TypeReconSmoke` khỏi switch-case để API tự động chặn và báo lỗi `invalid_type_recon: unsupported check type 'smoke'` nếu nhận được request loại này.
- Trong `executeGenericCheck` và các hàm wrapper liên quan, loại bỏ logic chuyển hướng gọi `RunSmokeCheck`/`RunSmokeCheckB`.

---

### Component: cdc-cms-web (Frontend UI)

#### [MODIFY] [useReconStatus.ts](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/hooks/useReconStatus.ts)
- Cập nhật interface `ReconReport`:
  - Xóa bỏ trường `tier`.
  - Bổ sung `source_host?: string | null` và `source_table?: string | null`.

#### [MODIFY] [ReconPipelineGrid.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ReconPipelineGrid.tsx)
- Cập nhật hàm helper `levelLabel` để map "Loại scan" hiển thị trên UI bằng thuộc tính `check_type` thay vì `tier`:
  - `smoke` / `count_total` ➔ `Smoke Check (Tier 1)`
  - `hash_window` / `segment_b_window` ➔ `Hash Window (Tier 2)`
  - `deep_check` ➔ `Deep Check (Tier 3)`
  - `orphan_prune` ➔ `Orphan Prune`
- Cập nhật cách hiển thị **Bảng nguồn** (Source Name) tại dòng 191 và 156: hiển thị đầy đủ thông tin nguồn trực quan nếu có `source_host` và `source_table` theo format: `[source_type] source_host / source_db . source_table`.

---

## Verification Plan

### Automated Tests
1. Chạy các unit tests ở backend:
   ```bash
   # Centralized Data Service
   go test -count=1 ./internal/service/recon/...
   go test -count=1 ./internal/handler/recon/...
   
   # CMS Service
   go test ./test/...
   ```
2. Build kiểm tra Frontend:
   ```bash
   npm run build
   ```
