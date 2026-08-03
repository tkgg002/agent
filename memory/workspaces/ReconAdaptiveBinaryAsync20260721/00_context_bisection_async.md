# 00 — Bối Cảnh & Phạm Vi: Refactor Recon Big Data (Adaptive Binary & Async Job)

> **Workspace:** `ReconAdaptiveBinaryAsync20260721`  
> **Ngày khởi tạo:** 2026-07-21  
> **Trạng thái:** DRAFTING ARCHITECTURE & WORKLOAD PLAN  

---

## 1. Bối Cảnh Hệ Thống (System Context)

Hệ thống Đối soát (Reconciliation Subsystem) thuộc `centralized-data-service` hiện tại vận hành đối soát giữa Nguồn dữ liệu (MongoDB) và Đích dữ liệu (PostgreSQL Shadow DB).

### Những Hạn Chế Cực Kỳ Chí Mạng Hiện Tại:
1. **Quét Tuyến Tính Cố Định (Fixed Linear Brute-Force):** 
   - Khi quét khoảng thời gian lớn (ví dụ: 30 ngày lookback window), hệ thống cắt cứng thành $2,880$ cửa sổ nhỏ (mỗi cửa sổ 15 phút).
   - Hệ thống bắt buộc phải thực thi $2,880$ câu lệnh truy vấn Hash & Count xuống cả Mongo và Postgres dù cho $99.9\%$ dữ liệu hoàn toàn trùng khớp không có lỗi.
   - Gây quá tải CPU/Network/Database IOPS và thời gian phản hồi kéo dài từ vài phút đến hàng chục phút.

2. **Khái Niệm Đồng Bộ HTTP (Synchronous Request Pattern):**
   - API `POST /api/reconciliation/check` xử lý đồng bộ trong 1 HTTP Request.
   - NGINX Gateway, API Gateway hoặc NATS Client Timeout (mặc định 10s - 60s) lập tức ngắt kết nối (`504 Gateway Timeout` / `NATS Timeout`), khiến người dùng mất dấu vết tiến trình đang chạy ngầm.

---

## 2. Mục Tiêu & Phạm Vi Refactor (Scope & Goals)

### Mục Tiêu Đạt Được:
- **Đại Phẫu 1 (Adaptive Binary Drill-Down - Chia Để Trị):**
  - Áp dụng thuật toán Merkle Tree Hash Bisection: Phân nhánh cắt đôi $[start, end] \rightarrow [start, mid] + [mid, end]$.
  - Nếu Hash khớp $\rightarrow$ Pruning (Cắt tỉa nhánh) lập tức mà không cần truy vấn sâu.
  - Khi dữ liệu 100% khớp, chỉ tốn **đúng 1 query Global Hash**.
  - Giảm thiểu $\mathbf{95\% - 99.6\%}$ số lượng query vô nghĩa xuống Database.

- **Đại Phẫu 2 (Async Stateful Batch Job & Checkpointing):**
  - Chuyển đổi API sang Mô hình Bất đồng bộ Event-Driven.
  - Client gửi Request $\rightarrow$ API ghi bảng `recon_jobs` (State: `PENDING`), bắn event NATS, và phản hồi HTTP `202 Accepted` kèm `job_id` trong **< 50ms**.
  - Background Worker nhận tin nhắn, thực thi `AdaptiveBinaryDrillDown`, lưu `checkpoint_ts` định kỳ vào DB.
  - Frontend Polling / Websocket theo dõi `progress_percent` và `status`.

---

## 3. Các Thành Phần Liên Quan (Components Touched)

1. **Control Plane & API Layer:** `internal/handler/recon_handler.go`, `internal/router/router.go`
2. **Core Engine:** `internal/service/recon/recon_bisection_engine.go` (Mới), `internal/service/recon/recon_job_worker.go` (Mới)
3. **Database Repository & Schema:** Bảng `cdc_system.recon_jobs`, `internal/repository/recon_job_repo.go` (Mới)
4. **NATS Messaging Event Bus:** Message subject `cdc.event.recon.job_created`
