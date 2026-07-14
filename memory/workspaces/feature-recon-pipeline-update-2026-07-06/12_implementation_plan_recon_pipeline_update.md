# Kế hoạch triển khai chi tiết - Cập nhật Reconciliation UI & API Pipeline

Kế hoạch này chi tiết hóa cách thức cập nhật luồng API đối soát và giao diện UI nhằm tự động đề xuất khoảng thời gian 30 ngày cho các chế độ đối soát sâu (Full Search/Deep Check) và hiển thị chính xác nút "Chữa lành" khi phát hiện lệch counts (cho dù lookback window báo OK).

## Các bước triển khai đã thực hiện

### 1. Cập nhật Backend (`cdc-cms-service`)
- **Định nghĩa Schema:** Thêm trường `HealNeeded bool` vào model DTO `LatestReportRow` trong `internal/app/queries/recon/recon_read_models.go`.
- **Hàm tiện ích:** Thiết lập hàm `ComputeHealNeeded` trong `internal/app/queries/recon/recon_enrichment.go` để tính toán trạng thái "cần chữa lành" dựa trên:
  - Trạng thái đối soát của lookback window (Recon Check): `status` là `drift`, `dest_missing`, hoặc `warning`.
  - Mismatch số lượng bản ghi của Smoke Check (Source/Shadow hoặc Shadow/Master tùy theo segment của row).
- **Enrichment Logic:** Tích hợp hàm `ComputeHealNeeded` vào handler `/api/reconciliation/report` tại `internal/api/recon/reconciliation_handler_reports.go`.

### 2. Cập nhật Frontend (`cdc-cms-web`)
- **Kiểu dữ liệu:** Bổ sung trường optional `heal_needed?: boolean` vào interface `ReconRow` tại `src/hooks/useReconStatus.ts`.
- **Tự động điền Date Range:** Thêm hàm `handleCheckModeChange` vào modal `ConfirmDestructiveModal.tsx` để tự động gán khoảng thời gian là 30 ngày gần nhất khi người dùng chuyển sang chế độ `full_diff` hoặc `deep`.
- **Logic hiển thị nút Chữa lành:**
  - Cập nhật bảng báo cáo ở `src/pages/DataIntegrity.tsx` để hiển thị nút "Chữa lành" nếu `record.heal_needed === true` (giữ fallback kiểm tra `status` cũ).
  - Cập nhật disable state của nút "Chữa lành" ở `src/components/ReconPipelineGrid.tsx` cho cả hai chặng (Row A và Row B).

### 3. Sửa lỗi biên dịch Frontend
- Loại bỏ các import không sử dụng và các biến/hàm khai báo thừa nhưng không sử dụng để đáp ứng tiêu chuẩn nghiêm ngặt của `tsc -b`.
