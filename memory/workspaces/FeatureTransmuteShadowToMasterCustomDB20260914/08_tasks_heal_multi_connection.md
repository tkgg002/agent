# DANH SÁCH NHIỆM VỤ: KHẮC PHỤC TRIỆT ĐỂ LỆCH PIPELINE BÁO CÁO RECON/HEAL & CHUẨN HÓA CENTRALIZED RESOLVERS
**Workspace:** `FeatureTransmuteShadowToMasterCustomDB20260914`  
**Mã đợt việc:** `TASKS-HEAL-PIPELINE-ISOLATION-V1`

---

## Phase 1: Backend CMS Service (`cdc-cms-service`) — Centralized Query Resolvers
- [ ] **Task 1.1: Tạo Helper Function xài chung `BuildReconReportWhere`**
  * File: `internal/infra/persistence/recon/recon_read_repo_gorm.go`
  * Chuẩn hóa logic WHERE cho `cdc_reconciliation_report`:
    - Nếu có `masterTable`:
      * Với `segment = 'shadow_master'`: `shadow_schema = ? AND (shadow_table = ? OR shadow_table = ?) AND master_table = ?`
      * Lọc chính xác theo cặp `(shadow_table, master_table)` thay vì ép `shadow_table = table`.
    - Phân biệt rõ ràng khi `table` là Master Table vs Shadow Table.
- [ ] **Task 1.2: Cập nhật `GetTableHistory` & `ListUnhealedReports`**
  * Áp dụng helper dùng chung `BuildReconReportWhere`.
  * Hỗ trợ nhận thêm tham số `shadow_table`, `master_table` từ query string để triệt tiêu 100% việc lẫn lộn giữa các pipeline cùng source.
- [ ] **Task 1.3: Cập nhật DTO & Handlers API**
  * `internal/api/recon/reconciliation_handler_reports.go`: Parse `shadow_table`, `master_table`, `master_schema` truyền vào query.
  * `internal/api/recon/reconciliation_handler_execute_heal.go`: `GetUnhealedReports` nhận và truyền `master_table`, `shadow_table`.

---

## Phase 2: Centralized Data Service (`centralized-data-service`) — Heal Multi-Connection & Prune Safety
- [ ] **Task 2.1: Bổ sung `master_binding_id` vào `publishTransmuteChunked`**
  * File: `internal/handler/recon/recon_execute_heal_handler.go`
  * Khi heal Segment B, resolve hoặc lấy `master_binding_id` từ `rpt` và đưa vào payload NATS `cdc.cmd.transmute`:
    ```json
    {
      "master_table": rpt.MasterTable,
      "master_binding_id": rpt.MasterBindingID,
      "_source_ids": sourceIDs,
      "triggered_by": "execute-heal-b"
    }
    ```
- [ ] **Task 2.2: Xóa bỏ hardcode `h.reconCore.MasterPlane()` trong Prune Orphan Master**
  * File: `internal/handler/recon/recon_execute_heal_handler.go` (dòng 434)
  * Thay thế bằng Dynamic Master DB qua `ConnectionManager.GetMasterDB(ctx, masterConnectionKey)`.
  * Tuyệt đối không xóa nhầm database `default_master` khi đang chạy prune cho `master_2`.

---

## Phase 3: CMS Web Frontend (`cdc-cms-web`) — UI Centralized Helpers & Modal Isolation
- [ ] **Task 3.1: Xây dựng Helper Function xài chung `getPipelineIdentity` & `getReportIdentity`**
  * File: `src/utils/pipelineIdentity.ts` (hoặc trong `src/components/ReconPipelineGrid.tsx`)
  * Trả về định danh đầy đủ: `{ shadowSchema, shadowTable, masterSchema, masterTable, masterBindingId }`.
- [ ] **Task 3.2: Mở rộng `ExecuteHealModalProps` & hooks**
  * File: `src/components/ExecuteHealModal.tsx`
  * Nhận các props rõ ràng: `shadowSchema`, `shadowTable`, `masterSchema`, `masterTable`.
  * Truyền đủ `masterTable` và `shadowTable` vào `useTableHistory` và `useUnhealedReports`.
- [ ] **Task 3.3: Cập nhật gọi `openHeal` trong `DataIntegrity.tsx`**
  * Khi bấm Heal từ `ReconPipelineGrid`, truyền đủ định danh cả Shadow lẫn Master.
  * Đảm bảo tab "Phiên đã xử lý" hiển thị đúng 100% phiên heal của chính pipeline đó.

---

## Phase 4: Kiểm Thử Nghiệm Thu & Live Verification (Gate DoD)
- [ ] **Task 4.1:** Verify `go build ./cmd/server` (`cdc-cms-service`).
- [ ] **Task 4.2:** Verify `go build ./cmd/worker` (`centralized-data-service`).
- [ ] **Task 4.3:** Verify `npm run build` (`cdc-cms-web`).
- [ ] **Task 4.4:** Kiểm tra live:
  * Mở Modal Heal trên dòng `export_jobs_2`: Tab "Phiên đã xử lý" PHẢI hiển thị phiên ID 230 vừa heal.
  * Mở Modal Heal trên dòng `export_jobs`: Tab "Phiên đã xử lý" KHÔNG ĐƯỢC hiển thị phiên của `export_jobs_2`.
