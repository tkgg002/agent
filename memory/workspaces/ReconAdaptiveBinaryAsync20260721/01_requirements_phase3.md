# 01 — Yêu Cầu Kỹ Thuật Phase 3: Single Adaptive Endpoint & Job Polling Handler

> **Workspace:** `ReconAdaptiveBinaryAsync20260721`  
> **Phase:** 3 — Control Plane API Integration  
> **Trạng thái:** SPECIFICATION COMPLETE  

---

## 1. Yêu Cầu Chức Năng Phase 3 (Functional Requirements)

### FR-P3-01: Single Adaptive Endpoint cho `POST /api/reconciliation/check`
- **FR-P3-01.1 (No Duplicate Routes):** KHÔNG tạo route `-async` mới để tránh rác API Contract. Giữ nguyên 1 Endpoint duy nhất `POST /api/reconciliation/check`.
- **FR-P3-01.2 (Sync Fast-path):** Nếu khoảng thời gian đối soát $\le 2\text{ giờ}$ (hoặc Hot Mode), Handler gọi trực tiếp `BinaryDrillDownEngine` và phản hồi HTTP `200 OK` + Data trong $< 300\text{ms}$.
- **FR-P3-01.3 (Async Job Path):** Nếu khoảng thời gian đối soát $> 2\text{ giờ}$ (ví dụ Big Data 30 ngày), Handler tự động khởi tạo record `recon_jobs` (Trạng thái `PENDING`), publish NATS Event `cdc.event.recon.job_created`, và phản hồi HTTP `202 Accepted` kèm `job_id` và `status_url` trong $< 50\text{ms}$.
- **FR-P3-01.4 (Trace Preservation):** Giữ nguyên TraceID trong OpenTelemetry Header và đính kèm vào `recon_jobs`.

### FR-P3-02: Endpoint Query Tiến Độ `GET /api/reconciliation/jobs/:job_id`
- **FR-P3-02.1:** Cho phép Client (CMS UI / Frontend) polling trạng thái Job.
- **FR-P3-02.2:** Trả về JSON chứa: `job_id`, `target_table`, `status` (`PENDING`, `RUNNING`, `COMPLETED`, `FAILED`), `progress_percent` ($0 - 100\%$), `total_diff_count`, `checkpoint_ts`, `result_summary` (khi `COMPLETED`), và `error_message` (khi `FAILED`).

---

## 2. DoD Quality Gates Cho Phase 3

- **(G1) No Breaking Changes:** Mọi call site cũ gửi đến `POST /api/reconciliation/check` đều trôi qua thành công.
- **(G2) 100% Pass Unit Tests:** Handlers có unit tests mô phỏng cả 2 nhánh Sync Fast-path và Async Job Path.
