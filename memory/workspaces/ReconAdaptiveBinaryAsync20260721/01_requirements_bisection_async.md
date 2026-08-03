# 01 — Yêu Cầu Kỹ Thuật: Refactor Adaptive Binary & Async Job

> **Workspace:** `ReconAdaptiveBinaryAsync20260721`  
> **Trạng thái:** SPECIFICATION COMPLETE  

---

## 1. Yêu Cầu Chức Năng (Functional Requirements)

### FR-01: Thuật Toán Cắt Đôi Tự Điều Chỉnh (Adaptive Binary Drill-Down)
- **FR-01.1:** Đệ quy cắt đôi dải thời gian $[start, end]$ thành $[start, mid]$ và $[mid, end]$.
- **FR-01.2:** Lấy Hash & Count từ Source (MongoDB) và Dest (Postgres) đồng thời.
- **FR-01.3 (Pruning):** Nếu `SrcHash == DstHash` hoặc cả hai bên `Count == 0`, dừng đệ quy tại nhánh đó và trả về `nil`.
- **FR-01.4 (Base Case):** Dừng đệ quy khi `(end - start) <= minWindowDuration` (mặc định 15 phút) hoặc `currentDepth >= maxDepth` (mặc định 12). Cửa sổ lá này được ghi nhận là `DriftWindow`.
- **FR-01.5 (Parallel Execution):** Sử dụng `errgroup` để thực thi 2 nhánh con song song với giới hạn worker pool.

### FR-02: Quản Lý Tiến Trình Đối Soát Bất Đồng Bộ (Async Stateful Job)
- **FR-02.1:** API `POST /api/reconciliation/check-async` tạo Job `recon_jobs` với trạng thái `PENDING`, đăng bài tin NATS, và trả về HTTP `202 Accepted` kèm `job_id` trong $< 50\text{ms}$.
- **FR-02.2:** Background Worker tiêu thụ tin nhắn NATS, chuyển Job sang `RUNNING`, chạy `AdaptiveBinaryDrillDownEngine`.
- **FR-02.3 (Checkpointing):** Lưu mốc `checkpoint_ts` và `progress_percent` vào DB sau khi xử lý xong từng nhánh lớn.
- **FR-02.4:** Khi hoàn thành, chuyển trạng thái Job sang `COMPLETED` (hoặc `FAILED` nếu có sự cố), lưu `result_summary` dạng JSONB.
- **FR-02.5:** API `GET /api/reconciliation/jobs/{job_id}` cho phép Client query trạng thái và kết quả.

---

## 2. Tiêu Chí Nghiệm Thu (Definition of Done — Gate Checklist)

- **(G1) Requirements Traceability:** Mọi FR-01 và FR-02 đều có Unit Test & Integration Test tương ứng.
- **(G2) Performance Benchmark:** 
  - Khi dữ liệu 30 ngày khớp 100%: Số lượng DB Query $= 1$, tổng thời gian execution $< 3\text{s}$.
  - Khi có 1 lỗi duy nhất trong 30 ngày: Số lượng DB Query $\le 25$, thời gian execution $< 5\text{s}$.
- **(G3) No Race Condition:** Mọi thao tác cập nhật state job / result summary được bảo vệ an toàn.
- **(G4) 100% Pass Unit Test Suite:** Lệnh `go test -v ./internal/service/recon/...` trôi qua 100%.
