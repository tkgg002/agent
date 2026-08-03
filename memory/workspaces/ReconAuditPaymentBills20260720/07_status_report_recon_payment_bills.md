# 07 — Báo Cáo Hiện Trạng & Giải Pháp Tổng Thể: Recon `payment_bills`

> **Cập nhật:** 2026-07-21 | **Workspace:** `ReconAuditPaymentBills20260720`  
> **Trạng thái:** ✅ CODE FIX COMPLETE & TEST PASSED — ⏳ PENDING PROD DEPLOYMENT & INDEX CREATION

---

## 1. Tóm Tắt Hiện Trạng Vấn Đề
Reconciliation check cho bảng `payment_bills` bị kéo dài **~90s** và báo **false drift trên cả 8/8 sub-windows** (dù `diff = 0` dữ liệu hoàn toàn khớp).

### Nguyên nhân gốc rễ:
1. **Lệch múi giờ 7 tiếng (P1):** Cột `lastUpdatedAt` thuộc kiểu `TIMESTAMPTZ`. Hàm parse Postgres timestamp cũ tự động cộng/trừ 7 tiếng múi giờ local ICT vào bản ghi `TIMESTAMPTZ` vốn đã chuẩn UTC $\rightarrow$ làm hỏng XOR hash của Postgres.
2. **Thiếu Index MongoDB (P2+P3):** Collection MongoDB `payment_bills` thiếu index `{ lastUpdatedAt: 1 }`, dẫn tới COLLSCAN mất 42.4s mỗi lần drill down.

---

## 2. Giải Pháp Tổng Thể Đã Thực Hiện
Đã bổ sung **Adaptive Schema-Aware Parsing**:
- Truy vấn `information_schema.columns` để nhận biết kiểu cột `TIMESTAMPTZ` hay `TIMESTAMP`.
- Cache kết quả kiểu cột bằng thread-safe `sync.RWMutex` map trong `ReconDestAgent` (không làm giảm hiệu năng, chỉ tốn 1 query đầu tiên).
- Nếu cột là `TIMESTAMPTZ`: Giữ nguyên UTC do driver `pgx` đọc ra, không áp dụng timezone shift.
- Nếu cột là `TIMESTAMP`: Giữ nguyên logic cũ.

---

## 3. Các Tệp Tin Đã Chỉnh Sửa
- `internal/service/recon/recon_dest_agent.go`
- `internal/service/recon/recon_dest_query.go`
- `internal/service/recon/recon_dest_hash.go`
- `internal/service/recon/recon_query.go`
- `internal/service/recon/recon_dest_agent_test.go`
- `internal/service/recon/recon_tier_a_test.go`
- `internal/service/recon/recon_smoke_test.go`

---

## 4. Kết Quả Testing
- `go test -v ./internal/service/recon/...` $\rightarrow$ **PASS 100% (0.699s)**.

---

## 5. Tài Liệu Workspace Đầy Đủ
1. **[00_context_recon_architecture.md](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/ReconAuditPaymentBills20260720/00_context_recon_architecture.md) (Tài liệu Kiến trúc & Luồng vận hành Tổng thể)**
2. [01_requirements_audit.md](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/ReconAuditPaymentBills20260720/01_requirements_audit.md)
3. [01_requirements_timezone_drift.md](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/ReconAuditPaymentBills20260720/01_requirements_timezone_drift.md)
4. [05_progress.md](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/ReconAuditPaymentBills20260720/05_progress.md)
5. [07_status_report_recon_payment_bills.md](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/ReconAuditPaymentBills20260720/07_status_report_recon_payment_bills.md) (file này)
6. [08_tasks_audit.md](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/ReconAuditPaymentBills20260720/08_tasks_audit.md)
7. [08_tasks_timezone_drift.md](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/ReconAuditPaymentBills20260720/08_tasks_timezone_drift.md)
8. [11_report_timezone_drift.md](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/ReconAuditPaymentBills20260720/11_report_timezone_drift.md)
9. [12_implementation_plan_timezone_drift.md](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/ReconAuditPaymentBills20260720/12_implementation_plan_timezone_drift.md)
10. [13_analysis_timezone_drift.md](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/ReconAuditPaymentBills20260720/13_analysis_timezone_drift.md)
