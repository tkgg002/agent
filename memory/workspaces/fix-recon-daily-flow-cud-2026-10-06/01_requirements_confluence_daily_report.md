# Yêu Cầu Kỹ Thuật: Báo Cáo Đối Soát Dữ Liệu Tự Động Hàng Ngày Trên Confluence Là Child Của Trang Core Trans Proxy History Service

- **Ngày khởi tạo:** 2026-10-12
- **Tác giả:** Brain (Architect)
- **Trang Confluence Mẹ (Parent):**
  - **URL:** `https://goopay.atlassian.net/wiki/spaces/~712020a3dcebcbe93646fea463f119dd38b08c/pages/463833042/Core+Trans+Proxy+History+Service`
  - **Base URL:** `https://goopay.atlassian.net`
  - **Space Key:** `~712020a3dcebcbe93646fea463f119dd38b08c`
  - **Parent Page ID:** `463833042`
  - **Parent Title:** `Core Trans Proxy History Service`

---

## 1. Yêu Cầu Chức Năng (Functional Requirements)

1. **R1 - Tạo trang con tự động (Child Page Creation):**
   - Mọi báo cáo kiểm tra dữ liệu ngày (Daily Data Check / Reconciliation Report) được tạo ra phải là **trang con trực tiếp** (Child Page) của trang `Core Trans Proxy History Service` (ID: `463833042`).
   - Tiêu đề trang: `Daily Data Check & Reconciliation Report - YYYY-MM-DD` (Ví dụ: `Daily Data Check & Reconciliation Report - 2026-10-12`).

2. **R2 - Nội dung báo cáo chuẩn hóa (Standardized Report Content):**
   - **Executive Summary:** Ngày chốt, trạng thái tổng thể (🟢 HEALTHY / 🔴 DRIFT), tổng pipeline, tỷ lệ toàn vẹn.
   - **Data Flow Matrix:** Bảng tổng hợp CUD từng pipeline (Kafka In, Shadow Written, Master Inserted, Master Updated, Master Skipped). Tận dụng query `daily_flow.go` đã làm ở Phase 2.
   - **Reconciliation Status:** Kết quả đối soát L1/L2/L3 (Missing, Stale, Diff, Healed) từ `cdc_reconciliation_report`.
   - **Drift Details:** Liệt kê ID bị lệch (nếu có).
   - **Action Checklist:** Checklist giao việc cho On-call Engineer nếu phát hiện lỗi.

3. **R3 - Cơ chế Idempotent (Chống trùng lặp trang):**
   - Khi re-run hoặc chạy lại báo cáo của cùng 1 ngày, hệ thống tìm trang đã có dưới parent `463833042` và cập nhật (`PUT /wiki/rest/api/content/{id}` kèm `version + 1`), không sinh trang con trùng lặp.

4. **R4 - Hai cơ chế kích hoạt (Trigger Mechanism):**
   - **Scheduled Daily Cron:** Tự động chạy mỗi sáng (06:00 AM) để chốt số liệu ngày hôm trước D-1.
   - **Manual On-Demand Trigger:** Endpoint API `POST /api/reconciliation/reports/confluence/publish?date=YYYY-MM-DD` cho phép xuất báo cáo ngay lập tức cho bất kỳ ngày nào.

---

## 2. Yêu Cầu Phi Chức Năng & Bảo Mật (Non-Functional Requirements)

1. **Bảo mật API Token:**
   - Không hardcode API Token trong source code.
   - Đọc từ biến môi trường `CONFLUENCE_API_TOKEN` hoặc file cấu hình bí mật.
2. **Khả năng chịu lỗi (Resilience):**
   - Timeout khi gọi Confluence API là 30 giây.
   - Nếu Confluence API tạm thời gián đoạn (5xx hoặc Rate Limit 429), không làm crash service CMS. Ghi log cảnh báo và retry với exponential backoff.
3. **Hiệu năng & Tối ưu:**
   - Sử dụng Confluence Storage Format (XHTML tối giản) nhẹ, tải nhanh, tương thích hoàn hảo với Confluence Cloud editor.
