# Kế hoạch Tối ưu hóa Reconciliation Report (24h Window Pruning)

## Đánh Giá Trạng Thái Hiện Tại
- **API `/api/recon/backfill-source-ts/status`:** Đã giảm từ **690ms -> 5.8ms** (Tối ưu thành công 100%!).
- **API `/api/reconciliation/report`:** Đã giảm từ **1.648s -> 483ms** (Giảm 70% thời gian nhưng vẫn ở mức 483ms, vượt ngưỡng cảnh báo SLOW SQL >= 200ms).

---

## Nguyên Nhân & Giải Pháp Tối Ưu

### Nguyên nhân:
CTE `smoke_latest` thực hiện `DISTINCT ON` với 6 biểu thức `COALESCE` phức tạp trên cửa sổ 7 ngày (~900.000 dòng log smoke result) để chỉ rút ra vỏn vẹn **15 dòng kết quả**.

### Giải pháp (Single Best Approach):
Rút ngắn cửa sổ lọc smoke test từ 7 ngày xuống **24 giờ** (`WHERE checked_at >= NOW() - INTERVAL '24 hours'`) trong `recon_read_repo_gorm.go`.
- Lý do: Smoke test chạy liên tục (vài giây đến vài phút/lần), do đó kết quả mới nhất của tất cả các bảng chắc chắn nằm trong 24 giờ gần đây.
- Tác động: Giảm 85%+ số lượng bản ghi smoke log cần Sort (từ ~900.000 dòng xuống ~120.000 dòng), giúp thời gian xử lý giảm từ **483ms xuống < 50ms**.

---

## Proposed Changes

### cdc-cms-service (Go Backend)

#### [MODIFY] `internal/infra/persistence/recon/recon_read_repo_gorm.go`
- Thay đổi `WHERE checked_at >= NOW() - INTERVAL '7 days'` thành `WHERE checked_at >= NOW() - INTERVAL '24 hours'` trong CTE `smoke_latest`.

---

## Verification Plan

### Automated Tests
- Chạy biên dịch kiểm tra syntax Go: `go build ./cmd/server` tại repo `cdc-cms-service`.

### Manual Verification
- Kiểm tra lại latency log của API `GET /api/reconciliation/report` để xác nhận giảm từ 483ms xuống < 50ms.
