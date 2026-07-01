# Tasks Breakdown: Fix False Drift on Recon payment_bills / Phân chia Công việc: Sửa lỗi đối soát báo khống 1.410 drift ảo trên bảng payment_bills

- [x] **Task 1: Khảo sát & Tìm Root Cause**
  - [x] Đọc logs đối soát và phân tích sự chênh lệch.
  - [x] Xác định lỗi lệch hệ quy chiếu thời gian giữa `_source_ts` ở Postgres và `lastUpdatedAt` ở MongoDB.
- [x] **Task 2: Nâng cấp `ReconDestAgent` hỗ trợ Dynamic Timestamp**
  - [x] Cập nhật signatures cho các hàm trong `recon_dest_query.go` để nhận tham số `timestampField`.
  - [x] Cập nhật signatures cho các hàm trong `recon_dest_hash.go`.
- [x] **Task 3: Triển khai quy đổi thời gian Postgres sang Epoch Milliseconds**
  - [x] Thực hiện phép tính `((EXTRACT(EPOCH FROM date_trunc('hour', "lastUpdatedAt")))::bigint * 1000)` trong câu SQL khi `timestampField` là domain timestamp.
- [x] **Task 4: Cập nhật Tier 1 & Tier 2**
  - [x] Cập nhật `recon_tier_a.go` lấy `resolvedTS` từ registry config.
  - [x] Cập nhật `recon_tier_b.go` truyền default `_source_ts`.
- [x] **Task 5: Đảm bảo tương thích ngược**
  - [x] Cập nhật file wrapper tương thích ngược `recon_dest_legacy.go`.
- [x] **Task 6: Viết Unit Tests & Thực thi kiểm thử**
  - [x] Thêm các unit test cases trong `recon_dest_agent_test.go`.
  - [x] Chạy test suite `go test -v ./internal/service/recon/...` thành công.
- [x] **Task 7: Hoàn thiện quy chuẩn Quản trị (Governance Compliance)**
  - [x] Khởi tạo `01_requirements.md`.
  - [x] Khởi tạo `03_implementation.md`.
  - [x] Khởi tạo `04_decisions.md`.
  - [x] Cập nhật `05_progress.md` dạng bảng log và RCA vi phạm quy trình.
  - [x] Khởi tạo `06_validation.md`.
  - [x] Khởi tạo `07_lessons.md`.
  - [x] Khởi tạo `report_recon_false_drift.md`.

---

## Task 8: Nâng cấp MaxWindowTs trên ReconDestAgent hỗ trợ Dynamic Timestamp
- **Phase**: GĐ2 Safety net
- **Service Group**: Utilities / Core Recon
- **Service(s)**: centralized-data-service / recon
- **Mô tả**: Nâng cấp signature và logic của `ReconDestAgent.MaxWindowTs` để nhận `timestampField` và query động, khắc phục lỗi tính lag ảo và trip circuit breaker. Cập nhật các hàm gọi trong `recon_tier_a.go`, `recon_tier_b.go`, `recon_smoke.go`, và bổ sung Unit Test tương ứng.
- **Trạng thái**: [x] DONE (đã thực hiện)

### [Context]
- Current state: `ReconDestAgent.MaxWindowTs` hiện tại đang bị fix cứng query cột `_source_ts`. Khi source agent dùng `lastUpdatedAt`, hàm lag sẽ lệch hệ quy chiếu.
- Dependencies: `recon_dest_query.go`, `recon_tier_a.go`, `recon_tier_b.go`, `recon_smoke.go`, `recon_dest_agent_test.go`
- ADR liên quan: N/A
- Logs/Error: N/A (Lag ảo tính ra ~148 ngày, trip circuit breaker)

### [Definition of Done]
- [x] Hàm `MaxWindowTs` nhận `timestampField` và query động (dùng `MAX(column)`) khi `timestampField` không phải `_source_ts`/empty.
- [x] Cập nhật tất cả các cuộc gọi trong `recon_tier_a.go`, `recon_tier_b.go`, và `recon_smoke.go`.
- [x] Bổ sung ít nhất 2 test cases (`TestDestAgent_MaxWindowTs_Default` và `TestDestAgent_MaxWindowTs_DomainTS`) trong `recon_dest_agent_test.go`.
- [x] **[QA Gate]**: Chạy `go test -v ./internal/service/recon/...` thành công 100%.
- [x] **[Security Gate]**: Chạy code review và validate SQL Injection (tên cột được validate qua `validateIdent`).
- [x] Blast radius verified (chỉ ảnh hưởng tới watermark calculation trong module `recon`).
- [x] Model Tracking: Ghi nhận task và update `05_progress.md`.
