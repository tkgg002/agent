# 09 Tasks Solution: Sửa Lỗi Luồng Recon Shadow-Master Của ReconJobWorker

## 1. Root Cause Analysis (Phân tích nguyên nhân gốc rễ)
Qua trace log User cung cấp cho `POST /api/reconciliation/check` trên bảng `payment_bills`:
1. `HandleReconCheck` trong `recon_check_handler.go` khi bắn `ReconJobCreatedEvent` qua NATS **không đóng gói trường `Segment`** (`payload.Segment`).
2. `ReconJobCreatedEvent` trong `recon_job_worker.go` thiếu field `Segment`.
3. `ChunkStreamBucketEngine` trong `recon_stream_bucket_engine.go` hiện chỉ chứa `sourceAgent` (Source DB) và `destAgent` (Shadow DB), chưa được wire `masterAgent` (Master DB) và chưa có logic đối soát Segment B (`shadow_master`).
4. Khi gọi `POST /api/reconciliation/check`, `ReconJobWorker` chỉ chạy `sourceAgent.HashWindow` vs `destAgent.HashWindow` (Segment A). Do Source DB và Shadow DB khớp nhau, job trả về `COMPLETED` với diff = 0, hoàn toàn bỏ qua việc đối soát chênh lệch giữa Shadow DB và Master DB!

## 2. Technical Solution (Giải pháp kỹ thuật)

### A. Cập nhật `ReconJobCreatedEvent` & `recon_check_handler.go`
- Bổ sung `Segment string json:"segment"` vào `ReconJobCreatedEvent`.
- Trong `HandleReconCheck`, truyền `Segment: payload.Segment` vào `eventPayload`.

### B. Mở rộng `ChunkStreamBucketEngine` hỗ trợ `masterAgent` & Segment B
- Thêm field `masterAgent *ReconDestAgent` và `db *gorm.DB` vào `ChunkStreamBucketEngine`.
- Thêm phương thức `WithMasterAgent(master *ReconDestAgent, db *gorm.DB) *ChunkStreamBucketEngine`.
- Wire `masterAgent` vào `chunkEngine` tại `server_setup.go`.
- Cập nhật `Execute`: Hỗ trợ tham số `segment string` (`"source_shadow"`, `"shadow_master"`, `"both"`):
  - Nếu `segment == "shadow_master"`: Đọc `MasterBindingRef` của `TargetTable`, chạy `checkDayChunkB` đối soát giữa `destAgent` (`ShadowRel`) và `masterAgent` (`MasterRel`) trên cột `_gpay_id` / `_source_ts`.
  - Nếu `segment == "both"`: Chạy cả Segment A và Segment B, gộp kết quả chênh lệch.

### C. Phân cấp OTel Trace Spans cho Segment B
- Cấu trúc Trace Spans khi đối soát `shadow_master`:
```text
cdc.recon.chunk_stream_bucket_b: payment_bills [start -> end]
 └── cdc.recon.chunk_day_01: payment_bills [start -> end]
      ├── cdc.recon.hash_window: payment_bills [HH:MM:SS -> HH:MM:SS]
      │    ├── pg.hash_window: shadow_testpbs.payment_bills [HH:MM:SS -> HH:MM:SS]
      │    ├── pg.hash_window: master_payment_bill_service.payment_bills [HH:MM:SS -> HH:MM:SS]
      │    └── cdc.recon.drift_drill_down: payment_bills [HH:MM:SS -> HH:MM:SS]  (nếu lệch)
      │         ├── pg.diff_idts: shadow_testpbs.payment_bills [HH:MM:SS -> HH:MM:SS]
      │         └── pg.diff_idts: master_payment_bill_service.payment_bills [HH:MM:SS -> HH:MM:SS]
```
