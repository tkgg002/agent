# Kế hoạch Lọc Bỏ Smoke Check Trong Lịch Sử & Hiển Thị Đúng Phiên Đã Chữa Lành

Tài liệu này mô tả phương án bổ sung tham số lọc `exclude_smoke` từ phía database lên đến frontend API nhằm loại bỏ toàn bộ bản ghi Smoke Check (đối soát nhanh) khỏi tab "Phiên đã xử lý" (Healed Reports) của modal Chữa lành. Đồng thời, làm rõ và xác minh cơ chế cập nhật trạng thái tổng (status) của phiên chữa lành.

## Phân tích Hiện trạng & Xác minh Trạng thái (Status)
1. **Trôi dữ liệu:** Hệ thống chạy Smoke Check đối soát nhanh mỗi phút một lần. Kết quả của Smoke Check được lưu trong bảng `cdc_system.cdc_recon_smoke_result` và gộp chung vào API Lịch sử `/api/reconciliation/report/:table` thông qua mệnh đề `UNION ALL`.
2. **Không phân trang thông minh:** Vì API chỉ trả về tối đa 100 bản ghi gần nhất (chủ yếu là Smoke Check), các báo cáo đối soát thực tế chạy bằng cửa sổ thời gian (như ID 91 chạy ngày 11/07) bị đẩy ra khỏi danh sách 100 bản ghi này. Do đó, việc lọc trên client `r.check_type !== 'smoke'` dẫn đến mảng trống.
3. **Xác minh logic trạng thái tổng (Status):**
   * Trong suốt quá trình hệ thống đang thực hiện chữa lành (luồng chạy transmuter / sync), status của report được cập nhật là `healing`.
   * Khi kết thúc xử lý, backend tính toán:
     `isFullyHealed := (HealedMissingDestCount >= MissingCount) && (HealedMismatchedCount >= StaleCount) && (PrunedMissingSrcCount >= OrphanCount)`
   * Trạng thái tổng chỉ được cập nhật về `healed` (đồng thời set `healed_at = now`) **khi và chỉ khi** cả 3 loại lỗi được chữa lành hoàn toàn (thỏa mãn `isFullyHealed == true`).
   * Nếu chỉ chữa lành được một phần (ví dụ `HealedMissingDestCount < MissingCount`), trạng thái tổng sẽ cập nhật về `partially_healed` (không set `healed_at` để giữ lại trong tab Chưa xử lý).
   * **Kết luận:** Logic cập nhật trạng thái tổng hiện tại trên backend hoàn toàn khớp với nghiệp vụ người dùng yêu cầu.

---

## Proposed Changes

### 1. Backend (cdc-cms-service)

#### [MODIFY] [recon_reader.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/queries/recon/recon_reader.go)
- Cập nhật signature hàm `GetTableHistory` trong interface `ReconReader` để nhận thêm tham số `excludeSmoke bool`.

#### [MODIFY] [recon_read_repo_gorm.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go)
- Hàm `GetTableHistory` nhận thêm tham số `excludeSmoke bool`.
- Nếu `excludeSmoke` là `true`, không thực hiện `UNION ALL` với bảng `cdc_recon_smoke_result`, chỉ truy vấn trực tiếp từ bảng `cdc_reconciliation_report` (sử dụng `baseQuery` thay vì `unionQuery`).

#### [MODIFY] [get_table_history.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/queries/recon/get_table_history.go)
- Thêm trường `ExcludeSmoke bool` vào struct `GetTableHistoryQuery`.
- Hàm `Handle` truyền `q.ExcludeSmoke` vào repository call.

#### [MODIFY] [reconciliation_handler_reports.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/recon/reconciliation_handler_reports.go)
- Trong controller `TableHistory`, đọc tham số query `exclude_smoke == "true"` và gán vào query struct.

#### [MODIFY] [queries_test.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/test/internal/app/queries/queries_test.go)
- Cập nhật mock `stubReconReader.GetTableHistory` để khớp signature (nhận thêm param `excludeSmoke bool`).

---

### 2. Frontend (cdc-cms-web)

#### [MODIFY] [useReconStatus.ts](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/hooks/useReconStatus.ts)
- Cập nhật hook `useTableHistory` để nhận tham số `excludeSmoke = false` và truyền param `exclude_smoke: 'true'` lên API khi active.

#### [MODIFY] [ExecuteHealModal.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ExecuteHealModal.tsx)
- Truyền `true` vào tham số `excludeSmoke` của `useTableHistory` để API chỉ trả về các phiên đối soát thực tế.
- Bộ lọc hiển thị `healedReports` lọc bỏ các bản ghi `ok`, `error`, và chỉ giữ lại những bản ghi thỏa mãn `isReportFullyHealed(r)`.

---

## Verification Plan

### Automated Tests
1. **Biên dịch backend cdc-cms-service:**
   ```bash
   go build ./cmd/server/...
   ```
2. **Biên dịch frontend cdc-cms-web:**
   ```bash
   npx tsc --noEmit
   ```

### Manual Verification
1. Mở modal Chữa lành cho `export_jobs`.
2. Kiểm tra tab "Phiên đã xử lý". Xác nhận ID 91 được hiển thị chính xác ở vị trí đầu tiên cùng trạng thái đã chữa lành (healed), không còn bị trống và không bị nhiễu bởi bất kỳ dòng Smoke Check nào.
3. Kiểm tra tab "Phiên chưa xử lý". Xác nhận ID 91 đã được dọn sạch khỏi danh sách chưa xử lý.
