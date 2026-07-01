# Workspace Context: Data Integrity Drift Mismatch (Ok status for mismatch count)

## Problem Description
- UI `/data-integrity` (tab tổng quan) hiển thị trạng thái "Khớp" (status = "ok") mặc dù có sự chênh lệch (drift) nhỏ giữa Source (39,988) và Shadow (39,987) (chênh lệch 1 bản ghi).
- Trạng thái drift_pct hiển thị là `0.00%` do tỷ lệ lệch cực kỳ nhỏ (1/39988 = 0.000025) và driftPct được tính bằng phần trăm.
- Trong logic backend, hàm `ComputeDriftStatus` (tại `cdc-cms-service/internal/app/queries/recon/recon_enrichment.go`) tính toán drift status dựa trên tỷ lệ phần trăm chênh lệch (`driftPct`). Nếu tỷ lệ này < 0.5%, nó trả về `"ok"`.
- Yêu cầu: Không báo "Khớp" khi có lệch (src != destCount). Trạng thái nên là "warning" (hoặc trạng thái phù hợp khác) thay vì "ok" để operator nhận biết được sự chênh lệch này.

## User Requirements
- Tìm và sửa logic tính trạng thái chênh lệch dữ liệu (drift status).
- Bất kỳ khi nào có chênh lệch (Source count != Dest count), không được báo là "Khớp" (status = "ok").
- Đảm bảo các unit test liên quan chạy đúng và phản ánh đúng logic mới.

## Definition of Done (DoD)
- logic tính toán status tại `ComputeDriftStatus` được cập nhật: nếu có chênh lệch (`src != destCount`), không trả về trạng thái `"ok"`.
- Test suite của `recon_enrichment_test.go` được cập nhật và pass 100%.
- Toàn bộ service `cdc-cms-service` build thành công.
- UI Data Integrity phản ánh đúng cảnh báo chênh lệch dữ liệu.
