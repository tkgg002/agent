# 11_report_recon_multiconn_isolation.md
## Báo Cáo Thay Đổi Mã Nguồn: Khắc Phục Lỗi Nhật Ký Đối Soát & Phân Lập Đa Kết Nối trans_his

---

### 1. Danh Sách Tệp Tin Thay Đổi & Số Lượng Dòng

| STT | Đường Dẫn Tệp Tin | Loại Thay Đổi | Số Dòng Thêm / Sửa | Mục Đích |
| :---: | :--- | :---: | :---: | :--- |
| 1 | `cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go` | Modify | ~25 dòng | Tách biệt WHERE branch cho Segment A và B trong `GetTableHistory`, sửa placeholders; dùng `LEFT JOIN LATERAL` trong `listLatestPrimary`; chuẩn hóa `DISTINCT ON` trong `listLatestLegacy`. |
| 2 | `centralized-data-service/internal/service/recon/recon_engine_run.go` | Modify | ~10 dòng | Gỡ bỏ ràng buộc `master_binding` trong `validatePipelineConnections` để chặng Ingest Source $\rightarrow$ Shadow chạy Smoke Check độc lập. |
| 3 | `centralized-data-service/internal/handler/recon/recon_check_handler.go` | Modify | ~12 dòng | Lưu Qualified Target Table (`<shadow_schema>.<shadow_table>`) vào `cdc_system.recon_jobs` khi trigger manual recon. |
| 4 | `cdc-cms-web/src/components/ReconPipelineGrid.tsx` | Modify | ~8 dòng | Thêm `source_connection_code` và `shadow_schema` vào khóa `uniqueMap` trong `buildPipelines` để phân lập 2 pipelines `trans_his`. |
| 5 | `cdc-cms-web/src/pages/DataIntegrity.tsx` | Modify | ~8 dòng | Khớp bản ghi modal action theo cả `target_table` và `shadow_schema`. |

---

### 2. Chi Tiết Thay Đổi (Overview & Diffs)

#### Tệp 1: `cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go`
- **Vấn đề:** Mệnh đề WHERE nối chuỗi `((shadow_schema = ? AND shadow_table = ?) OR (segment = 'shadow_master' AND master_table = ?)) AND master_table = ?` làm điều kiện `master_table` áp lên cả Segment A (`source_shadow`). Vì Segment A có `master_table = NULL`, 100% record Segment A bị loại bỏ khỏi tab Auto recon và Manual recon. Ngoài ra, `args` mâu thuẫn (`args[2]` nhận `table` thay vì `masterTable`).
- **Thay đổi:** Tách riêng 2 nhánh:
  + Segment A: `shadow_schema = ? AND shadow_table = ? AND (segment IS NULL OR segment != 'shadow_master')`.
  + Segment B: `segment = 'shadow_master' AND shadow_schema = ? AND master_table = ?`.
  + Mảng `args`: `[shadowSchema, table, shadowSchema, masterTable]`.
  + `listLatestPrimary`: Dùng `LEFT JOIN LATERAL (SELECT * FROM cdc_system.cdc_table_registry WHERE target_table = s.shadow_table AND ... ORDER BY id DESC LIMIT 1) reg ON TRUE`.
  + `listLatestLegacy`: Dùng `DISTINCT ON (COALESCE(shadow_schema, ''), shadow_table, COALESCE(NULLIF(master_schema, ''), ''), COALESCE(NULLIF(master_table, ''), ''), COALESCE(segment, 'source_shadow'))`.

#### Tệp 2: `centralized-data-service/internal/service/recon/recon_engine_run.go`
- **Vấn đề:** `validatePipelineConnections` JOIN `master_binding mb` với `schema_status = 'approved'` và yêu cầu `MasterConnID > 0`. Bất kỳ pipeline nào chưa có master binding approved đều bị skip hoàn toàn Segment A Smoke Check.
- **Thay đổi:** Gỡ bỏ `master_binding`, chỉ kiểm tra `source_conn_id > 0` và `shadow_conn_id > 0`.

#### Tệp 3: `centralized-data-service/internal/handler/recon/recon_check_handler.go`
- **Vấn đề:** Lưu tên bảng trần vào `cdc_system.recon_jobs`, khiến CMS `GetActiveReconJobs` lọc theo `shadow_schema.trans_his` không tìm thấy.
- **Thay đổi:** Chuẩn hóa `targetTable := payload.ShadowSchema + "." + payload.Table` trước khi insert vào `recon_jobs`.

#### Tệp 4: `cdc-cms-web/src/components/ReconPipelineGrid.tsx`
- **Vấn đề:** `uniqueMap` trong `buildPipelines` không có `source_connection_code`, khiến các row của 2 connector cùng bảng `trans_his` bị đè lên nhau.
- **Thay đổi:** Key của `uniqueMap` bổ sung `${srcConn}::${schema}::${table}...`.

#### Tệp 5: `cdc-cms-web/src/pages/DataIntegrity.tsx`
- **Vấn đề:** `reportByTarget.get(action.table)` dùng map key theo tên trần `trans_his`.
- **Thay đổi:** Tìm kiếm đối soát cả `target_table` VÀ `shadow_schema`.

---

### 3. Kết Quả Nghiệm Thu Thực Tế (DoD G3 Verification)
1. `cdc-cms-service`: `go build ./cmd/server` $\rightarrow$ Exit Code = 0 (Thành công 100%).
2. `centralized-data-service`: `go build ./cmd/worker` $\rightarrow$ Exit Code = 0 (Thành công 100%).
3. `cdc-cms-web`: `npm run build` $\rightarrow$ Exit Code = 0 (Thành công 100%).
