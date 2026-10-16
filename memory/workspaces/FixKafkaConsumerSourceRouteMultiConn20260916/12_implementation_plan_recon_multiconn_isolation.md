# 12_implementation_plan_recon_multiconn_isolation.md
## Kế hoạch Triển khai Kỹ thuật: Khắc phục Lỗi Hiển thị Nhật ký Đối soát & Cô lập Đa Kết nối cho trans_his

---

### Phase 1: Backend CMS Service (`cdc-cms-service`)
1. **Sửa `GetTableHistory` trong `internal/infra/persistence/recon/recon_read_repo_gorm.go`:**
   - Xây dựng lại mệnh đề WHERE phân tách rành mạch Segment A và Segment B:
     ```go
     var where string
     var args []interface{}
     if shadowSchema != "" {
         if masterTable != "" {
             where = "((shadow_schema = ? AND shadow_table = ? AND (segment IS NULL OR segment != 'shadow_master')) OR (segment = 'shadow_master' AND shadow_schema = ? AND master_table = ?))"
             args = []interface{}{shadowSchema, table, shadowSchema, masterTable}
         } else {
             where = "((shadow_schema = ? AND shadow_table = ?) OR (segment = 'shadow_master' AND shadow_schema = ?))"
             args = []interface{}{shadowSchema, table, shadowSchema}
         }
     } else {
         if masterTable != "" {
             where = "(shadow_table = ? OR master_table = ?)"
             args = []interface{}{table, masterTable}
         } else {
             where = "(shadow_table = ? OR master_table = ?)"
             args = []interface{}{table, table}
         }
     }
     ```
2. **Sửa `listLatestPrimary` & `listLatestLegacy` trong `recon_read_repo_gorm.go`:**
   - Sửa JOIN `cdc_table_registry reg` sang `LEFT JOIN LATERAL (...) reg ON TRUE` có `LIMIT 1`.
   - Chuẩn hóa `DISTINCT ON` trong `listLatestLegacy`.
3. **Kiểm tra biên dịch:** `go build ./cmd/server`.

---

### Phase 2: Centralized Data Service Worker Engine (`centralized-data-service`)
1. **Sửa `validatePipelineConnections` trong `internal/service/recon/recon_engine_run.go`:**
   - Bỏ JOIN `master_binding mb` và bỏ kiểm tra `cr_ms.id > 0`.
   - Chỉ kiểm tra `source_conn_id > 0` và `shadow_conn_id > 0` để Segment A Smoke Check chạy độc lập.
2. **Sửa `recon_check_handler.go`:**
   - Ghi `targetTable := payload.ShadowSchema + "." + payload.Table` vào `cdc_system.recon_jobs` để CMS `GetActiveReconJobs` tìm kiếm chính xác theo pipeline.
3. **Kiểm tra biên dịch:** `go build ./cmd/worker`.

---

### Phase 3: Frontend CMS Web (`cdc-cms-web`)
1. **Sửa `ReconPipelineGrid.tsx`:**
   - Trong `buildPipelines`: Bổ sung `source_connection_code` và `shadow_schema` vào `key` của `uniqueMap`.
2. **Sửa `DataIntegrity.tsx`:**
   - Khi tìm record cho action modal: Khớp cả `target_table` và `shadow_schema`.
3. **Kiểm tra biên dịch:** `npm run build`.

---

### Phase 4: Verification & Audit Log
1. Cập nhật `05_progress.md` append-only.
2. Cập nhật `08_tasks_recon_multiconn_isolation.md`.
3. Báo cáo nghiệm thu chi tiết từng file, từng dòng diffs cho User.
