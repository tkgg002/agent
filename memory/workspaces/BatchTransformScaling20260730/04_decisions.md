# 04_decisions.md — Nhật ký Quyết định Kiến trúc (ADRs)
# BatchTransformScaling20260730

---

## ADR-001: Dùng bảng riêng `cdc_system.transform_jobs` thay vì tái dùng `recon_jobs`

**Ngày:** 2026-07-30  
**Trạng thái:** APPROVED (user comment: "thằng recon_jobs như nào thì làm theo như vậy. viết migration ở cdc-cms-service nhé" → sau đó user update: "sao qua recon_job. liên quan gì nhau phải là transform_jobs chứ")

**Bối cảnh:**
- Plan ban đầu (`01_requirements.md` FR2) đề xuất tái dùng `recon_jobs` + thêm cột `job_type`.
- User phát hiện và yêu cầu đổi: Transform và Recon là 2 nghiệp vụ hoàn toàn khác nhau.

**Quyết định:**
- Tạo bảng mới `cdc_system.transform_jobs` với schema riêng.
- KHÔNG modify `recon_jobs`.
- Migration: `migrations/schema/recon_dlq/100_create_transform_jobs.sql`.

**Hệ quả:**
- `TransformJobRepo` là repo riêng biệt, không tái dùng `ReconJobRepo`.
- Cần tạo cả Worker repo lẫn CMS repo (cùng table, khác package).
- Trade-off: code duplication nhỏ nhưng domain isolation rõ ràng.

---

## ADR-002: `OnPublishResult` callback không được hook vào `BatchTransformHandler` mới

**Ngày:** 2026-07-30  
**Trạng thái:** INTENTIONAL (documented decision, không phải bug)

**Bối cảnh:**
- Các handler khác (scanHandler, schemaDDLHandler) dùng `OnPublishResult` callback từ `base.BaseHandler` để route kết quả ra activity log.
- `BatchTransformHandler` mới chạy async goroutine — không thể dùng `OnPublishResult` vì callback này là sync.

**Quyết định:**
- Ghi activity log trực tiếp via `governance.NewActivityLogger` trong `runTransformJob`.
- KHÔNG set `OnPublishResult` vì handler là non-blocking; callback cũ không phù hợp với mô hình async.

**Hệ quả:**
- Activity log vẫn được ghi (COMPLETED, FAILED, CANCELLED đều có log).
- Nhưng format log có thể khác nhỏ so với các handler khác (dùng `act.Quick` trực tiếp thay vì qua callback).

---

## ADR-003: `IsCancelRequested` trả về `bool` (không phải `(bool, error)`)

**Ngày:** 2026-07-31  
**Trạng thái:** APPROVED (aligned trong cả 2 repos sau BUG-02 fix)

**Quyết định:**
- Cả Worker repo lẫn CMS repo đều dùng signature `IsCancelRequested(...) bool`.
- DB error → trả `false` (safe default: không cancel khi không chắc trạng thái).
- Lý do: trong hot-loop của 50M records, error handling phức tạp làm chậm luồng; false-negative (không cancel khi nên cancel) ít nguy hiểm hơn false-positive (cancel nhầm).

---

## ADR-004: `progress_percent` luôn = 0 trong runtime

**Ngày:** 2026-07-31  
**Trạng thái:** ACCEPTED (limitation có document)

**Bối cảnh:**
- Không thể biết tổng số records cần transform mà không chạy `SELECT COUNT(*)` upfront — tốn kém với 50M+ records.

**Quyết định:**
- `progress_percent = 0` trong suốt RUNNING.
- `progress_percent = 100` khi COMPLETED/FAILED/CANCELLED.
- UI dùng `rows_affected` (live counter) để hiển thị tiến độ dạng "X rows processed".
- Heartbeat update mỗi 50 productive iterations: `rows_affected` tăng lên, UI polling thấy được.

**Hệ quả:**
- Progress bar FE sẽ hiển thị dạng indeterminate (hoặc dùng rows_affected thay percent).
- Không thể hiển thị "30% hoàn thành" — chỉ hiện "1,234,000 rows transformed".
