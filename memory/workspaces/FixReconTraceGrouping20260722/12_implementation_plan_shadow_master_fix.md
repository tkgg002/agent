# 12 Implementation Plan: Fix Recon Shadow-Master Flow

## 1. Các file sẽ chỉnh sửa
- **`internal/service/recon/recon_job_worker.go`**: Bổ sung `Segment` vào `ReconJobCreatedEvent`, truyền `event.Segment` sang `w.engine.ExecuteSegment`.
- **`internal/handler/recon/recon_check_handler.go`**: Truyền `Segment: payload.Segment` trong `HandleReconCheck` khi publish NATS message.
- **`internal/service/recon/recon_stream_bucket_engine.go`**:
  - Thêm `masterAgent` và `db`.
  - Thêm `ExecuteSegment(ctx, entry, segment, startTime, endTime)`.
  - Thêm logic đối soát Segment B (`shadow_master`) giữa `destAgent` (Shadow) và `masterAgent` (Master) dùng `MasterBindingRef`.
- **`internal/server/server_setup.go`**: Wire `masterAgent` vào `chunkEngine`.

## 2. Kế hoạch Verification
- Chạy unit test trong package `recon` và `handler/recon`: `go test ./internal/...`.
