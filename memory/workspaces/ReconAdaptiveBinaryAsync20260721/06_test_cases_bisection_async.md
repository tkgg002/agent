# 06 — Kế Hoạch Kiểm Thử & Test Cases: Adaptive Binary & Async Job

> **Workspace:** `ReconAdaptiveBinaryAsync20260721`  
> **Trạng thái:** TEST SUITE DESIGNED  

---

## 1. Danh Mục Test Cases Cho `BinaryDrillDownEngine`

| Test Case ID | Tên Kịch Bản | Mô Tả | Kết Quả Kỳ Vọng |
|:---|:---|:---|:---|
| **TC-BDD-01** | Global Match (0% Drift) | Đối soát khoảng 30 ngày (2,880 sub-windows), dữ liệu Mongo và Postgres hoàn toàn khớp. | Chỉ chạy đúng **1 query Hash Global**, trả về `nil` (Pruning thành công 100%), thời gian $< 2\text{s}$. |
| **TC-BDD-02** | Single Drift Window | Dữ liệu 30 ngày chỉ bị lệch 1 record duy nhất ở ngày thứ 15. | Đệ quy cắt đôi $\approx 12$ bước, định vị chính xác đúng 1 cửa sổ lá 15 phút bị drift. |
| **TC-BDD-03** | Multiple Scattered Drifts | Dữ liệu bị lệch ở 3 cửa sổ rải rác khác nhau trong tháng. | Đệ quy cắt tỉa các nhánh sạch, trả về đúng 3 `DriftWindow` bị lỗi. |
| **TC-BDD-04** | Empty Range (0 records) | Đối soát khoảng thời gian không có giao dịch nào phát sinh. | Trả về `nil` ngay ở bước Hash/Count check đầu tiên. |
| **TC-BDD-05** | Max Depth Boundary Check | Ép đệ quy chạm `maxDepth` (VD: maxDepth = 3). | Dừng đệ quy tại depth = 3 và trả về cửa sổ ở mức độ đó, không bị văng stack overflow. |

---

## 2. Danh Mục Test Cases Cho Async Stateful Job Workflow

| Test Case ID | Tên Kịch Bản | Mô Tả | Kết Quả Kỳ Vọng |
|:---|:---|:---|:---|
| **TC-JOB-01** | Async Trigger & Poll | Trigger `POST /api/reconciliation/check-async`. | Trả về HTTP `202 Accepted` + `job_id` trong $< 50\text{ms}$. Record `recon_jobs` có status `PENDING`. |
| **TC-JOB-02** | Worker Processing & Progress | Background Worker tiêu thụ NATS message. | Job chuyển sang `RUNNING`, `progress_percent` tăng dần từ 0% lên 100%. |
| **TC-JOB-03** | Job Completion State | Worker hoàn tất thuật toán Bisection. | Job chuyển sang `COMPLETED`, `result_summary` chứa danh sách `DriftWindow`. |
| **TC-JOB-04** | Error Recovery & Failure State | DB bị đứt kết nối giữa chừng. | Job cập nhật `status = FAILED`, ghi nhận `error_message` rõ ràng. |
