# 🔍 BÁO CÁO AUDIT PHẢN BIỆN — SỰ CỐ TAB PROGRESS RECON KHÔNG HIỂN THỊ TIẾN TRÌNH

---

## I. BỐI CẢNH & HIỆN TƯỢNG (TRIGGER)
- **Hiện tượng:** Khi người dùng bấm Recon Check cho bảng `hyperverge_audit_logs` (nhận được `Job ID: f82bf57f…`, `Trace ID: 09f39695…`), thông tin tiến trình chạy ngầm **không xuất hiện trong tab "Tiến trình đối soát thủ công" (Progress Tab)** trên giao diện CMS.
- **Tác động:** Người dùng không theo dõi được trạng thái `PENDING` -> `RUNNING` -> `COMPLETED` của các job recon chạy ngầm.

---

## II. ĐỐI SOÁT & PHÂN TÍCH GỐC RỄ (ROOT CAUSE ANALYSIS)

### 1. Phân tích Contract & Data Integrity:
- **Tầng Ghi (Worker Engine & DB):**
  - Khi `recon_check_handler.go` nhận NATS command từ CMS, `table` được chuẩn hóa thành FQN: `"shadow_gpay_ekyc.hyperverge_audit_logs"`.
  - Job được ghi vào PostgreSQL `cdc_system.recon_jobs` với:
    `target_table = 'shadow_gpay_ekyc.hyperverge_audit_logs'`.
- **Tầng Đọc (CMS Frontend & Backend Query):**
  - Frontend `ReconPipelineGrid.tsx` lấy `historyTable = "hyperverge_audit_logs"` (bảng trần) và gọi hook `useActiveReconJobs(historyTable)`.
  - HTTP Request gửi đi: `GET /api/reconciliation/jobs/active?target_table=hyperverge_audit_logs`.
  - Backend `recon_read_repo_gorm.go` thực thi:
    `SELECT * FROM cdc_system.recon_jobs WHERE status IN ('PENDING', 'RUNNING') AND target_table = 'hyperverge_audit_logs'`.
- **Gốc rễ:** Có sự không đồng nhất (mismatch) giữa định dạng lưu trữ (`qualified schema.table`) và điều kiện lọc truy vấn (`bare table`). So sánh chuỗi chính xác `=` thất bại hoàn toàn.

---

## III. CHI TIẾT CÁC FILE ĐÃ SỬA & ĐỐI SOÁT TỪNG DÒNG (LINE-BY-LINE AUDIT)

### 1. `cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go` (Dòng 584-590)
```diff
 	if targetTable != "" {
-		q = q.Where("target_table = ?", targetTable)
+		if strings.Contains(targetTable, ".") {
+			q = q.Where("target_table = ?", targetTable)
+		} else {
+			q = q.Where("target_table = ? OR target_table LIKE ?", targetTable, "%."+targetTable)
+		}
 	}
```
- **Phản biện & Đánh giá an toàn:**
  - Nếu query truyền FQN (`shadow_gpay_ekyc.hyperverge_audit_logs`) -> Query khớp chính xác 100%.
  - Nếu query truyền bảng trần (`hyperverge_audit_logs`) -> Hỗ trợ match cả bảng legacy lẫn bảng có tiền tố schema (`%.hyperverge_audit_logs`), không làm rớt dữ liệu của client cũ.

### 2. `cdc-cms-service/internal/api/recon/reconciliation_handler_reports.go` (Dòng 130-136)
```diff
 func (h *ReconciliationHandler) GetActiveJobs(c *fiber.Ctx) error {
 	targetTable := strings.TrimSpace(c.Query("target_table"))
+	shadowSchema := strings.TrimSpace(c.Query("shadow_schema"))
+	if shadowSchema != "" && targetTable != "" && !strings.Contains(targetTable, ".") {
+		targetTable = shadowSchema + "." + targetTable
+	}
 	jobs, err := h.reader.GetActiveReconJobs(c.UserContext(), targetTable)
```
- **Phản biện & Đánh giá an toàn:**
  - Nhận thêm `shadow_schema` từ query param để chủ động dựng FQN trước khi query database.

### 3. `cdc-cms-web/src/hooks/useReconStatus.ts` (Dòng 513-525)
```diff
-export function useActiveReconJobs(table: string | null) {
+export function useActiveReconJobs(table: string | null, shadowSchema?: string | null) {
   return useQuery<ActiveReconJobsResponse>({
-    queryKey: ['active-recon-jobs', table],
+    queryKey: ['active-recon-jobs', table, shadowSchema],
     queryFn: async () => {
       const params: Record<string, string> = {};
       if (table) params.target_table = table;
+      if (shadowSchema) params.shadow_schema = shadowSchema;
```
- **Phản biện & Đánh giá an toàn:**
  - Thêm `shadowSchema` vào queryKey của React Query để tránh cache stale giữa các schema khác nhau có cùng tên bảng.

### 4. `cdc-cms-web/src/components/ReconPipelineGrid.tsx` (Dòng 277)
```diff
-  const { data: activeJobs, isLoading: isLoadingJobs } = useActiveReconJobs(historyTable);
+  const { data: activeJobs, isLoading: isLoadingJobs } = useActiveReconJobs(historyTable, historySchema);
```
- **Phản biện & Đánh giá an toàn:**
  - Truyền đầy đủ `historySchema` từ state pipeline hiện tại vào hook.

---

## IV. TIẾN TRÌNH QC BIÊN DỊCH TOÀN BỘ HỆ THỐNG (FULL-STACK VERIFICATION)

Đã chạy kiểm tra đồng thời cả 3 thành phần:
1. `cdc-cms-service`: `go build -o /dev/null ./cmd/server` -> **CMS_EXIT_CODE=0 (PASS)**
2. `centralized-data-service`: `go build -o /dev/null ./cmd/worker` -> **WORKER_EXIT_CODE=0 (PASS)**
3. `cdc-cms-web`: `npm run build` (TypeScript check + Vite bundle) -> **WEB_EXIT_CODE=0 (PASS)**

---

## V. VÒNG LẶP PHẢN TỈNH (SELF-IMPROVEMENT LOOP)
- **Bài học rút ra:** Khi nâng cấp định danh FQN (`schema.table`) ở tầng Database & Ingestion, BẮT BUỘC rà soát toàn bộ các API Read/Query (Polling hooks, Dashboard queries) để đảm bảo câu lệnh SQL WHERE không bị lệch format so với dữ liệu mới được insert.

