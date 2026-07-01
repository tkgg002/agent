# Implementation Plan - Fix Data Integrity Drift Status Mismatch
# Kế hoạch Thực hiện - Sửa lỗi lệch trạng thái đối soát dữ liệu (Khớp khi lệch)

## English Version

### Goal
Fix the bug where the data integrity dashboard shows "Khớp" (status = "ok") even when there is a small difference (drift) between the source and destination counts (e.g. 39,988 vs 39,987). Also enable "Heal" (Chữa lành) and "Prune orphan" actions on the UI when status is "warning".

### Root Cause
1. In `cdc-cms-service/internal/app/queries/recon/recon_enrichment.go`, `ComputeDriftStatus` calculates the percentage difference (`driftPct`). If `driftPct` is less than `0.5%`, it falls back to `status := "ok"`. This results in displaying "Khớp" (status = "ok") when the counts actually differ but by a tiny margin.
2. In frontend `DataIntegrity.tsx`, "Chữa lành" and "Prune orphan" buttons were hidden for "warning" status, preventing users from manual recovery/pruning for small drifts.

### Proposed Changes

#### Backend
- Modify `ComputeDriftStatus` so that:
  - If `src == destCount`, return `status = "ok"`.
  - Otherwise, if there is a difference (`src != destCount`):
    - If `driftPct < 0.5%`, set `status = "warning"` (instead of `"ok"`).

#### Frontend
- Modify `DataIntegrity.tsx` in `cdc-cms-web` to:
  - Show "Chữa lành" button if status is `'drift'`, `'dest_missing'`, or `'warning'`.
  - Show "Prune orphan" button if status is `'drift'` or `'warning'`.
  - Count `'warning'` and `'dest_missing'` tables as drifted in `driftCount` statistics.

#### Files to Modify
- `cdc-cms-service/internal/app/queries/recon/recon_enrichment.go`
- `cdc-cms-service/test/internal/app/queries/recon_enrichment_test.go`
- `cdc-cms-web/src/pages/DataIntegrity.tsx`

### Verification Plan
- Run unit tests:
  ```bash
  go test -v ./test/internal/app/queries/...
  ```
- Build the service:
  ```bash
  go build -o /dev/null ./cmd/cms-service/...
  ```
- Build the web app:
  ```bash
  npm run build
  ```

---

## Tiếng Việt Version

### Mục tiêu
Sửa lỗi khi trang tổng quan đối soát hiển thị trạng thái "Khớp" (status = "ok") mặc dù có sự lệch số lượng bản ghi nhỏ giữa source và destination (ví dụ: 39,988 vs 39,987). Đồng thời hiển thị nút "Chữa lành" và "Prune orphan" trên UI khi trạng thái là "warning" (cảnh báo) để người dùng có thể thực hiện đồng bộ lại hoặc prune bản ghi lệch.

### Nguyên nhân gốc rễ
1. Tại `cdc-cms-service/internal/app/queries/recon/recon_enrichment.go`, hàm `ComputeDriftStatus` tính toán phần trăm chênh lệch (`driftPct`). Nếu `driftPct` dưới `0.5%`, trạng thái mặc định rơi về `status := "ok"`. Điều này dẫn đến việc hiển thị "Khớp" mặc dù thực tế số lượng đếm khác nhau.
2. Trên frontend `DataIntegrity.tsx`, nút "Chữa lành" và "Prune orphan" bị ẩn với trạng thái `"warning"`, khiến người dùng không thể thực hiện dọn dẹp hoặc sync các bản ghi lệch nhỏ.

### Đề xuất Thay đổi

#### Backend
- Thay đổi logic hàm `ComputeDriftStatus`:
  - Nếu `src == destCount`, trả về `status = "ok"`.
  - Nếu có chênh lệch (`src != destCount`):
    - Nếu `driftPct < 0.5%`, gán `status = "warning"` thay vì `"ok"`.

#### Frontend
- Sửa đổi `DataIntegrity.tsx` trong `cdc-cms-web`:
  - Hiện nút **Chữa lành** khi `record.status` là `'drift'`, `'dest_missing'`, hoặc `'warning'`.
  - Hiện nút **Prune orphan** khi `record.status` là `'drift'` hoặc `'warning'`.
  - Thêm `'warning'` và `'dest_missing'` vào việc tính toán `driftCount` trong phần tổng hợp thống kê.

#### Các tệp cần sửa đổi
- `cdc-cms-service/internal/app/queries/recon/recon_enrichment.go`
- `cdc-cms-service/test/internal/app/queries/recon_enrichment_test.go`
- `cdc-cms-web/src/pages/DataIntegrity.tsx`

### Kế hoạch Xác minh
- Chạy unit test backend:
  ```bash
  go test -v ./test/internal/app/queries/...
  ```
- Biên dịch service:
  ```bash
  go build -o /dev/null ./cmd/cms-service/...
  ```
- Biên dịch web app:
  ```bash
  npm run build
  ```
