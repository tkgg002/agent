tier: T2

# 01. Requirements — Kiểm tra và xử lý lỗi Confluence Scheduler 403 (`confluence_403`)

## 1. Bối cảnh & Triệu chứng từ User
User cung cấp các dòng log lỗi từ scheduler lúc 06:00 sáng các ngày 16/10, 17/10, 18/10, 19/10/2026:
```text
06:00:02 19/10/2026	confluence	cdc_transaction_history	error	-	-	scheduler	-	check existing confluence page failed: search content returned status 403
06:00:03 18/10/2026	confluence	cdc_transaction_history	error	-	-	scheduler	-	check existing confluence page failed: search content returned status 403
06:00:03 17/10/2026	confluence	cdc_transaction_history	error	-	-	scheduler	-	check existing confluence page failed: search content returned status 403
06:00:02 16/10/2026	confluence	cdc_transaction_history	error	-	-	scheduler	-	check existing confluence page failed: search content returned status 403
```
Yêu cầu: Kiểm tra lại nguyên nhân và đề xuất giải pháp khắc phục triệt để.

## 2. Mục tiêu
1. **Điều tra thực chứng (S2):**
   - Truy vết mã nguồn `confluence` runner / client trong `centralized-data-service` (nơi sinh ra lỗi `check existing confluence page failed: search content returned status 403`).
   - Kiểm tra cấu hình schedule trong DB (`worker_schedules` / `cdc_activity_log`) của job `confluence` cho `cdc_transaction_history` (lúc `06:00:00`).
   - Kiểm tra thực tế request HTTP tới Confluence API (URL, Auth header / Token / Email / Cookie / Cloudflare / Atlassian API) trả về HTTP `403` với response body cụ thể là gì.
2. **Thiết kế giải pháp (S3) & Kiểm định Decision Gate (`DG1`/`DG2`):**
   - Xác định rõ Root Cause (`VERIFIED`) kèm `file:line` và runtime output.
   - Đề xuất 1 giải pháp tối ưu nhất trong `09_tasks_solution_confluence_403.md`, chạy `laya_gate.py` (`DG1`, `DG2`) và trình User duyệt (`APPROVE`).
