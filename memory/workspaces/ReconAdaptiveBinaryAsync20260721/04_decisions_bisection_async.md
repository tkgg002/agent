# 04 — Quyết Định Kiến Trúc (ADR): Refactor Adaptive Binary & Async Job

> **Workspace:** `ReconAdaptiveBinaryAsync20260721`  
> **Trạng thái:** ACCEPTED ADRs  

---

## ADR-001: Sử dụng Thuật Toán Merkle Tree Bisection (Chia Để Trị) Thay Cho Linear Window Scanning

* **Bối cảnh:** Khi quét đối soát dữ liệu trong khoảng thời gian dài (như 30 ngày), luồng cũ cắt cố định 2,880 cửa sổ 15 phút và quét tuyến tính $O(N)$. Điều này làm treo DB và gây lãng phí tài nguyên vì $99.9\%$ dữ liệu là hoàn toàn khớp.
* **Quyết định:** Sử dụng thuật toán đệ quy cắt đôi (Bisection) dạng Cây Hash (Merkle Tree). Cắt đôi $[start, end] \rightarrow [start, mid] + [mid, end]$. Nếu Hash khớp $\rightarrow$ Prune (Bỏ qua nhánh).
* **Hệ quả:** 
  - Khớp 100%: Số câu lệnh DB giảm từ 2,880 xuống **1 câu lệnh**.
  - Có lỗi lẻ tóm được trong $\log_2(N)$ bước.

---

## ADR-002: Chuyển Đổi API Check Từ Synchronous HTTP Sang Asynchronous Stateful Job (NATS Queue)

* **Bối cảnh:** Các lệnh check thời gian dài dính HTTP Gateway Timeout (504) và NATS Request Timeout khi đứng chờ đồng bộ.
* **Quyết định:** Sử dụng Pattern **Async Stateful Job**:
  - API `POST /api/reconciliation/check-async` ghi DB `recon_jobs` (PENDING), pub NATS Event, trả HTTP `202 Accepted` ngay lập tức.
  - Worker nền thực thi async, cập nhật `progress_percent` và `checkpoint_ts`.
  - Client Polling `GET /api/reconciliation/jobs/:job_id`.
* **Hệ quả:** Triệt tiêu hoàn toàn HTTP Timeout, giao diện FE phản hồi mượt mà với Progress Bar.
