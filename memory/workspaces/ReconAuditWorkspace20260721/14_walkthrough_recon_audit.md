# 14 — Báo Cáo Walkthrough Kiểm Thử Thực Tế (Walkthrough Report)

## 📌 Tổng Quan Tác Vụ
Khắc phục triệt để lỗi chênh lệch giả lập (**Phantom Reconciliation Drift — 40 sub-windows**) và sai lệch chỉ số báo cáo trong cơ sở dữ liệu `cdc_reconciliation_report` đối với pipeline `payment_bills`.

---

## 🛠️ Các Thay Đổi Đã Triển Khai

### 1. Sửa Lỗi Decode Mongo BSON Int64 (`recon_hash.go`)
- **Vấn đề**: Hàm `extractMongoIDFromRaw` trước đây rơi vào nhánh fallback `rawVal.String()` đối với BSON kiểu `Int64` (`0x12`), biến ID số nguyên (ví dụ `504`) thành chuỗi định dạng JSON `{"$numberLong":"504"}`. Điều này khiến dấu vân tay (XOR Hash fingerprint) ở MongoDB không khớp với PostgreSQL shadow.
- **Giải pháp**: Bổ sung kiểm tra `Int64OK()`, `Int32OK()`, và `DoubleOK()` trong `extractMongoIDFromRaw`, trả về đúng chuỗi số nguyên dạng sần `"504"`.

### 2. Định Danh Cột Primary Key Động (`recon_stream_bucket_engine.go`)
- **Vấn đề**: Các bảng Shadow PostgreSQL có thể sử dụng cột `_source_id` kiểu string thay vì `_id`.
- **Giải pháp**: Thêm hàm `resolvePKFields(ctx, entry)` tự động dùng `ColumnExists` để kiểm tra sự tồn tại của `_source_id` trên Shadow PostgreSQL, gán đúng `dstPK = "_source_id"` khi cần.

### 3. Chuẩn Hóa Kết Quả Trả Về Engine (`ChunkEngineResult`)
- **Vấn đề**: Engine cũ chỉ trả về `[]DriftWindow`, khiến `ReconJobWorker` dùng `len(drifts)` (số cửa sổ 15 phút bị drift = 216/215) làm giá trị `diff` và `missing_count`, gây sai lệch báo cáo chỉ số record.
- **Giải pháp**: Định nghĩa `ChunkEngineResult` chứa `TotalSrcCount`, `TotalDstCount`, `RecordDiff`, `MissingCount`, và danh sách `Drifts`.

### 4. Cập Nhật ReconJobWorker & DB Insert (`recon_job_worker.go`)
- **Vấn đề**: Bảng `cdc_reconciliation_report` nhận các giá trị `diff=216, missing_count=216` thay vì tổng số record thực tế.
- **Giải pháp**: Cập nhật worker tiêu thụ `*ChunkEngineResult`, ghi đúng `source_count`, `dest_count`, `diff`, và `missing_count` vào DB. Khi dữ liệu hoàn toàn khớp (No Drift), thiết lập `diff = 0` và `missing_count = 0`.

---

## 🧪 Kết Quả Kiểm Thử

### 1. Automated Unit Tests
```bash
go test -v ./internal/service/recon/...
```
- **Kết quả**: 100% tests PASSED trong **0.820s**.

### 2. Live Integration Verification (End-to-End via NATS & Database)
Đã phát sự kiện `ReconJobCreatedEvent` qua NATS (`cdc.event.recon.job_created`) tới `ReconJobWorker` cho pipeline `payment_bills` trên khoảng thời gian từ `2026-07-14 09:23:00` đến `2026-07-21 09:23:00`.

**Báo cáo ghi nhận thực tế trong cơ sở dữ liệu (`cdc_reconciliation_report`):**
```
ID: 68
RunID: 516967c2-a755-4968-88a8-852222d3e468
SourceCount: 1230
DestCount: 1230
Diff: 0
MissingCount: 0
Status: COMPLETED
CheckedAt: 2026-07-21 09:52:18
```

- **Số lượng bản ghi Nguồn (Mongo)**: `1230`
- **Số lượng bản ghi Đích (Postgres Shadow)**: `1230`
- **Số lượng chênh lệch (Diff)**: `0` (ZERO DRIFT 🟢)
- **Số lượng thiếu (Missing)**: `0` (ZERO MISSING 🟢)
- **Số sub-window bị drift**: `0`
- **Trạng thái Job**: `COMPLETED`
