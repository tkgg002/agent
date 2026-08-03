# 12 — Kế Hoạch Triển Khai: Chuẩn Hóa Mốc Thời Gian Linh Hoạt & Đồng Bộ CMS Reconciliation Report

> **Workspace:** `ReconAdaptiveBinaryAsync20260721`  
> **Ngày thực hiện:** 2026-07-21  
> **Role thực hiện:** Brain (Chairman & Architect)  
> **Mục tiêu:** Hỗ trợ mốc thời gian không 00:00:00 (2h, 7d, Custom, Lag 120s) & Đồng bộ CMS Report (`cdc_reconciliation_report`)  

---

## I. TỔNG QUAN GIẢI PHÁP (OVERVIEW)

Mục tiêu đợt nâng cấp này:
1. **Chuẩn Hóa Mốc Thời Gian Linh Hoạt (Flexibility & Lag Buffer 120s)**:
   - Hỗ trợ các preset đầu vào: `2h` (Hot mode), `7d` / `7ng` (Cold mode), hoặc Custom Range (`start_time`, `end_time` epoch millis).
   - Tự động áp dụng **Lag Buffer 120s** (`upper = now - 120s`) khi `end_time` không được truyền chủ động, triệt tiêu race condition ghi dữ liệu realtime.
   - `ChunkStreamBucketEngine` xử lý chính xác dải thời gian $[startTime, endTime)$ bất kỳ (không tự ý làm tròn về `00:00:00Z`).
2. **Đồng Bộ Dữ Liệu CMS Dashboard (`cdc_system.cdc_reconciliation_report`)**:
   - Khi `ReconJobWorker` (Async Job Path) hoặc `CheckHandler` (Sync Fast-Path) hoàn thành đối soát, ngoài việc cập nhật `cdc_system.recon_jobs`, hệ thống tự động ghi 1 bản ghi tổng hợp sang bảng `cdc_system.cdc_reconciliation_report`.
   - Giúp giao diện CMS UI hiện tại lập tức hiển thị trạng thái `MATCHED`/`MISMATCHED`, số lượng `diff`, `recon_start_time`, và `recon_end_time` chính xác.

---

## II. CHI TIẾT CÁC THAY ĐỔI CẦN THỰC HIỆN

### 1. Control Plane Handler Time Range Resolver (`recon_check_handler.go`)
- Nâng cấp `resolveTimeRange`:
  - Parse linh hoạt `payload.Lookback`: `"2h"`, `"7d"`, `"7ng"`, hoặc custom duration.
  - Khóa mốc `upper = now - 120s` khi `EndTime` không được truyền.
  - Parse `lower = upper - lookback` hoặc từ `StartTime`.

### 2. Stream Engine Non-00:00:00 Alignment (`recon_stream_bucket_engine.go`)
- Bắt đầu duyệt từ `currStart := startTime` chính xác.
- Chia 96 sub-windows 15m tính từ `currStart` và cắt biên `subEnd = min(subStart + 15m, currEnd)`.
- Bảo đảm 100% từng millisecond trong $[startTime, endTime)$ được bao phủ.

### 3. CMS Report Sync in Worker & Handler (`recon_job_worker.go`)
- Inject `ReportRepository` vào `ReconJobWorker`.
- Khi job `COMPLETED`, lưu 1 bản ghi `ReconciliationReport` vào `cdc_system.cdc_reconciliation_report`:
  - `RunID`: `jobID`
  - `TargetTable`: `table`
  - `Segment`: `"source_shadow"`
  - `CheckType`: `"chunk_stream_bucket"`
  - `Status`: `"MATCHED"` / `"MISMATCHED"`
  - `Diff`: `totalDiff`
  - `ReconStartTime`: `startTime`
  - `ReconEndTime`: `endTime`
  - `CheckedAt`: `time.Now().UTC()`

---

## III. BẢNG VERIFICATION PLAN

- Unit Test Presets (`2h`, `7d`, `7ng`, custom timestamps + lag 120s).
- Unit Test Stream Engine với mốc thời gian lẻ (`11:37:52Z` đến `13:37:52Z`).
- Unit Test `ReconJobWorker` sync dữ liệu sang `ReconciliationReport`.
- Verification command: `go test -v ./internal/service/recon/...` & `go test -v ./internal/handler/recon/...`.
