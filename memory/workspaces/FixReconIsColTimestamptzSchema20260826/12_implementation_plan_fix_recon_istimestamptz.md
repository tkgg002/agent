# 12_implementation_plan_fix_recon_istimestamptz.md: Kế hoạch Triển khai Chi tiết của AI

## Phase 1: Phân tích & Tracing
- Tracing commit `f80b03e` (hôm qua) phát hiện điểm hổng khi `entry.ShadowSchema` rỗng làm `QualifiedTarget()` chỉ trả tên bảng trần.
- Lập bộ tài liệu workspace tại `FixReconIsColTimestamptzSchema20260826`.

## Phase 2: Triển khai Code (Muscle Role)
1. Edit `internal/service/recon/recon_stream_bucket_engine.go`: Phân giải `ShadowSchema` nếu rỗng.
2. Edit `internal/service/recon/recon_job_worker.go`: Truyền `entryCopy` có `ShadowSchema`.

## Phase 3: Kiểm thử & Phê duyệt
1. Chạy `go test -v ./internal/service/recon/...` $\rightarrow$ PASS 100%.
2. Audit DoD Gates G1 - G8.
3. Chạy `python3 agent/tooling/verify_governance.py`.
