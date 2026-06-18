# 08_tasks

- [x] T1. Backup recon_core.go, worker_server.go, metadata_registry_service.go (*.bak)
- [x] T2. recon_core.go: thêm `ReapStaleRuns(ctx) (int64, error)` (global cancel stale>15')
- [x] T3. worker_server.go: gọi ReapStaleRuns lúc startup (Start())
- [x] T4. worker_server.go: gọi ReapStaleRuns đầu mỗi chu kỳ reconcile
- [x] T5. metadata_registry_service.go: chỉ resolve source-URI cho connection có nguồn (referencedConnIDs)
- [x] T6. `go build ./...` = 0
- [x] T7. Build + restart worker binary; startup reaper dọn 7 row → verify count(running&stale)=0
- [x] T8. Verify log mới: hết warn default_master; hết 23505 beginRun failed
- [x] T9. Ghi 03_implementation + 06_validation + append 05_progress
- [x] T10. Pre-flight Rule 14 (files vật lý), Rule 16 G1–G8
